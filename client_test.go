package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func newTestClient(t *testing.T, fn func(req *http.Request) *http.Response) *Client {
	c, err := New("http://localhost:8080/")
	tnoerror(t, err)
	c.SetHTTPClientTransport(roundTripFunc(fn))
	return c
}

func TestClientDo(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		path      string
		query     url.Values
		headers   http.Header
		vbody     interface{}
		vresp     interface{}
		verr      interface{}
		expectErr bool
		fn        func(req *http.Request) *http.Response
	}{
		{
			name: "content-type header on empty body",
			fn: func(req *http.Request) *http.Response {
				tempty(t, req.Header.Get(HeaderContentType))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "content-type header with body",
			vbody: struct{}{},
			fn: func(req *http.Request) *http.Response {
				tstrequal(t, MIMEApplicationJSON, req.Header.Get(HeaderContentType))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "send string body",
			vbody: "test-body",
			fn: func(req *http.Request) *http.Response {
				b, _ := ioutil.ReadAll(req.Body)
				tstrequal(t, "\"test-body\"", string(b))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "send byte body",
			vbody: []byte("test-body"),
			fn: func(req *http.Request) *http.Response {
				b, _ := ioutil.ReadAll(req.Body)
				tstrequal(t, "test-body", string(b))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "send bytes.Buffer body",
			vbody: bytes.NewBufferString("test-body"),
			fn: func(req *http.Request) *http.Response {
				b, _ := ioutil.ReadAll(req.Body)
				tstrequal(t, "test-body", string(b))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "send io.Reader body",
			vbody: io.Reader(bytes.NewBufferString("test-body")),
			fn: func(req *http.Request) *http.Response {
				b, _ := ioutil.ReadAll(req.Body)
				tstrequal(t, "test-body", string(b))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "send io.ReadCloser body",
			vbody: io.NopCloser(bytes.NewBufferString("test-body")),
			fn: func(req *http.Request) *http.Response {
				b, _ := ioutil.ReadAll(req.Body)
				tstrequal(t, "test-body", string(b))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:  "receive response",
			vresp: &map[string]interface{}{},
			fn: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
				}
			},
		},
		{
			name:  "send query",
			query: url.Values{"foo": []string{"bar"}},
			fn: func(req *http.Request) *http.Response {
				tstrcontains(t, req.URL.RawQuery, "foo=bar")
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:    "send header",
			headers: http.Header{textproto.CanonicalMIMEHeaderKey("foo"): []string{"bar"}},
			fn: func(req *http.Request) *http.Response {
				tstrequal(t, "bar", req.Header.Get("foo"))
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:      "invalid method",
			method:    ";",
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:      "invalid body",
			vbody:     make(chan int),
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK}
			},
		},
		{
			name:      "http code 400",
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest}
			},
		},
		{
			name:      "read http error response failed",
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       ioutil.NopCloser(iotest.ErrReader(errors.New("err"))),
				}
			},
		},
		{
			name:      "read http response failed",
			vresp:     &map[string]interface{}{},
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(iotest.ErrReader(errors.New("err"))),
				}
			},
		},
		{
			name:      "http error and verr set",
			verr:      &map[string]interface{}{},
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
				}
			},
		},
		{
			name:      "http error and verr json unmarshal error",
			verr:      make(chan int),
			expectErr: true,
			fn: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, tt.fn)
			err := c.do(context.Background(), tt.method, tt.path, tt.query, tt.headers, tt.vbody, tt.vresp, tt.verr)
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error but got: %s", err)
			} else if tt.expectErr && err == nil {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func tstrcontains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected string %q contains %q", s, substr)
	}
}

func tempty(t *testing.T, s string) {
	t.Helper()
	if s != "" {
		t.Fatalf("expected empty string but got: %q", s)
	}
}

func tstrequal(t *testing.T, a, b string) {
	t.Helper()
	if a != b {
		t.Fatalf("expected equal strings but got: %q %q", a, b)
	}
}

func tnoerror(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error but got: %s", err)
	}
}
