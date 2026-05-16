package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	handler, err := createHandler()
	if err != nil {
		panic(err)
	}

	if os.Getenv("APP_ENV") == "production" {
		lh := &lambdaHandler{handler: handler}
		lambda.Start(lh.Handle)
	} else {
		startDevServer(handler)
	}
}
