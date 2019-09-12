package http

import (
	"context"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/logger"
	"github.com/Nerufa/go-shared/metric"
	"github.com/Nerufa/go-shared/provider"
	"github.com/Nerufa/go-shared/tracing"
	"github.com/go-chi/chi"
	"net/http"
)

// HTTP
type HTTP struct {
	ctx     context.Context
	log     logger.Logger
	cfg     Config
	metric  metric.Scope
	tracing tracing.Tracer
	mux     *chi.Mux
}

// ListenAndServe
func (m *HTTP) ListenAndServe() (err error) {
	server := &http.Server{
		Addr:    m.cfg.Bind,
		Handler: m.mux,
	}

	m.log.Info("start listen and serve http at %v", logger.Args(m.cfg.Bind))

	go func() {
		<-m.ctx.Done()
		m.log.Info("context cancelled, shutdown is raised")
		if e := server.Shutdown(context.Background()); e != nil {
			m.log.Error("graceful shutdown error, %v", logger.Args(e))
		}
	}()

	if err = server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			err = nil
		}
	}

	m.log.Info("http server stopped successfully")
	return
}

type Cors struct {
	Allowed []string
}

// Config
type Config struct {
	Debug   bool   `fallback:"shared.debug"`
	Bind    string `required:"true"`
	Cors    Cors
	invoker *invoker.Invoker
}

// OnReload
func (c *Config) OnReload(callback func(ctx context.Context)) {
	c.invoker.OnReload(callback)
}

// Reload
func (c *Config) Reload(ctx context.Context) {
	c.invoker.Reload(ctx)
}

// New
func New(ctx context.Context, mux *chi.Mux, set provider.AwareSet, cfg *Config) *HTTP {
	return &HTTP{
		ctx:     ctx,
		cfg:     *cfg,
		metric:  set.Metric,
		tracing: set.Tracer,
		mux:     mux,
		log:     set.Logger.WithFields(logger.Fields{"service": Prefix}),
	}
}
