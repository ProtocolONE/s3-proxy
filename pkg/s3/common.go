package s3

import (
	"context"
	"io"
)

const (
	Prefix       = "pkg.s3"
	UnmarshalKey = "s3"
)

type Storage interface {
	Upload(ctx context.Context, key string, r io.Reader) (n int64, err error)
	Download(ctx context.Context, key string, w io.Writer) (n int64, err error)
	Delete(ctx context.Context, key string) error
	Endpoint() string
	DownloadURL() string
}
