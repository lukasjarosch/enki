package server
import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type HttpConfig struct {
	Port        string        `mapstructure:"http-port"`
	GracePeriod time.Duration `mapstructure:"http-grace-period"`
}

type HttpServer struct {
	logger  *zap.Logger
	config  *HttpConfig
	healthy bool
	requestDuration prometheus.Histogram
}

func NewHttpServer(logger *zap.Logger, config *HttpConfig) *HttpServer {
	srv := &HttpServer{
		logger:  logger.Named("http"),
		config:  config,
		healthy: false,
	}

	srv.registerMetrics()

	return srv
}

// Health returns a http.HandlerFunc, it reports the gRPC server health: OK or UNHEALTHY
func (srv *HttpServer) Health() http.HandlerFunc {
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

func (srv *HttpServer) registerMetrics()  {
	srv.requestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_duration_ms",
		Help:    "Request duration in milliseconds",
		Buckets: []float64{50, 100, 250, 500, 1000},
	})
	prometheus.MustRegister(srv.requestDuration)
}

func (srv *HttpServer) ListenAndServe(ctx context.Context, wg *sync.WaitGroup, handler http.Handler) {
	defer wg.Done()

	if srv.config.Port == "" {
		srv.logger.Error("missing http port, server will not be started")
		return
	}

	httpServer := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%s", srv.config.Port), Handler: handler}

	// serve
	go func() {
		srv.logger.Info("http server started", zap.String("port", srv.config.Port))
		srv.healthy = true
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.logger.Fatal("http server crashed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	srv.logger.Info("http server shutdown requested")
	srv.healthy = false

	gracePeriod := 5 * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracePeriod)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		srv.logger.Warn("gRPC server graceful shutdown timed-out", zap.Error(err), zap.Duration("grace period", gracePeriod))
	} else {
		srv.logger.Info("http server stopped gracefully")
	}
}
