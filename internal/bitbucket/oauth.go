package bitbucket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultOAuthBaseURL = "https://bitbucket.org"

// loginTimeout is how long BrowserLogin waits for the user to complete
// authorization in the browser.
const loginTimeout = 5 * time.Minute

// OAuthApp is a Bitbucket OAuth 2.0 consumer used for the browser login flow
// and token refresh. The consumer must be created by a workspace admin with a
// callback URL of http://localhost:<port>/callback.
type OAuthApp struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	http         *http.Client
}

// NewOAuthApp creates an OAuth client for the given consumer credentials.
func NewOAuthApp(clientID, clientSecret string) *OAuthApp {
	return &OAuthApp{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      defaultOAuthBaseURL,
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

// TokenSet is the result of a code exchange or refresh.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// AuthorizeURL builds the browser authorization URL.
func (a *OAuthApp) AuthorizeURL(redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", a.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	return a.BaseURL + "/site/oauth2/authorize?" + q.Encode()
}

// ExchangeCode trades an authorization code for tokens.
func (a *OAuthApp) ExchangeCode(code, redirectURI string) (*TokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	return a.postToken(form)
}

// Refresh trades a refresh token for a new token set. Bitbucket rotates
// refresh tokens: when the response carries a new one, callers must persist it.
func (a *OAuthApp) Refresh(refreshToken string) (*TokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return a.postToken(form)
}

func (a *OAuthApp) postToken(form url.Values) (*TokenSet, error) {
	req, err := http.NewRequest(http.MethodPost, a.BaseURL+"/site/oauth2/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.SetBasicAuth(a.ClientID, a.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	var tr oauthTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK || tr.Error != "" {
		msg := tr.ErrorDesc
		if msg == "" {
			msg = tr.Error
		}
		if msg == "" {
			msg = fmt.Sprintf("token endpoint returned %d", resp.StatusCode)
		}
		return nil, &AuthError{Msg: "OAuth: " + msg}
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned no access token")
	}
	return &TokenSet{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
	}, nil
}

type callbackResult struct {
	code string
	err  error
}

// BrowserLogin runs the authorization-code flow: it listens on
// 127.0.0.1:port, opens the authorization URL via openBrowser, and exchanges
// the returned code. port 0 picks a free port (tests only — real consumers
// need a fixed callback URL). Progress messages go to out.
func (a *OAuthApp) BrowserLogin(port int, openBrowser func(url string) error, out io.Writer) (*TokenSet, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("cannot listen on port %d — something else is using it: %w\nFree the port and retry, or use --port N (the consumer callback URL must be registered as http://localhost:N/callback)", port, err)
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", actualPort)

	resultCh := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			desc := q.Get("error_description")
			if desc == "" {
				desc = e
			}
			http.Error(w, "Authorization failed. You can close this tab.", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("authorization denied: %s", desc)}
			return
		}
		if q.Get("state") != state {
			http.Error(w, "State mismatch. You can close this tab.", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("state mismatch in OAuth callback")}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "Missing code. You can close this tab.", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("no authorization code in callback")}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body><p>Logged in — you can close this tab and return to the terminal.</p></body></html>")
		resultCh <- callbackResult{code: code}
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	authURL := a.AuthorizeURL(redirectURI, state)
	fmt.Fprintf(out, "Opening browser for Bitbucket login...\nIf it does not open, visit:\n  %s\n", authURL)
	if openBrowser != nil {
		_ = openBrowser(authURL)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return a.ExchangeCode(res.code, redirectURI)
	case <-time.After(loginTimeout):
		return nil, fmt.Errorf("timed out waiting for browser authorization (%s)", loginTimeout)
	}
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
