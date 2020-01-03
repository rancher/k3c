package kicker

import (
	"context"
	"sync"
)

type Kicker interface {
	Kicked() bool
	Kick()
}

type kicker struct {
	kicked bool
	ctx    context.Context
	c      sync.Cond
}

func (k *kicker) Kicked() bool {
	k.c.L.Lock()
	for !k.kicked {
		k.c.Wait()
	}
	k.c.L.Unlock()

	select {
	case <-k.ctx.Done():
		return false
	default:
		return true
	}
}

func (k *kicker) Kick() {
	k.c.L.Lock()
	k.kicked = true
	k.c.Broadcast()
	k.c.L.Unlock()
}

func New(ctx context.Context, initialValue bool) Kicker {
	k := &kicker{
		ctx: ctx,
		c: sync.Cond{
			L: &sync.Mutex{},
		},
		kicked: initialValue,
	}

	go func() {
		<-ctx.Done()
		k.Kick()
	}()

	return k
}
