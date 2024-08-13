package coinbase

import (
	"sync"

	"github.com/gbkr-com/mkt"
)

// A Subscriber to the Coinbase Exchange ticker.
type Subscriber struct {
	url           string
	subscriptions map[string]*Connection
	onQuote       func(*mkt.Quote)
	onTrade       func(*mkt.Trade)
	onError       func(error)
	lock          sync.Mutex
}

// NewSubscriber returns a [*Subscriber] ready to use.
func NewSubscriber(url string, onQuote func(*mkt.Quote), onTrade func(*mkt.Trade), onError func(error)) *Subscriber {
	return &Subscriber{
		url:           url,
		subscriptions: make(map[string]*Connection),
		onQuote:       onQuote,
		onTrade:       onTrade,
		onError:       onError,
	}
}

// Subscribe to the given symbol.
func (x *Subscriber) Subscribe(symbol string) {

	if symbol == "" {
		return
	}

	x.lock.Lock()
	defer x.lock.Unlock()

	if _, ok := x.subscriptions[symbol]; ok {
		return
	}

	conn := &Connection{
		url:     x.url,
		symbol:  symbol,
		onQuote: x.onQuote,
		onTrade: x.onTrade,
		onError: x.onError,
	}
	conn.Open()
	x.subscriptions[symbol] = conn

}

// Unsubscribe to the given symbol.
func (x *Subscriber) Unsubscribe(symbol string) {

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
