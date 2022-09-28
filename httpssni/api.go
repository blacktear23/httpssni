package httpssni

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type HeaderIterator interface {
	Iterate(key string, value string) bool
}

type HTTPSCtx struct {
	Method     string
	Addr       string
	HostPath   string
	Headers    map[string]string
	body       []byte
	skipVerify bool
	timeout    int
}

type HTTPResponse struct {
	StatusCode    int
	Proto         string
	Header        map[string]string
	body          io.ReadCloser
	ContentLength int64
}

type ReadResult struct {
	Buf   []byte
	Error string
	Size  int
}

func (r *ReadResult) GetBuffer() []byte {
	return r.Buf
}

func NewHTTPResponse(resp *http.Response) *HTTPResponse {
	header := map[string]string{}
	for k, v := range resp.Header {
		header[k] = v[0]
	}
	return &HTTPResponse{
		StatusCode:    resp.StatusCode,
		Proto:         resp.Proto,
		ContentLength: resp.ContentLength,
		Header:        header,
		body:          resp.Body,
	}
}

func (r *HTTPResponse) GetHeader(hdr string) string {
	val, have := r.Header[hdr]
	if !have {
		return ""
	}
	return val
}

func (r *HTTPResponse) Range(it HeaderIterator) {
	for k, val := range r.Header {
		if !it.Iterate(k, val) {
			break
		}
	}
}

func (r *HTTPResponse) Read(size int) *ReadResult {
	buf := make([]byte, size)
	n, err := r.body.Read(buf)
	if err != nil {
		return &ReadResult{
			Error: err.Error(),
			Size:  n,
			Buf:   buf,
		}
	}
	return &ReadResult{
		Error: "",
		Size:  n,
		Buf:   buf,
	}
}

func (r *HTTPResponse) Close() int {
	err := r.body.Close()
	if err != nil {
		return -1
	}
	return 0
}

type Response struct {
	Resp  *HTTPResponse
	Error string
}

func NewHTTPSCtx(method string, hostPath string, addr string) *HTTPSCtx {
	return &HTTPSCtx{
		Method:     method,
		Addr:       addr,
		HostPath:   hostPath,
		Headers:    map[string]string{},
		body:       nil,
		skipVerify: false,
		timeout:    30,
	}
}

func (c *HTTPSCtx) SetSkipVerify(skip bool) {
	c.skipVerify = skip
}

func (c *HTTPSCtx) SetHeader(header string, value string) {
	c.Headers[header] = value
}

func (c *HTTPSCtx) SetBody(body []byte) {
	c.body = body
}

func (c *HTTPSCtx) SetTimeout(timeout int) {
	c.timeout = timeout
}

func (c *HTTPSCtx) PerformRequest() *Response {
	var body io.Reader = nil
	if c.body != nil {
		body = bytes.NewBuffer(c.body)
	}
	url := fmt.Sprintf("https://%s", c.HostPath)
	req, err := http.NewRequest(c.Method, url, body)
	if err != nil {
		return &Response{
			Resp:  nil,
			Error: err.Error(),
		}
	}
	for k, v := range c.Headers {
		req.Header.Add(k, v)
	}

	raddr := c.Addr
	tp := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := net.Dialer{}
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, fmt.Sprintf("%s:%s", raddr, port))
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.skipVerify},
	}

	client := &http.Client{
		Transport: tp,
		Timeout:   time.Duration(c.timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return &Response{
			Resp:  nil,
			Error: err.Error(),
		}
	}
	return &Response{
		Resp:  NewHTTPResponse(resp),
		Error: "",
	}
}
