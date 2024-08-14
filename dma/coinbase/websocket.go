package coinbase

import (
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

// Connection wraps a Coinbase websocket connection.
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

// Request is a Coinbase Exchange websocket request.
type Request struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

func (x *Connection) subscribeRequest() ([]byte, error) {
	msg := &Request{
		Type:       "subscribe",
		ProductIDs: []string{x.symbol},
		Channels:   []string{"ticker"},
	}
	return json.Marshal(&msg)
}

func (x *Connection) unsubscribeRequest() ([]byte, error) {
	msg := &Request{
		Type:       "unsubscribe",
		ProductIDs: []string{x.symbol},
		Channels:   []string{"ticker"},
	}
	return json.Marshal(&msg)
}

// MessageType is a minimal Coinbase Exchange websocket message.
type MessageType struct {
	Type    string `json:"type"`    // Values are "ticker" or "error".
	Message string `json:"message"` // Present when type is "error".
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

		var mt MessageType
		if err = json.Unmarshal(b, &mt); err != nil {
			x.onError(err)
			return
		}

		if mt.Type == "error" {
			x.onError(fmt.Errorf("ticker: %s", mt.Message))
			return
		}
		if mt.Type != "ticker" {
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

// Ticker is a Coinbase Exchange websocket ticker message.
type Ticker struct {
	Symbol  string `json:"product_id"`
	BidPx   string `json:"best_bid"`
	BidSize string `json:"best_bid_size"`
	AskPx   string `json:"best_ask"`
	AskSize string `json:"best_ask_size"`
	LastQty string `json:"last_size"`
	LastPx  string `json:"price"`
	TradeID int64  `json:"trade_id"`
}

func parse(b []byte) (*mkt.Quote, *mkt.Trade, int64, error) {

	var ticker Ticker
	if err := json.Unmarshal(b, &ticker); err != nil {
		return nil, nil, 0, err
	}

	var (
		err   error
		quote mkt.Quote
		trade mkt.Trade
	)

	quote.Symbol = ticker.Symbol

	if quote.BidPx, err = decimal.NewFromString(ticker.BidPx); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_bid: %w", err)
	}
	if quote.BidSize, err = decimal.NewFromString(ticker.BidSize); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_bid_size: %w", err)
	}
	if quote.AskPx, err = decimal.NewFromString(ticker.AskPx); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_ask: %w", err)
	}
	if quote.AskSize, err = decimal.NewFromString(ticker.AskSize); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_ask_size: %w", err)
	}

	trade.Symbol = ticker.Symbol

	if trade.LastQty, err = decimal.NewFromString(ticker.LastQty); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: last_size: %w", err)
	}
	if trade.LastPx, err = decimal.NewFromString(ticker.LastPx); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: price: %w", err)
	}

	return &quote, &trade, ticker.TradeID, nil
}
