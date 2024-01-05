package rpc

import (
	"context"
	"strconv"

	"github.com/quay/clair/v4/rpc/internal/proto"
)

type TODO any

// IndexerServer implements the [proto.Indexer_Server] interface.
type indexerServer struct {
	core TODO
}

// GetIndex implements [proto.Indexer_Server].
func (srv *indexerServer) GetIndex(ctx context.Context, call proto.Indexer_getIndex) error {
	d, err := call.Args().Manifest()
	if err != nil {
		return err
	}

	// Do some actual database work here.

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	m, err := res.NewMeta()
	if err != nil {
		return err
	}
	if err := m.SetDigest(d); err != nil {
		return err
	}
	m.Error().SetOk()
	if err := res.SetMeta(m); err != nil {
		return err
	}

	iter := proto.Index_ServerToClient(&indexServer{srv, nil})
	if err := res.SetIndex(iter); err != nil {
		return err
	}

	return nil
}

// GetMeta implements [proto.Indexer_Server].
func (*indexerServer) GetMeta(ctx context.Context, call proto.Indexer_getMeta) error {
	args := call.Args()
	d, err := args.Manifest()
	if err != nil {
		return err
	}

	// Database query here.

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	m, err := res.NewMeta()
	if err != nil {
		return err
	}

	if err := m.SetDigest(d); err != nil {
		return err
	}
	m.Error().SetUnset()
	if err := m.SetState("unimplemented"); err != nil {
		return err
	}

	if err := res.SetMeta(m); err != nil {
		return err
	}

	return nil
}

// Submit implements [proto.Indexer_Server].
func (*indexerServer) Submit(context.Context, proto.Indexer_submit) error {
	panic("unimplemented")
}

// IndexServer implements the [proto.Index_Server] interface.
//
// It's created in [(*indexerServer).GetIndex] and serves the result of a
// submission to a [proto.Indexer_Server].
type indexServer struct {
	parent     *indexerServer
	coreReport TODO
}

// AttrsFor implements [proto.Index_Server].
func (*indexServer) AttrsFor(context.Context, proto.Index_attrsFor) error {
	panic("unimplemented")
}

// Environments implements [proto.Index_Server].
func (*indexServer) Environments(context.Context, proto.Index_environments) error {
	panic("unimplemented")
}

// Packages implements [proto.Index_Server].
func (srv *indexServer) Packages(ctx context.Context, call proto.Index_packages) error {
	iter := call.Args().Cb()

	// run some incremental query, and get an iterable of values:
	for i := 0; i < 10; i++ {
		// Build the message
		var pkg proto.Package
		pkg.SetId(strconv.Itoa(i))

		// Send the message
		err := iter.Next(ctx, func(p proto.Iterator_next_Params) error {
			return p.SetItem(pkg.EncodeAsPtr(p.Segment()))
		})
		if err != nil {
			return nil
		}
	}
	fut, done := iter.Done(ctx, nil)
	defer done()
	select {
	case <-fut.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}

}

var (
	_ proto.Indexer_Server = (*indexerServer)(nil)
	_ proto.Index_Server   = (*indexServer)(nil)
)

func _() {
	proto.Indexer_ServerToClient(new(indexerServer))
}
