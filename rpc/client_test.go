package rpc

import (
	"context"

	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
)

func ExampleClient() {
	c1, c2 := transport.NewPipe(2048)
	_ = c2
	tr := rpc.NewTransport(c1)
	c := &Client{
		c: rpc.NewConn(tr, nil),
	}
	ctx := context.Background()
	// Ignore above this, just need to magic a Client out of somewhere.

	m, err := c.Service(ctx)
	if err != nil {
		panic(err)
	}
	fut, done := m.Capabilities(ctx, nil)
	defer done()
	res, err := fut.Struct()
	if err != nil {
		panic(err)
	}
	avail, err := res.Avail()
	if err != nil {
		panic(err)
	}
	as := make([]string, avail.Len())
	for i := range as {
		as[i] = avail.At(i).String()
	}

}
