package main

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type lambdaHandler struct {
	handler http.Handler
}

func (h *lambdaHandler) Handle(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	bodyReader := strings.NewReader(req.Body)
	httpReq, err := http.NewRequestWithContext(ctx, req.RequestContext.HTTP.Method, req.RawPath, bodyReader)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: 500, Body: "internal error"}, nil
	}

	httpReq.URL.RawQuery = req.RawQueryString

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
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
