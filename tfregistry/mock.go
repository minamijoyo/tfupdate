package tfregistry

import (
	"net/http"
	"net/http/httptest"
	"net/url"
)

// newMockServer returns a new mock server for testing.
func newMockServer() (*http.ServeMux, *url.URL) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mockServerURL, _ := url.Parse(server.URL)
	return mux, mockServerURL
}

// newTestClient returns a new client for testing.
func newTestClient(mockServerURL *url.URL) *Client {
	httpClient := &http.Client{}
	c := NewClient(httpClient)
	c.BaseURL = mockServerURL
	return c
}
