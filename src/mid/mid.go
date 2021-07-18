package mid

import (
	"context"
	"fmt"
	"load-test/handler"

	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) handler.Middleware {
	m := func(handler handler.Handler) handler.Handler {
		h := middlewareFunc(func(ctx context.Context, payload []byte) ([]byte, error) {
			logger.Info(
				"BEFORE",
				zap.ByteString("payload", (payload)),
			)

			out, err := handler.Invoke(ctx, payload)
			if out != nil {
				logger.Info(
					"AFTER",
					zap.ByteString("output", out),
				)
			}

			return out, err
		})

		return h
	}

	return m

}

func Errors(logger *zap.Logger) handler.Middleware {
	m := func(handler handler.Handler) handler.Handler {
		h := middlewareFunc(func(ctx context.Context, payload []byte) ([]byte, error) {
			out, err := handler.Invoke(ctx, payload)
			if err != nil {
				logger.Error(
					"ERROR",
					zap.Error(err),
				)
			}

			return out, err
		})
		return h
	}

	return m
}

func Panics(logger *zap.Logger) handler.Middleware {
	m := func(handler handler.Handler) handler.Handler {
		h := middlewareFunc(func(ctx context.Context, payload []byte) (out []byte, err error) {
			defer func() {
				r := recover()
				if r != nil {
					err = fmt.Errorf("panic %v", r)

					logger.Error(
						"PANIC",
						zap.Stack("stack"),
					)
				}
			}()

			return handler.Invoke(ctx, payload)
		})
		return h
	}

	return m
}

type middlewareFunc func(ctx context.Context, payload []byte) ([]byte, error)

func (m middlewareFunc) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	return m(ctx, payload)
}
