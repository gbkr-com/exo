package binance

// Request is a stream request.
type Request struct {
	Method string   `json:"method"` // "SUBSCRIBE" or "UNSUBSCRIBE".
	Params []string `json:"params"` // Example "btcusdt@ticker".
	ID     string   // Unique per request.
}

// Ticker is the message for an individual symbol ticker stream.
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
