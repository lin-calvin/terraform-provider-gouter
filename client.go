package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotFound 表示资源在 gouter API 中不存在（HTTP 404）。
// Read 应捕获它并调用 RemoveResource，让 Terraform 决定重建。
var ErrNotFound = errors.New("not found")

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func NewClient(endpoint, token string) *Client {
	return &Client{
		endpoint: strings.TrimRight(endpoint, "/"),
		token:    token,
		http:     &http.Client{},
	}
}

func (c *Client) url(path, id string) string {
	// URL-encode the ID but preserve slashes (Go mux handles them)
	encoded := url.PathEscape(id)
	encoded = strings.ReplaceAll(encoded, "%2F", "/")
	return c.endpoint + path + "/" + encoded
}

func (c *Client) do(method, url string, body, result any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrNotFound, url)
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s: %s", url, errResp.Error)
		}
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}

	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

func (c *Client) Get(path, id string, result any) error {
	return c.do("GET", c.url(path, id), nil, result)
}

func (c *Client) Post(path string, body, result any) error {
	return c.do("POST", c.endpoint+"/"+strings.TrimLeft(path, "/"), body, result)
}

func (c *Client) Delete(path, id string) error {
	return c.do("DELETE", c.url(path, id), nil, nil)
}
