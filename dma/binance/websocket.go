package binance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

// Factory is the [dma.ConnectionFactory] for a [Connection].
func Factory(
	url string,
	symbol string,
	onQuote func(*mkt.Quote),
	onTrade func(*mkt.Trade),
	onError func(error),
	limiter *utl.RateLimiter,
	lifetime time.Duration,
) *Connection {
	return &Connection{
		url:      url,
		symbol:   symbol,
		onQuote:  onQuote,
		onTrade:  onTrade,
		onError:  onError,
		limiter:  limiter,
		lifetime: lifetime,
	}
}

// Connection wraps a Binance websocket connection.
type Connection struct {
	url      string
	symbol   string
	onQuote  func(*mkt.Quote)
	onTrade  func(*mkt.Trade)
	onError  func(error)
	limiter  *utl.RateLimiter
	lifetime time.Duration

	conn *websocket.Conn
	ctx  context.Context
	cxl  context.CancelFunc
	exit *sync.WaitGroup
}

// OpenWebSocket opens the connection.
func (x *Connection) OpenWebSocket() {

	x.limiter.Block()

	x.ctx, x.cxl = context.WithCancel(context.Background())
	x.exit = &sync.WaitGroup{}

	if err := x.connect(); err != nil {
		x.onError(err)
		return
	}

	b, err := x.subscribeRequest()
	if err != nil {
		x.onError(err)
		return
	}
	if err = x.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		x.onError(err)
		return
	}

	x.exit.Add(1)
	go x.listen()

}

// CloseWebSocket closes the connection.
func (x *Connection) CloseWebSocket() {

	x.limiter.Block()

	x.cxl()
	x.exit.Wait()
}

func (x *Connection) connect() error {
	var (
		response *http.Response
		err      error
	)
	dialer := &websocket.Dialer{}
	x.conn, response, err = dialer.Dial(x.url, http.Header{})
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("Connection: StatusCode: %d", response.StatusCode)
	}
	return nil
}

// Request is a stream request.
type Request struct {
	Method string   `json:"method"` // "SUBSCRIBE" or "UNSUBSCRIBE".
	Params []string `json:"params"` // Example "btcusdt@ticker" - the symbol must be lower case.
	ID     int64    `json:"id"`     // Unique per request.
}

func (x *Connection) subscribeRequest() ([]byte, error) {
	sym := strings.ToLower(x.symbol)
	msg := &Request{
		Method: "SUBSCRIBE",
		Params: []string{sym + "@bookTicker", sym + "@trade"},
		ID:     1,
	}
	return json.Marshal(&msg)
}

func (x *Connection) unsubscribeRequest() ([]byte, error) {
	sym := strings.ToLower(x.symbol)
	msg := &Request{
		Method: "UNSUBSCRIBE",
		Params: []string{sym + "@bookTicker", sym + "@trade"},
		ID:     2,
	}
	return json.Marshal(&msg)
}

// Response to a non-query request.
type Response struct {
	Result *string `json:"result,omitempty"` // nil means success.
	ID     int64   `json:"id"`               //
}

func (x *Connection) listen() {

	var reconnecting bool

	defer func() {

		b, _ := x.unsubscribeRequest()
		x.conn.WriteMessage(websocket.TextMessage, b)

		x.conn.Close()
		x.exit.Done()

		if reconnecting {
			x.OpenWebSocket()
		}

	}()

	c := time.After(x.lifetime)

	messages := make(chan []byte, 16)
	go dma.ReadWebSocket(x.conn, messages)

	for {

		select {
		case <-x.ctx.Done():
			return
		case <-c:
			reconnecting = true
			return
		case b := <-messages:
			if bytes.HasPrefix(b, []byte(`{"result":`)) {
				continue
			}
			quote, trade, err := parse(b)
			if err != nil {
				x.onError(err)
				return
			}

			if quote != nil {
				x.onQuote(quote)
			}

			if trade != nil {
				x.onTrade(trade)
			}
		}

	}

}

// Ticker is the composite of the messages from the two streams.
type Ticker struct {
	Symbol  string `json:"s"`
	BidPx   string `json:"b"` // bookTicker
	BidSize string `json:"B"` // bookTicker
	AskPx   string `json:"a"` // bookTicker
	AskSize string `json:"A"` // bookTicker
	LastQty string `json:"q"` // trade
	LastPx  string `json:"p"` // trade
}

func parse(b []byte) (*mkt.Quote, *mkt.Trade, error) {

	var ticker Ticker
	if err := json.Unmarshal(b, &ticker); err != nil {
		return nil, nil, err
	}

	var err error

	if ticker.BidPx != "" {
		quote := &mkt.Quote{Symbol: ticker.Symbol}
		if quote.BidPx, err = decimal.NewFromString(ticker.BidPx); err != nil {
			return nil, nil, fmt.Errorf("@bookTicker: b: %w", err)
		}
		if quote.BidSize, err = decimal.NewFromString(ticker.BidSize); err != nil {
			return nil, nil, fmt.Errorf("@bookTicker: B: %w", err)
		}
		if quote.AskPx, err = decimal.NewFromString(ticker.AskPx); err != nil {
			return nil, nil, fmt.Errorf("@bookTicker: a: %w", err)
		}
		if quote.AskSize, err = decimal.NewFromString(ticker.AskSize); err != nil {
			return nil, nil, fmt.Errorf("@bookTicker: A: %w", err)
		}
		return quote, nil, nil
	}

	trade := &mkt.Trade{Symbol: ticker.Symbol}
	if trade.LastQty, err = decimal.NewFromString(ticker.LastQty); err != nil {
		return nil, nil, fmt.Errorf("@trade: q: %w", err)
	}
	if trade.LastPx, err = decimal.NewFromString(ticker.LastPx); err != nil {
		return nil, nil, fmt.Errorf("@trade: p: %w", err)
	}
	return nil, trade, nil
}
