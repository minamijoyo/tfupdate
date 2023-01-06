package tfregistry

import (
	"context"
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

// ArtifactoryClient manages communication with the Terraform Registry API.
type ArtifactoryClient struct {
	// httpClient is a http client which communicates with the API.
	httpClient *http.Client
	// BaseURL is a base url for API requests. Defaults to the public Terraform Registry API.
	BaseURL *url.URL
}

// NewClient returns a new Client instance.
func NewArtifactoryClient(httpClient *http.Client) *ArtifactoryClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	c := &ArtifactoryClient{httpClient: httpClient, BaseURL: baseURL}
	return c
}

// newRequest builds a http Request instance.
func (c *ArtifactoryClient) newRequest(ctx context.Context, method string, subPath string, body io.Reader) (*http.Request, error) {
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

func (c *ArtifactoryClient) ModuleLatestForProvider(ctx context.Context, req *ModuleLatestForProviderRequest) (*ModuleLatestForProviderResponse, error) {
	if len(req.Namespace) == 0 {
		return nil, fmt.Errorf("Invalid request. Namespace is required. req = %#v", req)
	}
	if len(req.Name) == 0 {
		return nil, fmt.Errorf("Invalid request. Name is required. req = %#v", req)
	}
	if len(req.Provider) == 0 {
		return nil, fmt.Errorf("Invalid request. Provider is required. req = %#v", req)
	}

	if c.BaseURL == nil {

	}
	subPath := fmt.Sprintf("%s%s/%s/%s/versions", moduleV1Service, req.Namespace, req.Name, req.Provider)
	//  https://artifactory.foo.internal:443/artifactory/api/terraform/v1/modules/terraform__modules/k8s-namespace/module_name/versions
	httpRequest, err := c.newRequest(ctx, "GET", subPath, nil)
	if err != nil {
		return nil, err
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to HTTP Request: err = %s, req = %#v", err, httpRequest)
	}

	if httpResponse.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP Status Code: %d", httpResponse.StatusCode)
	}

	var res ModuleLatestForProviderResponse
	if err := decodeBody(httpResponse, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
