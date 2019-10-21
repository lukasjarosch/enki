package interceptor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	enkimetadata "github.com/lukasjarosch/enki/metadata"
)

func RequestId() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {

			requestID := md.Get(enkimetadata.RequestID)
			if len(requestID) > 0 {
				ctx = context.WithValue(ctx, enkimetadata.RequestID, requestID)
				return handler(ctx, req)
			}

			newRequestID := newRequestID()
			md.Append(enkimetadata.RequestID, newRequestID)
			ctx = metadata.NewIncomingContext(ctx, md)
			ctx = context.WithValue(ctx, enkimetadata.RequestID, newRequestID)
			return handler(ctx, req)
		}

		newRequestID := newRequestID()
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(enkimetadata.RequestID, newRequestID))
		ctx = context.WithValue(ctx, enkimetadata.RequestID, newRequestID)
		return handler(ctx, req)
	}

}

func newRequestID() string {
	return uuid.New().String()
}
