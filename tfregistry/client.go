package tfregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

// To avoid depending on a specific version of Terraform,
// we implement a pure Terraform Registry API client.
// https://www.terraform.io/docs/registry/api.html
//
// Currently only the official public registry is supported.
// There are other APIs and request/response fields,
// but we define only the ones we need here to keep it simple.

const (
	// The public Terraform Registry API endpoint.
	defaultBaseURL = "https://registry.terraform.io/"
)

// Client manages communication with the Terraform Registry API.
type Client struct {
	// httpClient is a http client which communicates with the API.
	httpClient *http.Client
	// BaseURL is a base url for API requests. Defaults to the public Terraform Registry API.
	BaseURL *url.URL
}

// NewClient returns a new Client instance.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	c := &Client{httpClient: httpClient, BaseURL: baseURL}
	return c
}

// newRequest builds a http Request instance.
func (c *Client) newRequest(ctx context.Context, method string, subPath string, body io.Reader) (*http.Request, error) {
	endpointURL := *c.BaseURL
	endpointURL.Path = path.Join(c.BaseURL.Path, subPath)

	req, err := http.NewRequest(method, endpointURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: err = %s, method = %s, endpointURL = %s, body = %#v", err, method, endpointURL.String(), body)
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// decodeBody decodes a raw body data into a specific response type structure.
func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(out)
	if err != nil {
		return fmt.Errorf("failed to decode response: err = %s, resp = %#v", err, resp)
	}

	return nil
}
