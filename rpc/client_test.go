package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	"github.com/quay/zlog"
	"golang.org/x/sync/errgroup"

	"github.com/quay/clair/v4/rpc/internal/proto"
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

func TestSimple(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)
	pc, ps := net.Pipe()
	ctx, done := context.WithCancelCause(ctx)
	clientDone := errors.New("client done")
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// Client
		defer done(clientDone)
		conn := rpc.NewConn(rpc.NewPackedStreamTransport(pc), nil)
		defer conn.Close()

		srv := proto.Main(conn.Bootstrap(ctx))
		t.Log("call start")
		fut, done := srv.Capabilities(ctx, nil) // No Params
		defer done()
		res, err := fut.Struct()
		if err != nil {
			return err
		}
		capList, err := res.Avail()
		if err != nil {
			return err
		}
		caps := make([]proto.Services, capList.Len())
		for i := 0; i < capList.Len(); i++ {
			caps[i] = capList.At(i)
		}
		t.Log("call resolved")
		t.Logf("capabilities: %+v", caps)

		idxfut, done := srv.Indexer(ctx, nil)
		defer done()
		idx := idxfut.Srv()
		if err := idx.Resolve(ctx); err != nil {
			return fmt.Errorf("resolving indexer service: %v", err)
		}
		mFut, done := idx.GetMeta(ctx, func(p proto.Indexer_getMeta_Params) error {
			d, err := p.NewManifest()
			if err != nil {
				return err
			}
			d.SetAlgorithm("sha256")
			d.SetDigest(bytes.Repeat([]byte{0x00, 0x01}, 16))
			return nil
		})
		defer done()
		if _, err := mFut.Struct(); err == nil {
			return errors.New("GetMeta: wanted error, got nil")
		}

		return nil
	})
	eg.Go(func() error {
		// Server
		client := proto.Main_ServerToClient(mainImpl{})
		conn := rpc.NewConn(rpc.NewPackedStreamTransport(ps), &rpc.Options{
			BootstrapClient: capnp.Client(client),
		})
		defer conn.Close()
		t.Log("server started")
		select {
		case <-ctx.Done():
			t.Log("server done")
			return conn.Close()
		case <-conn.Done():
			return nil
		}
	})

	if err := eg.Wait(); err != nil {
		t.Error(err)
	}
}

type mainImpl struct{}

var _ proto.Main_Server = mainImpl{}

func (m mainImpl) Capabilities(ctx context.Context, call proto.Main_capabilities) error {
	call.Go()
	caps := []string{"indexer"}
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	avail, err := res.NewAvail(int32(len(caps)))
	if err != nil {
		return err
	}
	for i, s := range caps {
		avail.Set(i, proto.ServicesFromString(s))
	}

	return nil
}

func (m mainImpl) Indexer(ctx context.Context, call proto.Main_indexer) error {
	idx := proto.Indexer_ServerToClient(indexerImpl{})

	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetSrv(idx)
}

func (m mainImpl) Matcher(ctx context.Context, call proto.Main_matcher) error {
	mx := proto.Matcher_ServerToClient(matcherImpl{})

	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetSrv(mx)
}

type indexerImpl struct{}

func (i indexerImpl) GetIndex(ctx context.Context, call proto.Indexer_getIndex) error {
	return errors.ErrUnsupported
}

func (i indexerImpl) GetMeta(ctx context.Context, call proto.Indexer_getMeta) error {
	return errors.ErrUnsupported
}

func (i indexerImpl) Submit(ctx context.Context, call proto.Indexer_submit) error {
	return errors.ErrUnsupported
}

type matcherImpl struct{}

func (m matcherImpl) Match(ctx context.Context, call proto.Matcher_match) error {
	return errors.ErrUnsupported
}
