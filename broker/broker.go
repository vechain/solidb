// Package broker proxies requests from client, and routes to target nodes.
package broker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/blob"
	"github.com/vechain/solidb/blobio"
	"github.com/vechain/solidb/kv"
	"github.com/vechain/solidb/node"
	"github.com/vechain/solidb/quorum"
	"github.com/vechain/solidb/spec"
	"github.com/vechain/solidb/specmgr"
	"github.com/vechain/solidb/utils/httpx"
)

// result implements quorum.Vote
type result struct {
	entry *spec.Entry
	blob  *blob.Blob
	err   error
}

func (r *result) Errored() bool {
	return r.err != nil
}

func (r *result) Data() interface{} {
	return r.blob
}

// Broker broker is entry to access solidb
type Broker struct {
	store   kv.Store
	specMgr *specmgr.SpecManager
	nodeRPC *node.RPC
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New create an broker instance
func New(store kv.Store, specMgr *specmgr.SpecManager) *Broker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broker{
		store:   store,
		specMgr: specMgr,
		nodeRPC: node.NewRPC().WithContext(ctx),
		cancel:  cancel,
	}
}

// Shutdown stop all goroutines and wait for stopped
func (b *Broker) Shutdown() {
	b.cancel()
	b.wg.Wait()
}

func (b *Broker) run(f func()) {
	go func() {
		b.wg.Add(1)
		defer b.wg.Done()
		f()
	}()
}

// GetBlob get blob by its key
func (b *Broker) GetBlob(ctx context.Context, key blob.Key) (*blobio.OptBlob, error) {
	approved, err := b.specMgr.GetByTag(specmgr.TagApproved)
	if err != nil {
		return nil, err
	}
	if approved.V == nil {
		return nil, errors.New("no approved spec")
	}

	entries := approved.V.SAT.Locate(key.ToHex())
	ch := make(chan quorum.Vote, len(entries))
	for _, entry := range entries {
		entry := entry
		b.run(func() {
			defer func() {
				if err := recover(); err != nil {
					e := errors.Errorf("get blob: goroutine recovered %v", err)
					log.Warnln(e)
					ch <- &result{err: e}
				}
			}()
			r := result{entry: &entry}
			rpc := b.nodeRPC.WithAddr(entry.Addr)
			blob, err := rpc.GetBlob(key)
			if err != nil {
				r.err = err
				if !httpx.IsCausedByContextCanceled(err) {
					log.Warnf("Get blob from node %v: %v", entry, err)
				}
			} else {
				r.blob = blob.V
			}
			ch <- &r
		})
	}

	data, err := quorum.HandleRead(ctx, ch, len(entries))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return &blobio.OptBlob{}, nil
	}
	return &blobio.OptBlob{V: data.(*blob.Blob)}, nil
}

// PutBlob store a blob
func (b *Broker) PutBlob(ctx context.Context, blob *blob.Blob) error {
	key := blob.Key()
	approved, err := b.specMgr.GetByTag(specmgr.TagApproved)
	if err != nil {
		return err
	}
	if approved.V == nil {
		return errors.New("no approved spec")
	}
	newest, err := b.specMgr.GetNewest()
	if err != nil {
		return err
	}
	if newest.V == nil {
		return errors.New("no spec found")
	}

	approvedEntries := approved.V.SAT.Locate(key.ToHex())
	ch := make(chan *result, len(approvedEntries))
	qch := make(chan quorum.Vote, len(approvedEntries))
	for _, entry := range approvedEntries {
		entry := entry
		b.run(func() {
			defer func() {
				if err := recover(); err != nil {
					e := errors.Errorf("put blob: goroutine recovered %v", err)
					log.Warnln(e)
					ch <- &result{err: e}

				}
			}()

			r := result{entry: &entry}
			rpc := b.nodeRPC.WithAddr(entry.Addr)
			if err := rpc.PutBlob(blob); err != nil {
				r.err = err
				if !httpx.IsCausedByContextCanceled(err) {
					log.Warnf("Put blob to node %v, error: %v", entry, err)
				}
			}
			ch <- &r
			qch <- &r
		})
	}

	if err := quorum.HandleWrite(ctx, qch, len(approvedEntries)); err != nil {
		return err
	}

	var okEntries []*spec.Entry
	for i := 0; i < cap(ch); i++ {
		r := <-ch
		if r.err == nil {
			okEntries = append(okEntries, r.entry)
		}
	}

	hasFault := len(okEntries) != len(approvedEntries)
	if !hasFault {
		var newEntries []spec.Entry
		for _, e := range newest.V.SAT.Locate(key.ToHex()) {
			if approved.V.SAT.FindEntry(e.ID) != nil {
				continue
			}
			newEntries = append(newEntries, e)
		}
		ch := make(chan bool, len(newEntries))
		for _, entry := range newEntries {
			entry := entry
			b.run(func() {
				defer func() {
					if err := recover(); err != nil {
						e := errors.Errorf("put blob: goroutine recovered %v", err)
						log.Warnln(e)
						ch <- false
					}
				}()
				rpc := b.nodeRPC.WithAddr(entry.Addr)
				if err := rpc.PutBlob(blob); err != nil {
					ch <- false
					if !httpx.IsCausedByContextCanceled(err) {
						log.Warnf("Put blob to node %v: %v", entry, err)
					}
				} else {
					ch <- true
				}
			})
		}

		for i := 0; i < cap(ch); i++ {
			if r := <-ch; !r {
				hasFault = true
				break
			}
		}
	}
	if hasFault {
		if err := blobio.MarkBlob(b.store, key, blobio.FaultBlobMark); err != nil {
			return err
		}
	}
	return nil
}
