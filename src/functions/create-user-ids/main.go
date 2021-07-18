package main

import (
	"context"
	"load-test/handler"
	"load-test/mid"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
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

const TEST_USERS_NUM = 5

type IDInfo struct {
	ID string `json:"id"`
}

type Output struct {
	IDs []IDInfo `json:"ids"`
}

func lambdaHandler(ctx context.Context) (Output, error) {
	ids := make([]IDInfo, TEST_USERS_NUM)
	for i := 0; i < TEST_USERS_NUM; i++ {
		ids[i] = IDInfo{ID: uuid.NewString()}
	}

	return Output{IDs: ids}, nil
}
