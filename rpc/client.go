package rpc

import (
	"context"
	"fmt"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/quic-go/quic-go"

	"github.com/quay/clair/v4/rpc/internal/proto"
)

type Client struct {
	c *rpc.Conn
}

func NewClient(ctx context.Context, conn quic.Connection) (*Client, error) {
	// TODO(hank) Add snappy or lz4 compression?
	s, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("rpc: constructing Client: %v", err)
	}

	tr := rpc.NewPackedStreamTransport(s)
	c := &Client{
		c: rpc.NewConn(tr, nil),
	}

	_, file, line, _ := runtime.Caller(1)
	runtime.SetFinalizer(c, func(_ *Client) {
		panic(fmt.Sprintf("%s:%d: Client not closed", file, line))
	})

	return c, nil
}

func (c *Client) Close() error {
	runtime.SetFinalizer(c, nil)
	return c.c.Close()
}

func (c *Client) Service(ctx context.Context) (proto.Main, error) {
	return proto.Main(c.c.Bootstrap(ctx)), nil
}
