package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"
)

func dumpRequest(req *http.Request, resp *http.Response) {
	var buf bytes.Buffer

	buf.WriteString("\n")
	buf.WriteString("=================\n")
	buf.WriteString(">>>> request <<<<\n")
	buf.WriteString("=================\n")
	buf.WriteString("\n")

	buf.WriteString(">>>> command <<<<\n")
	buf.WriteString("\n")

	if ct := req.Header.Get("Content-Type"); req.Method == http.MethodGet ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/x-www-form-urlencoded") {
		buf.WriteString(curlcmd(req))
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString(">>>> request dump <<<<\n")
	buf.WriteString("\n")
	out, _ := httputil.DumpRequest(req, false)
	buf.Write(out)

	if strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		reqBody, _ := ioutil.ReadAll(req.Body)
		_ = json.Indent(&buf, reqBody, "", "  ")
		buf.WriteString("\n\n")
	}

	buf.WriteString("**** response ****\n")
	buf.WriteString("\n")
	out, _ = httputil.DumpResponse(resp, false)
	buf.Write(out)

	var respBody io.ReadCloser
	resp.Body, respBody, _ = drainBody(resp.Body)
	respBodyData, _ := ioutil.ReadAll(respBody)
	json.Indent(&buf, respBodyData, "", "  ")
	buf.WriteString("\n")

	buf.WriteString("=====================\n")
	buf.WriteString(">>>> end request <<<<\n")
	buf.WriteString("=====================\n")

	fmt.Println(buf.String())
}

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err := b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func bashEscape(str string) string {
	return `'` + strings.Replace(str, `'`, `'\''`, -1) + `'`
}

func curlcmd(req *http.Request) string {
	var cmd []string

	cmd = append(cmd, "curl")

	if req.URL != nil && req.URL.Scheme == "https" {
		cmd = append(cmd, "-k")
	}

	cmd = append(cmd, "-X", bashEscape(req.Method))

	if req.Body != nil {
		var buff bytes.Buffer

		if _, err := buff.ReadFrom(req.Body); err != nil {
			return ""
		}
		if len(buff.String()) > 0 {
			cmd = append(cmd, "-d", bashEscape(buff.String()))
		}
	}

	var keys []string
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		cmd = append(cmd, "-H", bashEscape(fmt.Sprintf("%s: %s", k, strings.Join(req.Header[k], " "))))
	}

	cmd = append(cmd, bashEscape(req.URL.String()))
	return strings.Join(cmd, " ")
}
