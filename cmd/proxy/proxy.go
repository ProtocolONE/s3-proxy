package proxy

import (
	"context"
	"github.com/Nerufa/go-shared/entrypoint"
	"github.com/ProtocolONE/s3-proxy/cmd"
	"github.com/ProtocolONE/s3-proxy/pkg/http"
	"github.com/spf13/cobra"
)

const Prefix = "cmd.proxy"

var (
	Cmd = &cobra.Command{
		Use:           "proxy",
		Short:         "S3 Proxy Server",
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(_ *cobra.Command, _ []string) {
			var (
				s *http.HTTP
				c func()
				e error
			)
			cmd.Slave.Executor(func(ctx context.Context) error {
				initial, _ := entrypoint.CtxExtractInitial(ctx)
				s, c, e = http.Build(ctx, initial, cmd.Observer)
				if e != nil {
					return e
				}
				c()
				return nil
			}, func(ctx context.Context) error {
				if e := s.ListenAndServe(); e != nil {
					return e
				}
				return nil
			})
		},
	}
)

func init() {
	// pflags
	Cmd.PersistentFlags().StringP(http.UnmarshalKeyBind, "b", ":8080", "bind address")
}
