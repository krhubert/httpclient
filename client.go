package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	HeaderContentType   = "Content-Type"
	MIMEApplicationJSON = "application/json"
)

// DumpRequest control if requests should be dump to stdout.
var DumpRequest = false

// Client is responsible for communication with strapi api.
type Client struct {
	url    *url.URL
	authfn func(req *http.Request)
	client *http.Client
}

func New(address string) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(address, "/"))
	if err != nil {
		return nil, fmt.Errorf("httpclient: cannot parse url: %w", err)
	}

	return &Client{
		url:    u,
		client: &http.Client{},
	}, nil
}

func (c *Client) SetAuthFunction(authfn func(req *http.Request)) {
	c.authfn = authfn
}

// SetHTTPClientTransport sets http transport, so it can be mocked or used for tests.
func (c *Client) SetHTTPClientTransport(transport http.RoundTripper) {
	c.client.Transport = transport
}

func (c *Client) Get(path string, query url.Values, headers http.Header, vresp, verr interface{}) error {
	return c.do(context.Background(), http.MethodGet, path, query, headers, nil, vresp, verr)
}

func (c *Client) Post(path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(context.Background(), http.MethodPost, path, query, headers, vbody, vresp, verr)
}

func (c *Client) Patch(path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(context.Background(), http.MethodPatch, path, query, headers, vbody, vresp, verr)
}

func (c *Client) Delete(path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(context.Background(), http.MethodDelete, path, query, headers, vbody, vresp, verr)
}

func (c *Client) GetCtx(ctx context.Context, path string, query url.Values, headers http.Header, vresp, verr interface{}) error {
	return c.do(ctx, http.MethodGet, path, query, headers, nil, vresp, verr)
}

func (c *Client) PostCtx(ctx context.Context, path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(ctx, http.MethodPost, path, query, headers, vbody, vresp, verr)
}

func (c *Client) PatchCtx(ctx context.Context, path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(ctx, http.MethodPatch, path, query, headers, vbody, vresp, verr)
}

func (c *Client) DeleteCtx(ctx context.Context, path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	return c.do(ctx, http.MethodDelete, path, query, headers, vbody, vresp, verr)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, headers http.Header, vbody, vresp, verr interface{}) error {
	u := &url.URL{
		Scheme: c.url.Scheme,
		Host:   c.url.Host,
		Path:   path,
	}

	// save url for logging error.
	// get it without query because it may contains sensitive data
	safeurl := u.Redacted()

	// now, we can set the query
	u.RawQuery = query.Encode()

	var body io.Reader

	if vbody != nil {
		switch t := vbody.(type) {
		case []byte:
			body = bytes.NewBuffer(t)
		case *bytes.Buffer:
			body = t
		case io.ReadCloser:
			body = t
		case io.Reader:
			body = t
		default:
			b, err := json.Marshal(t)
			if err != nil {
				return NewClientError(safeurl, 0, "encode request body failed", err)
			}
			body = bytes.NewBuffer(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return NewClientError(safeurl, 0, "create http request failed", err)
	}

	if c.authfn != nil {
		c.authfn(req)
	}

	for key, value := range headers {
		req.Header[key] = value
	}

	// add content-type "application/json" if missing
	if req.Header.Get(HeaderContentType) == "" && vbody != nil {
		req.Header.Add(HeaderContentType, MIMEApplicationJSON)
	}

	var reqBody io.ReadCloser
	if DumpRequest {
		req.Body, reqBody, _ = drainBody(req.Body)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return NewClientError(safeurl, 0, "http do request failed", err)
	}
	defer resp.Body.Close()

	if DumpRequest {
		req.Body = reqBody
		dumpRequest(req, resp)
	}

	if isErrorCode(resp.StatusCode) {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return NewClientError(safeurl, resp.StatusCode, "read response error body failed", err)
		}

		if verr != nil {
			if err := json.Unmarshal(respBody, verr); err != nil {
				return NewClientError(safeurl, resp.StatusCode, "decode response body failed", err)
			}
		}

		return NewClientError(safeurl, resp.StatusCode, "http returned error status code", errors.New(string(respBody)))
	}

	if vresp != nil {
		if err := json.NewDecoder(resp.Body).Decode(vresp); err != nil {
			return NewClientError(safeurl, resp.StatusCode, "decode response body failed", err)
		}
	}

	return nil
}

func isErrorCode(code int) bool {
	return code >= 400
}
