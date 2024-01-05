package rpc

//go:generate go run compile.go clair.capnp

var NextProtos = []string{`clair-rpc-v1`}
