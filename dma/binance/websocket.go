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

	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

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

// Open the connection.
func (x *Connection) Open() {

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

// Close the connection.
func (x *Connection) Close() {

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
	msg := &Request{
		Method: "SUBSCRIBE",
		Params: []string{strings.ToLower(x.symbol) + "@ticker"},
		ID:     1,
	}
	return json.Marshal(&msg)
}

func (x *Connection) unsubscribeRequest() ([]byte, error) {
	msg := &Request{
		Method: "UNSUBSCRIBE",
		Params: []string{strings.ToLower(x.symbol) + "@ticker"},
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

	var (
		lastTradeID  int64
		reconnecting bool
	)

	defer func() {

		b, _ := x.unsubscribeRequest()
		x.conn.WriteMessage(websocket.TextMessage, b)

		x.conn.Close()
		x.exit.Done()

		if reconnecting {
			x.Open()
		}

	}()

	c := time.After(x.lifetime)

	for {

		select {
		case <-x.ctx.Done():
			return
		case <-c:
			reconnecting = true
			return
		default:
		}

		t, b, err := x.conn.ReadMessage()
		if err != nil {
			x.onError(err)
			return
		}
		if t != websocket.TextMessage {
			continue
		}

		if bytes.HasPrefix(b, []byte(`{"result":`)) {
			continue
		}

		if !bytes.HasPrefix(b, []byte(`{"e":"24hrTicker"`)) {
			continue
		}

		quote, trade, tradeID, err := parse(b)
		if err != nil {
			x.onError(err)
			return
		}

		if quote != nil {
			x.onQuote(quote)
		}

		if trade != nil && tradeID != lastTradeID {
			x.onTrade(trade)
			lastTradeID = tradeID
		}

	}

}

// Ticker is the message for an individual symbol ticker stream. However json.Unmarshal
// fails with real data (claiming field "L" is a string), but unmarshaling into
// a basic map does work - albeit with "L" being parsed to a float64.
// The struct is kept for reference only.
type Ticker struct {
	Symbol  string `json:"s"`
	BidPx   string `json:"b"`
	BidSize string `json:"B"`
	AskPx   string `json:"a"`
	AskSize string `json:"A"`
	LastQty string `json:"Q"`
	LastPx  string `json:"c"`
	TradeID int64  `json:"L"`
}

func parse(b []byte) (*mkt.Quote, *mkt.Trade, int64, error) {

	var ticker map[string]any
	if err := json.Unmarshal(b, &ticker); err != nil {
		return nil, nil, 0, err
	}

	var (
		s     string
		ok    bool
		err   error
		quote mkt.Quote
		trade mkt.Trade
	)

	quote.Symbol, ok = ticker["s"].(string)
	if !ok {
		return nil, nil, 0, nil
	}

	s, ok = ticker["b"].(string)
	if !ok {
		return nil, nil, 0, nil
	}
	if quote.BidPx, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: b: %w", err)
	}

	s, ok = ticker["B"].(string)
	if !ok {
		return nil, nil, 0, nil
	}
	if quote.BidSize, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: B: %w", err)
	}

	s, ok = ticker["a"].(string)
	if !ok {
		return nil, nil, 0, nil
	}
	if quote.AskPx, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: a: %w", err)
	}

	s, ok = ticker["A"].(string)
	if !ok {
		return nil, nil, 0, nil
	}
	if quote.AskSize, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: A: %w", err)
	}

	trade.Symbol = quote.Symbol

	s, ok = ticker["Q"].(string)
	if !ok {
		return &quote, nil, 0, nil
	}
	if trade.LastQty, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: Q: %w", err)
	}

	s, ok = ticker["c"].(string)
	if !ok {
		return &quote, nil, 0, nil
	}
	if trade.LastPx, err = decimal.NewFromString(s); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: c: %w", err)
	}

	f, ok := ticker["L"].(float64)
	tradeID := int64(f)

	return &quote, &trade, tradeID, nil
}
