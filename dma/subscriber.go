package dma

import (
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma/binance"
	"github.com/gbkr-com/exo/dma/bitmex"
	"github.com/gbkr-com/exo/dma/coinbase"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

// Connectable defines the implemented websocket connections.
type Connectable interface {
	*binance.Connection | *bitmex.Connection | *coinbase.Connection
	Open()
	Close()
}

// A ConnectionFactory manufactures a connection.
type ConnectionFactory[T Connectable] func(
	url string,
	symbol string,
	onQuote func(*mkt.Quote),
	onTrade func(*mkt.Trade),
	onError func(error),
	limiter *utl.RateLimiter,
	lifetime time.Duration,
) T

// Subscriber for a specific exchange.
type Subscriber[T Connectable] struct {
	url           string
	onQuote       func(*mkt.Quote)
	onTrade       func(*mkt.Trade)
	onError       func(error)
	limiter       *utl.RateLimiter
	lifetime      time.Duration
	factory       ConnectionFactory[T]
	subscriptions map[string]T
	lock          sync.Mutex
}

// NewSubscriber returns a [*Subscriber] ready to use.
func NewSubscriber[T Connectable](
	url string,
	factory ConnectionFactory[T],
	onQuote func(*mkt.Quote),
	onTrade func(*mkt.Trade),
	onError func(error),
	limiter *utl.RateLimiter,
	lifetime time.Duration,
) *Subscriber[T] {
	return &Subscriber[T]{
		url:           url,
		factory:       factory,
		onQuote:       onQuote,
		onTrade:       onTrade,
		onError:       onError,
		limiter:       limiter,
		lifetime:      lifetime,
		subscriptions: make(map[string]T),
	}
}

// Subscribe to the given symbol.
func (x *Subscriber[T]) Subscribe(symbol string) {

	if symbol == "" {
		return
	}

	x.lock.Lock()
	defer x.lock.Unlock()

	if _, ok := x.subscriptions[symbol]; ok {
		return
	}

	conn := x.factory(x.url, symbol, x.onQuote, x.onTrade, x.onError, x.limiter, x.lifetime)
	conn.Open()
	x.subscriptions[symbol] = conn

}

// Unsubscribe to the given symbol.
func (x *Subscriber[T]) Unsubscribe(symbol string) {

	if symbol == "" {
		return
	}

	x.lock.Lock()
	defer x.lock.Unlock()

	conn, ok := x.subscriptions[symbol]
	if !ok {
		return
	}

	conn.Close()
	delete(x.subscriptions, symbol)

}
