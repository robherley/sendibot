package sendico

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultBaseURL is the default base URL for the Sendico API.
	DefaultBaseURL = "https://sendico.com"
)

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
	hmacSecret string
	baseURL    string
}

func New(ctx context.Context, opts ...ClientOption) (*Client, error) {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    DefaultBaseURL,
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.FindHMAC(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

// HMACSecret returns the HMAC secret key used to sign requests to the Sendico API.
func (c *Client) HMACSecret() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hmacSecret
}

// FindHMAC finds the HMAC secret key used to sign requests to the Sendico API. This is very jank, it will go through
// the frontend's SSR'd nuxt data and attempt to find the latest hmac secret key(s). This will most likely break at
// some point in the future.
func (c *Client) FindHMAC(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return NewRequestError(err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return NewUnexpectedResponseCodeError(res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	selection := doc.Find("script#__NUXT_DATA__")
	if selection.Length() == 0 {
		return errors.New("script tag not found")
	}

	var unstruct []any
	if err := json.Unmarshal([]byte(selection.Nodes[0].FirstChild.Data), &unstruct); err != nil {
		return err
	}

	ptr := int64(-1)
	for _, obj := range unstruct {
		switch v := obj.(type) {
		case map[string]any:
			if val, ok := v["$sapi_tokens"]; ok {
				ptr = int64(val.(float64))
				break
			}
		}
	}

	if ptr == -1 {
		return errors.New("unable to find reference to secret key")
	}

	keyPtrs := unstruct[ptr].([]any)
	secretKeys := make([]string, len(keyPtrs))
	for i, keyPtr := range keyPtrs {
		secretKeys[i] = unstruct[int64(keyPtr.(float64))].(string)
	}

	if len(secretKeys) == 0 {
		return errors.New("no secret keys found")
	}

	newSecret := DecodeHMACKey(secretKeys[len(secretKeys)-1])
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.hmacSecret != newSecret
	c.hmacSecret = newSecret
	slog.Info("refreshing HMAC secret key", "changed", changed)
	return nil
}

func (c *Client) req(ctx context.Context, method, path string, body io.Reader, hmac *HMACAttributes, opts ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, NewRequestError(err)
	}

	for _, opt := range opts {
		opt(req)
	}

	if hmac != nil {
		req.Header.Set("X-Sendico-Signature", hmac.Signature)
		req.Header.Set("X-Sendico-Nonce", hmac.Nonce)
		req.Header.Set("X-Sendico-Timestamp", fmt.Sprintf("%d", hmac.Timestamp))
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewRequestError(err)
	}

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		slog.ErrorContext(ctx, "unexpected status code",
			"body", string(body),
			"code", res.StatusCode,
			"path", path,
			"method", method,
		)
		_ = res.Body.Close()
		return nil, NewUnexpectedResponseCodeError(res.StatusCode)
	}

	return res, err
}

// Translate translates the given text from English to Japanese.
func (c *Client) Translate(ctx context.Context, text string) (string, error) {
	path := "/api/translate"

	request := orderedmap.New[string, any]()
	request.Set("from", "en")
	request.Set("string", text)
	request.Set("to", "ja")

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	hmac, err := BuildHMAC(HMACInput{
		Secret:  c.HMACSecret(),
		Path:    path,
		Payload: request,
	})
	if err != nil {
		return "", err
	}
	resp, err := c.req(ctx, http.MethodPost, path, bytes.NewReader(requestJSON), hmac, func(req *http.Request) {
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

type SearchOptions struct {
	TermJP   string
	MinPrice *int
	MaxPrice *int
}

// Search performs a search for the given term on the specified merchant. It will only return the first page of results.
// The supplied search term must be in Japanese.
func (c *Client) Search(ctx context.Context, shop Shop, opts SearchOptions) ([]Item, error) {
	path := url.URL{
		Path: fmt.Sprintf("/api/%s/items", shop.Identifier()),
	}

	params := orderedmap.New[string, any]()
	params.Set("global", "1")
	if opts.MaxPrice != nil {
		params.Set("max_price", fmt.Sprintf("%d", *opts.MaxPrice))
	}
	if opts.MinPrice != nil {
		params.Set("min_price", fmt.Sprintf("%d", *opts.MinPrice))
	}
	params.Set("page", "1")
	params.Set("search", opts.TermJP)

	q := path.Query()
	for pair := params.Oldest(); pair != nil; pair = pair.Next() {
		q.Add(pair.Key, fmt.Sprintf("%v", pair.Value))
	}
	path.RawQuery = q.Encode()

	hmac, err := BuildHMAC(HMACInput{
		Secret:  c.HMACSecret(),
		Path:    path.Path,
		Payload: params,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.req(ctx, http.MethodGet, path.String(), nil, hmac, func(req *http.Request) {
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

// BulkSearch performs a search for the given term on the specified merchants. It will only return the first page of results.
func (c *Client) BulkSearch(ctx context.Context, shops []Shop, opts SearchOptions) ([]Item, error) {
	items := make([]Item, 0)
	itemsMu := sync.Mutex{}

	g := new(errgroup.Group)
	for _, shop := range shops {
		shop := shop
		g.Go(func() error {
			results, err := c.Search(ctx, shop, opts)
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
