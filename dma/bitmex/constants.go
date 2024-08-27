package bitmex

// Connection parameters for BitMex. These are only provided for
// convenience in testing - operational values should be in the environment.
const (
	WebSocketURL             = "wss://ws.bitmex.com/realtime"
	WebSocketTestURL         = "wss://ws.testnet.bitmex.com/realtime"
	WebSocketRequestsPerHour = 720
)

// Constants for the BitMex HTTP interface.
const (
	OrderTestURL         = "https://testnet.bitmex.com/api/v1/order"
	RequestExpirySeconds = 5
)
