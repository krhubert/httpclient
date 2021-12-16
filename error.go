package httpclient

import (
	"strconv"
	"strings"
)

type ClientError struct {
	Url        string
	StatusCode int
	Message    string
	Reason     error
}

func NewClientError(url string, statusCode int, message string, err error) *ClientError {
	return &ClientError{
		Url:        url,
		StatusCode: statusCode,
		Message:    message,
		Reason:     err,
	}
}

func (e *ClientError) Error() string {
	var buf strings.Builder
	buf.WriteString("httpclient: ")
	if e.Url != "" {
		buf.WriteString("url=")
		buf.WriteString(e.Url)
		buf.WriteRune(' ')
	}
	if e.StatusCode > 0 {
		buf.WriteString("statusCode=")
		buf.WriteString(strconv.Itoa(e.StatusCode))
		buf.WriteRune(' ')
	}
	if e.Message != "" {
		buf.WriteString(e.Message)
	}

	if e.Reason != nil {
		buf.WriteString(": ")
		buf.WriteString(e.Reason.Error())
	}
	return buf.String()
}

func (e *ClientError) Unwrap() error {
	return e.Reason
}
