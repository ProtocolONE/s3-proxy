package main

import (
	"github.com/ProtocolONE/s3-proxy/cmd/proxy"
	"github.com/ProtocolONE/s3-proxy/cmd/root"
	"github.com/ProtocolONE/s3-proxy/cmd/version"
)

func main() {
	root.Execute(proxy.Cmd, version.Cmd)
}
