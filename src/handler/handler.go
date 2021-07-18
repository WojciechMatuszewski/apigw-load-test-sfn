package handler

import "github.com/aws/aws-lambda-go/lambda"

type Handler lambda.Handler

func NewHandler(lambdaHandler interface{}, mw ...Middleware) Handler {
	return wrapMiddleware(lambda.NewHandler(lambdaHandler), mw)
}

type Middleware func(handler Handler) Handler

func wrapMiddleware(handler Handler, mw []Middleware) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h := mw[i]
		if h != nil {
			handler = h(handler)
		}
	}

	return handler
}
