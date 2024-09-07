package run

import (
	"fmt"
	"slices"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
)

type mockSubscriber struct {
	subs    []string
	working sync.WaitGroup
}

func (x *mockSubscriber) Subscribe(symbol string) {
	x.subs = append(x.subs, symbol)
	x.working.Done()
}

func (x *mockSubscriber) Unsubscribe(symbol string) {
	x.subs = slices.DeleteFunc(x.subs, func(s string) bool { return s == symbol })
	x.working.Done()
}

// -----------------------------------------------------------------------------

type mockDelegateFactory[T mkt.AnyOrder] struct {
	printing bool
	out      chan struct{}
}

func (x *mockDelegateFactory[T]) New(T) Delegate[T] {
	return &mockDelegate[T]{
		printing: x.printing,
		out:      x.out,
	}
}

type mockDelegate[T mkt.AnyOrder] struct {
	printing bool
	out      chan struct{}
}

func (x *mockDelegate[T]) Action(upd *Ticker, _ []redis.XMessage, _ []*mkt.Report) bool {
	defer func() {
		if x.out != nil {
			x.out <- struct{}{}
		}
	}()
	if upd == nil {
		return false
	}
	if upd.Quote != nil {
		if x.printing {
			fmt.Println(upd.Quote)
		}
	}
	if upd.Trade != nil {
		if x.printing {
			fmt.Println(upd.Trade)
		}
	}
	return false
}

func (x *mockDelegate[T]) CleanUp() {}
