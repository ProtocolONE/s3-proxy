package http

import (
	"context"
	"github.com/Nerufa/go-shared/config"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/provider"
	"github.com/ProtocolONE/s3-proxy/pkg/proxy"
	"github.com/go-chi/chi"
	"github.com/google/wire"
	"github.com/rs/cors"
	"github.com/thoas/go-funk"
	"net/http"
	"net/url"
)

var mux *chi.Mux

// Cfg
func Cfg(cfg config.Configurator) (*Config, func(), error) {
	c := &Config{}
	c.invoker = invoker.NewInvoker()
	e := cfg.UnmarshalKeyOnReload(UnmarshalKey, c)
	return c, func() {}, e
}

// CfgTest
func CfgTest() (*Config, func(), error) {
	c := &Config{}
	c.invoker = invoker.NewInvoker()
	return c, func() {}, nil
}

// Mux
func Mux(routers Routers, cfg *Config) (*chi.Mux, func(), error) {
	if mux != nil {
		return mux, func() {}, nil
	}
	mux = chi.NewRouter()
	if cfg.Debug || len(cfg.Cors.Allowed) == 0 {
		mux.Use(cors.AllowAll().Handler)
	} else {
		m := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
			AllowOriginRequestFunc: func(r *http.Request, origin string) bool {
				u, e := url.Parse(r.Header.Get("Origin"))
				if e != nil {
					return false
				}
				return funk.ContainsString(cfg.Cors.Allowed, u.Host)
			},
		})
		mux.Use(m.Handler)
	}
	// Middleware
	routers.Proxy.Use(mux)
	// Routers
	routers.Proxy.Routers(mux)
	return mux, func() {}, nil
}

// Routers
type Routers struct {
	Proxy *proxy.S3Proxy
}

var ProviderRouters = wire.NewSet(
	wire.Struct(new(Routers), "*"),
)

var ProviderRoutersTest = wire.NewSet(
	wire.Struct(new(Routers), "*"),
)

// Provider
func Provider(ctx context.Context, mux *chi.Mux, set provider.AwareSet, cfg *Config) (*HTTP, func(), error) {
	g := New(ctx, mux, set, cfg)
	return g, func() {}, nil
}

var (
	WireSet     = wire.NewSet(Provider, Cfg, Mux, ProviderRouters, proxy.WireSet)
	WireTestSet = wire.NewSet(Provider, CfgTest, Mux, ProviderRoutersTest, proxy.WireTestSet)
)
