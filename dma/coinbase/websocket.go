package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gbkr-com/mkt"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

// Connection wraps a Coinbase websocket connection.
type Connection struct {
	url     string
	symbol  string
	onQuote func(*mkt.Quote)
	onTrade func(*mkt.Trade)
	onError func(error)

	conn *websocket.Conn
	ctx  context.Context
	cxl  context.CancelFunc
	exit *sync.WaitGroup
}

// Open the connection.
func (x *Connection) Open() {

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

	x.cxl()
	x.exit.Wait()

	b, _ := x.unsubscribeRequest()
	x.conn.WriteMessage(websocket.TextMessage, b)

	x.conn.Close()
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

// ExchangeRequest is a Coinbase Exchange websocket request.
type ExchangeRequest struct {
	Type       string   `json:"type"`
	ProductIDs []string `json:"product_ids"`
	Channels   []string `json:"channels"`
}

func (x *Connection) subscribeRequest() ([]byte, error) {
	msg := &ExchangeRequest{
		Type:       "subscribe",
		ProductIDs: []string{x.symbol},
		Channels:   []string{"ticker"},
	}
	return json.Marshal(&msg)
}

func (x *Connection) unsubscribeRequest() ([]byte, error) {
	msg := &ExchangeRequest{
		Type:       "unsubscribe",
		ProductIDs: []string{x.symbol},
		Channels:   []string{"ticker"},
	}
	return json.Marshal(&msg)
}

// ExchangeMessageType is a minimal Coinbase Exchange websocket message.
type ExchangeMessageType struct {
	Type    string `json:"type"`    // Values are "ticker" or "error".
	Message string `json:"message"` // Present when type is "error".
}

func (x *Connection) listen() {

	defer x.exit.Done()

	var lastTradeID int64

	for {

		select {
		case <-x.ctx.Done():
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

		var mt ExchangeMessageType
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

		x.onQuote(quote)

		if tradeID != lastTradeID {
			x.onTrade(trade)
			lastTradeID = tradeID
		}
	}

}

// ExchangeTicker is a Coinbase Exchange websocket ticker message.
type ExchangeTicker struct {
	ProductID   string `json:"product_id"`
	BestBid     string `json:"best_bid"`
	BestBidSize string `json:"best_bid_size"`
	BestAsk     string `json:"best_ask"`
	BestAskSize string `json:"best_ask_size"`
	TradeID     int64  `json:"trade_id"`
	LastSize    string `json:"last_size"`
	Price       string `json:"price"`
}

func parse(b []byte) (*mkt.Quote, *mkt.Trade, int64, error) {

	var ticker ExchangeTicker
	if err := json.Unmarshal(b, &ticker); err != nil {
		return nil, nil, 0, err
	}

	var (
		err   error
		quote mkt.Quote
		trade mkt.Trade
	)

	quote.Symbol = ticker.ProductID

	if quote.BidPx, err = decimal.NewFromString(ticker.BestBid); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_bid: %w", err)
	}
	if quote.BidSize, err = decimal.NewFromString(ticker.BestBidSize); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_bid_size: %w", err)
	}
	if quote.AskPx, err = decimal.NewFromString(ticker.BestAsk); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_ask: %w", err)
	}
	if quote.AskSize, err = decimal.NewFromString(ticker.BestAskSize); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: best_ask_size: %w", err)
	}

	trade.Symbol = ticker.ProductID

	if trade.LastQty, err = decimal.NewFromString(ticker.LastSize); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: last_size: %w", err)
	}
	if trade.LastPx, err = decimal.NewFromString(ticker.Price); err != nil {
		return nil, nil, 0, fmt.Errorf("ticker: price: %w", err)
	}

	return &quote, &trade, ticker.TradeID, nil
}
