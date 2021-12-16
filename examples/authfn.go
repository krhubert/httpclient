package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/krhubert/httpclient"
)

func main() {
	c, err := httpclient.New("https://api.example.com")
	if err != nil {
		log.Fatal(err)
	}

	// header auth
	c.SetAuthFunction(func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer access-token")
	})

	// query auth
	c.SetAuthFunction(func(req *http.Request) {
		q := req.URL.Query()
		q.Add("token", "query-token")
		req.URL.RawQuery = q.Encode()
	})

	// complex auth
	c.SetAuthFunction(func(req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/oauth") {
			// do nothing on oauth
			return
		}

		if req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer access-token")
		}
	})
}
