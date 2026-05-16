package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type lambdaHandler struct {
	handler http.Handler
}

func (h *lambdaHandler) Handle(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	log.Printf("BRIDGE method=%s path=%s bodyLen=%d ctype=%s",
		req.RequestContext.HTTP.Method, req.RawPath, len(req.Body), req.Headers["content-type"])

	u := &url.URL{
		Path:     req.RawPath,
		RawQuery: req.RawQueryString,
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = strings.NewReader(req.Body)
	}

	httpReq := httptest.NewRequest(req.RequestContext.HTTP.Method, u.String(), body)

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httpReq)

	resp := events.LambdaFunctionURLResponse{
		StatusCode: rec.Code,
		Headers:    make(map[string]string),
		Body:       rec.Body.String(),
	}

	for k, vals := range rec.Header() {
		resp.Headers[k] = strings.Join(vals, ", ")
	}

	return resp, nil
}
