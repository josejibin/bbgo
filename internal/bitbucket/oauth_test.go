package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTokenServer(t *testing.T, wantGrant string, resp map[string]any, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/site/oauth2/access_token" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-id" || pass != "test-secret" {
			t.Errorf("missing or wrong basic auth: %s/%s", user, pass)
		}
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		if got := form.Get("grant_type"); got != wantGrant {
			t.Errorf("grant_type = %q, want %q", got, wantGrant)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestAuthorizeURL(t *testing.T) {
	app := NewOAuthApp("test-id", "test-secret")
	u := app.AuthorizeURL("http://localhost:8976/callback", "abc123")

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("parsing authorize URL: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "test-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("state") != "abc123" {
		t.Errorf("state = %q", q.Get("state"))
	}
	if q.Get("redirect_uri") != "http://localhost:8976/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
}

func TestExchangeCode(t *testing.T) {
	srv := newTokenServer(t, "authorization_code", map[string]any{
		"access_token":  "at-1",
		"refresh_token": "rt-1",
		"expires_in":    7200,
	}, http.StatusOK)
	defer srv.Close()

	app := NewOAuthApp("test-id", "test-secret")
	app.BaseURL = srv.URL

	ts, err := app.ExchangeCode("the-code", "http://localhost:8976/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if ts.AccessToken != "at-1" || ts.RefreshToken != "rt-1" {
		t.Errorf("unexpected token set: %+v", ts)
	}
	if until := time.Until(ts.ExpiresAt); until < 1*time.Hour || until > 3*time.Hour {
		t.Errorf("ExpiresAt not ~2h out: %v", ts.ExpiresAt)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	srv := newTokenServer(t, "refresh_token", map[string]any{
		"access_token":  "at-2",
		"refresh_token": "rt-2",
		"expires_in":    7200,
	}, http.StatusOK)
	defer srv.Close()

	app := NewOAuthApp("test-id", "test-secret")
	app.BaseURL = srv.URL

	ts, err := app.Refresh("rt-1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if ts.AccessToken != "at-2" {
		t.Errorf("AccessToken = %q", ts.AccessToken)
	}
	if ts.RefreshToken != "rt-2" {
		t.Errorf("rotated RefreshToken = %q, want rt-2", ts.RefreshToken)
	}
}

func TestTokenErrorResponse(t *testing.T) {
	srv := newTokenServer(t, "refresh_token", map[string]any{
		"error":             "invalid_grant",
		"error_description": "refresh token revoked",
	}, http.StatusBadRequest)
	defer srv.Close()

	app := NewOAuthApp("test-id", "test-secret")
	app.BaseURL = srv.URL

	_, err := app.Refresh("rt-dead")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("error type = %T, want *AuthError", err)
	}
	if !strings.Contains(err.Error(), "refresh token revoked") {
		t.Errorf("error message %q missing description", err.Error())
	}
}

// TestBrowserLogin drives the full loopback flow: the fake "browser" follows
// the authorize URL's redirect_uri with a code, and the local server hands it
// to the token endpoint.
func TestBrowserLogin(t *testing.T) {
	tokenSrv := newTokenServer(t, "authorization_code", map[string]any{
		"access_token":  "at-login",
		"refresh_token": "rt-login",
		"expires_in":    7200,
	}, http.StatusOK)
	defer tokenSrv.Close()

	app := NewOAuthApp("test-id", "test-secret")
	app.BaseURL = tokenSrv.URL

	fakeBrowser := func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := parsed.Query()
		redirect := q.Get("redirect_uri")
		state := q.Get("state")
		go func() {
			resp, err := http.Get(fmt.Sprintf("%s?code=the-code&state=%s", redirect, state))
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	ts, err := app.BrowserLogin(0, fakeBrowser, io.Discard)
	if err != nil {
		t.Fatalf("BrowserLogin: %v", err)
	}
	if ts.AccessToken != "at-login" {
		t.Errorf("AccessToken = %q", ts.AccessToken)
	}
}

// TestBrowserLoginIgnoresBadState verifies that requests with a wrong state
// (drive-by GETs from other local processes or web pages) neither abort the
// login nor deliver a code — the legitimate callback still succeeds after.
func TestBrowserLoginIgnoresBadState(t *testing.T) {
	tokenSrv := newTokenServer(t, "authorization_code", map[string]any{
		"access_token":  "at-good",
		"refresh_token": "rt-good",
		"expires_in":    7200,
	}, http.StatusOK)
	defer tokenSrv.Close()

	app := NewOAuthApp("test-id", "test-secret")
	app.BaseURL = tokenSrv.URL

	fakeBrowser := func(authURL string) error {
		parsed, _ := url.Parse(authURL)
		q := parsed.Query()
		redirect := q.Get("redirect_uri")
		state := q.Get("state")
		go func() {
			// Drive-by requests with bad/missing state must be ignored...
			for _, bad := range []string{"?code=evil&state=wrong", "?error=x&error_description=%1b[2Jpwned"} {
				resp, err := http.Get(redirect + bad)
				if err != nil {
					t.Errorf("drive-by request failed: %v", err)
					return
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("drive-by request got %d, want 400", resp.StatusCode)
				}
			}
			// ...and the legitimate callback still wins.
			resp, err := http.Get(fmt.Sprintf("%s?code=the-code&state=%s", redirect, state))
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}

	ts, err := app.BrowserLogin(0, fakeBrowser, io.Discard)
	if err != nil {
		t.Fatalf("BrowserLogin: %v", err)
	}
	if ts.AccessToken != "at-good" {
		t.Errorf("AccessToken = %q", ts.AccessToken)
	}
}

func TestSanitizeParam(t *testing.T) {
	in := "\x1b[2Jrun this\x07 command\x00"
	if got := sanitizeParam(in); got != "[2Jrun this command" {
		t.Errorf("sanitizeParam = %q", got)
	}
	long := strings.Repeat("a", 300)
	if got := sanitizeParam(long); len(got) != 200 {
		t.Errorf("length cap = %d, want 200", len(got))
	}
}
