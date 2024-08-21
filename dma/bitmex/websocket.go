package bitmex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

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

// Connection wraps a BitMex websocket connection.
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

// Command is a stream request.
type Command struct {
	Op   string   `json:"op"`   // "subscribe" or "unsubscribe"
	Args []string `json:"args"` //
}

func (x *Connection) subscribeRequest() ([]byte, error) {
	msg := &Command{
		Op:   "subscribe",
		Args: []string{"quote:" + x.symbol, "trade:" + x.symbol},
	}
	return json.Marshal(&msg)
}

func (x *Connection) unsubscribeRequest() ([]byte, error) {
	msg := &Command{
		Op:   "unsubscribe",
		Args: []string{"quote:" + x.symbol, "trade:" + x.symbol},
	}
	return json.Marshal(&msg)
}

func (x *Connection) listen() {

	var (
		// lastTradeID  int64
		reconnecting bool
	)

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

		if bytes.HasPrefix(b, []byte(`{"table":"quote"`)) {
			quote, err := parseQuote(b)
			if err != nil {
				x.onError(err)
				return
			}
			if quote != nil {
				x.onQuote(quote)
			}
			continue
		}

		if bytes.HasPrefix(b, []byte(`{"table":"trade"`)) {
			trades, err := parseTrade(b)
			if err != nil {
				x.onError(err)
				return
			}
			if trades != nil {
				for _, v := range trades {
					x.onTrade(v)
				}
			}
			continue
		}

	}

}

// A Quote table update.
type Quote struct {
	Data []struct {
		Symbol  string  `json:"symbol"`
		BidSize float64 `json:"bidSize"`
		BidPx   float64 `json:"bidPrice"`
		AskPx   float64 `json:"askPrice"`
		AskSize float64 `json:"askSize"`
	} `json:"data"`
}

func parseQuote(b []byte) (*mkt.Quote, error) {

	var data Quote
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	if len(data.Data) != 1 {
		// The subscription is for level 1 only.
		return nil, nil
	}

	row := data.Data[0]

	var quote mkt.Quote
	quote.Symbol = row.Symbol
	quote.BidPx = decimal.NewFromFloat(row.BidPx)
	quote.BidSize = decimal.NewFromFloat(row.BidSize)
	quote.AskPx = decimal.NewFromFloat(row.AskPx)
	quote.AskSize = decimal.NewFromFloat(row.AskSize)

	return &quote, nil
}

// A Trade table update.
type Trade struct {
	Data []struct {
		Symbol  string  `json:"symbol"`
		LastQty float64 `json:"size"`
		LastPx  float64 `json:"price"`
	} `json:"data"`
}

func parseTrade(b []byte) ([]*mkt.Trade, error) {

	var data Trade
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	trades := []*mkt.Trade{}
	for _, v := range data.Data {
		trade := &mkt.Trade{
			Symbol:  v.Symbol,
			LastQty: decimal.NewFromFloat(v.LastQty),
			LastPx:  decimal.NewFromFloat(v.LastPx),
		}
		trades = append(trades, trade)
	}

	return trades, nil
}
