package sendico

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

var (
	// APISecretRegex is a regular expression that matches the API secret embedded in the Sendico NUXT JS snippet.
	APISecretRegex = regexp.MustCompile(`apiSecret:"([^"]+)"`)

	// DefaultBaseURL is the default base URL for the Sendico API.
	DefaultBaseURL = "https://sendico.com"
)

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.Client = httpClient
	}
}

func WithAPISecret(secret string) ClientOption {
	return func(c *Client) {
		c.secret = secret
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

type Client struct {
	*http.Client
	secret  string
	baseURL string
}

func New(ctx context.Context, opts ...ClientOption) (*Client, error) {
	c := &Client{
		Client:  http.DefaultClient,
		baseURL: DefaultBaseURL,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.secret == "" {
		if err := c.findAPISecret(ctx); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) req(ctx context.Context, method, path string, body io.Reader, opts ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, NewRequestError(err)
	}

	if c.secret != "" {
		req.Header.Set("Sendico-Secure", c.secret)
	}

	for _, opt := range opts {
		opt(req)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, NewRequestError(err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.ErrorContext(ctx, "unexpected status code",
			"body", string(body),
			"code", resp.StatusCode,
			"path", path,
			"method", method,
		)
		_ = resp.Body.Close()
		return nil, NewUnexpectedResponseCodeError(resp.StatusCode)
	}

	return resp, err
}

func (c *Client) findAPISecret(ctx context.Context) error {
	resp, err := c.req(ctx, http.MethodGet, "/", nil, func(req *http.Request) {
		req.Header.Set("Accept", "text/html")
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	matches := APISecretRegex.FindSubmatch(bytes)
	if len(matches) < 2 {
		return ErrSecretNotFound
	}

	c.secret = string(matches[1])
	return nil
}

// Translate translates the given text from English to Japanese.
func (c *Client) Translate(ctx context.Context, text string) (string, error) {
	request := struct {
		String string `json:"string"`
		From   string `json:"from"`
		To     string `json:"to"`
	}{
		String: text,
		From:   "en",
		To:     "ja",
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	resp, err := c.req(ctx, http.MethodPost, "/api/translate", bytes.NewReader(requestJSON), func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	response := struct {
		Code int    `json:"code"`
		Data string `json:"data"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Data, nil
}

// Search performs a search for the given term on the specified merchant. It will only return the first page of results.
// The supplied search term must be in Japanese.
func (c *Client) Search(ctx context.Context, shop Shop, termJP string) ([]Item, error) {
	path := url.URL{
		Path: fmt.Sprintf("/api/%s/items", shop.Identifier()),
	}
	q := path.Query()
	q.Set("search", termJP)
	q.Set("page", "1")
	q.Set("global", "1")
	path.RawQuery = q.Encode()

	resp, err := c.req(ctx, http.MethodGet, path.String(), nil, func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response := struct {
		Code int `json:"code"`
		Data struct {
			Items      []Item `json:"items"`
			TotalItems int    `json:"total_items"`
		} `json:"data"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Data.Items, nil
}

func (c *Client) BulkSearch(ctx context.Context, termJP string, shops ...Shop) ([]Item, error) {
	items := make([]Item, 0)
	itemsMu := sync.Mutex{}

	g := new(errgroup.Group)
	for _, shop := range shops {
		shop := shop
		g.Go(func() error {
			results, err := c.Search(ctx, shop, termJP)
			if err != nil {
				return err
			}

			itemsMu.Lock()
			defer itemsMu.Unlock()
			items = append(items, results...)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return items, nil
}
