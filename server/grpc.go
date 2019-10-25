package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/lukasjarosch/enki/interceptor"
)

// HttpConfig defines all configuration fields for the gRPC server
type GrpcConfig struct {
	Port        string        `mapstructure:"grpc-port"`
	GracePeriod time.Duration `mapstructure:"grpc-grace-period"`
}

// GrpcServer defines the default behaviour of gRPC servers
type GrpcServer struct {
	GoogleGrpc      *grpc.Server
	logger          *zap.Logger
	config          *GrpcConfig
	listener        net.Listener
	healthy         bool
	requestDuration prometheus.Histogram
}

// NewGrpcServer returns a new, pre-initialized, GrpcServer instance
// The application will terminate if the server cannot bind to the configured port.
// If the application does not terminate, the port is open and a raw gRPC server has been created after
// the call of NewGrpcServer()
// 'tracer' may be nil, in this case the feature is disabled
func NewGrpcServer(logger *zap.Logger, config *GrpcConfig) *GrpcServer {
	srv := &GrpcServer{
		logger: logger.Named("grpc"),
		config: config,
	}

	srv.setupGrpc()

	return srv
}

// setupGrpc will create a new, raw google gRPC server as well as the listener
// If the listener cannot bind to the port, it's considered a fatal error on which
// the application will be terminated.
func (srv *GrpcServer) setupGrpc() {
	var err error

	grpcprometheus.EnableHandlingTimeHistogram()

	srv.GoogleGrpc = grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpcrecovery.UnaryServerInterceptor(),
			interceptor.RequestId(),
			grpcopentracing.UnaryServerInterceptor(),
			grpcprometheus.UnaryServerInterceptor,
		)),
	)
	srv.listener, err = net.Listen("tcp", fmt.Sprintf(":%v", srv.config.Port))
	if err != nil {
		srv.logger.Fatal("failed to listen on port", zap.Error(err))
	}
}

// ListenAndServe ties everything together and runs the gRPC server in a separate goroutine.
// The method then blocks until the passed context is cancelled, so this method should also be started
// as goroutine if more work is needed after starting the gRPC server.
func (srv *GrpcServer) ListenAndServe(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO serve in goroutine
	go func() {
		srv.logger.Info("gRPC server running", zap.String("port", srv.config.Port))
		if err := srv.GoogleGrpc.Serve(srv.listener); err != nil {
			srv.healthy = false
			srv.logger.Fatal("gRPC server crashed", zap.Error(err))
		}
	}()

	// server is healthy, tell everyone \(°ヮﾟ°)/
	srv.healthy = true

	<-ctx.Done()

	// health checks fail from now on
	srv.healthy = false

	srv.logger.Info("gRPC server shutdown requested")
	srv.shutdownGrpc()
}

// Health returns a http.HandlerFunc, it reports the gRPC server health: OK or UNHEALTHY
func (srv *GrpcServer) Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This endpoint must always return a 200.
		// If it does not return a 200, the health endpoint itself is broken.
		// If the service is healthy or not is defined through the atomic 'healthy' var
		w.WriteHeader(http.StatusOK)

		if srv.healthy {
			_, _ = w.Write([]byte("OK"))
		} else {
			_, _ = w.Write([]byte("UNHEALTHY"))
		}
	}
}

// shutdownGrpc gracefully shuts down the gRPC server
func (srv *GrpcServer) shutdownGrpc() {
	stopped := make(chan struct{})
	go func() {
		srv.GoogleGrpc.GracefulStop()
		close(stopped)
	}()
	t := time.NewTicker(srv.config.GracePeriod)
	select {
	case <-t.C:
		srv.logger.Warn("gRPC server graceful shutdown timed-out", zap.Duration("grace period", srv.config.GracePeriod))
	case <-stopped:
		srv.logger.Info("gRPC server stopped gracefully")
		t.Stop()
	}
}
