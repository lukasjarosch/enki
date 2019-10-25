package interceptor

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

func ZipkinInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		span, ctx := opentracing.StartSpanFromContext(ctx, info.FullMethod)
		defer span.Finish()

		defer func() {
			if span := opentracing.SpanFromContext(ctx); span != nil {
				span.SetTag("error", err.Error())
			}
		}()

		return handler(ctx, req)
	}
}