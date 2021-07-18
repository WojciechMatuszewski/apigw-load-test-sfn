package main

import (
	"context"
	"load-test/handler"
	"load-test/mid"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	h := handler.NewHandler(
		lambdaHandler,
		mid.Logger(logger),
		mid.Errors(logger),
		mid.Panics(logger),
	)
	lambda.StartHandler(h)
}

func lambdaHandler(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusOK, Body: "Hi there!"}, nil
}
