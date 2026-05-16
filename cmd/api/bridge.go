package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type lambdaHandler struct {
	handler http.Handler
}

func (h *lambdaHandler) Handle(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	log.Printf("BRIDGE method=%s path=%s bodyLen=%d ctype=%s",
		req.RequestContext.HTTP.Method, req.RawPath, len(req.Body), req.Headers["content-type"])

	httpReq, err := http.NewRequestWithContext(ctx, req.RequestContext.HTTP.Method, req.RawPath, nil)
	if err != nil {
		log.Printf("BRIDGE NewRequest error: %v", err)
		return events.LambdaFunctionURLResponse{StatusCode: 500, Body: "internal error"}, nil
	}

	httpReq.URL.RawQuery = req.RawQueryString

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if len(req.Body) > 0 {
		body := req.Body
		httpReq.Body = io.NopCloser(strings.NewReader(body))
		httpReq.ContentLength = int64(len(body))
		httpReq.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(body)), nil
		}
	}

	rec := &responseRecorder{
		headers:    make(http.Header),
		statusCode: 200,
	}

	h.handler.ServeHTTP(rec, httpReq)

	resp := events.LambdaFunctionURLResponse{
		StatusCode: rec.statusCode,
		Headers:    make(map[string]string),
		Body:       rec.body.String(),
	}

	for k, vals := range rec.headers {
		resp.Headers[k] = strings.Join(vals, ", ")
	}

	return resp, nil
}

type responseRecorder struct {
	headers    http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}
