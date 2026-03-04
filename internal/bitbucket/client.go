package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.bitbucket.org"

// AuthError indicates a 401 response.
type AuthError struct{ Msg string }

func (e *AuthError) Error() string { return e.Msg }

// NotFoundError indicates a 404 response.
type NotFoundError struct{ Msg string }

func (e *NotFoundError) Error() string { return e.Msg }

// Client is a Bitbucket API HTTP client with Basic Auth and retry logic.
type Client struct {
	Username string
	Password string // app password
	Verbose  bool
	http     *http.Client
}

// NewClient creates a new Bitbucket API client.
func NewClient(username, password string, verbose bool) *Client {
	return &Client{
		Username: username,
		Password: password,
		Verbose:  verbose,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// Do executes an HTTP request with retry on 429.
func (c *Client) Do(method, path string, body io.Reader) (*http.Response, error) {
	url := baseURL + path

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.SetBasicAuth(c.Username, c.Password)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if c.Verbose {
			fmt.Printf("--> %s %s\n", method, url)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		if c.Verbose {
			fmt.Printf("<-- %d %s\n", resp.StatusCode, resp.Status)
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			resp.Body.Close()
			return nil, &AuthError{Msg: "Auth failed — run `bbgo config verify`"}
		case http.StatusNotFound:
			resp.Body.Close()
			return nil, &NotFoundError{Msg: "Not found — check repo slug and PR ID"}
		case http.StatusTooManyRequests:
			resp.Body.Close()
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if c.Verbose {
				fmt.Printf("    Rate limited, waiting %v...\n", wait)
			}
			time.Sleep(wait)
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode >= 400 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Get performs a GET request and returns the response.
func (c *Client) Get(path string) (*http.Response, error) {
	return c.Do("GET", path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling body: %w", err)
	}
	return c.Do("POST", path, strings.NewReader(string(data)))
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.Do("DELETE", path, nil)
}

// GetJSON performs a GET request and decodes the JSON response into v.
func (c *Client) GetJSON(path string, v interface{}) error {
	resp, err := c.Get(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// PostJSON performs a POST and decodes the JSON response into v.
func (c *Client) PostJSON(path string, body, v interface{}) error {
	resp, err := c.Post(path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}
