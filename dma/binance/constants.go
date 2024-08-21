package binance

import "github.com/gbkr-com/mkt"

// Connection parameters for Binance. These are only provided for
// convenience in testing - operational values should be in the environment.
const (
	WebSocketURL               = "wss://data-stream.binance.vision/ws/ticker"
	WebSocketRequestsPerSecond = 5
)

// FIX constants for the Spot market.
const (
	FIXTimeStamp = mkt.FIXUTCMicros
)
