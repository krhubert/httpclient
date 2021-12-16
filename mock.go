package httpclient

import (
	"net/http"
)

// FakeRoundTrip is fake transport, that logs connection.
// It may be used during test and debugs
type FakeRoundTrip struct{}

// NewFakeRoundTrip creates a fake transport.
func NewFakeRoundTrip() *FakeRoundTrip {
	return &FakeRoundTrip{}
}

func (f *FakeRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
	}, nil
}
