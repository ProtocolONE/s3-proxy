package proxy

import (
	"context"
	"github.com/Nerufa/go-shared/config"
	"github.com/Nerufa/go-shared/invoker"
	"github.com/Nerufa/go-shared/provider"
	"github.com/ProtocolONE/s3-proxy/pkg/s3"
	"github.com/google/wire"
)

// Cfg
func Cfg(cfg config.Configurator) (*Config, func(), error) {
	c := &Config{}
	c.invoker = invoker.NewInvoker()
	e := cfg.UnmarshalKeyOnReload(UnmarshalKey, c, StringToHumanizeSizeHookFunc())
	return c, func() {}, e
}

// CfgTest
func CfgTest() (*Config, func(), error) {
	c := &Config{}
	c.invoker = invoker.NewInvoker()
	return c, func() {}, nil
}

// Provider
func Provider(ctx context.Context, set provider.AwareSet, cfg *Config, storage s3.Storage) (*S3Proxy, func(), error) {
	g := New(ctx, set, cfg, storage)
	return g, func() {}, nil
}

var (
	WireSet = wire.NewSet(
		Provider,
		Cfg,
		s3.WireSet,
		wire.Bind(new(s3.Storage), new(*s3.S3)),
	)
	WireTestSet = wire.NewSet(
		Provider,
		CfgTest,
		s3.WireTestSet,
		wire.Bind(new(s3.Storage), new(*s3.S3)),
	)
)
