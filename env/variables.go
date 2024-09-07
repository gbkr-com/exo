package env

import (
	"time"
)

// RedisXReadTimeout is the timeout when reading a Redis stream.
var RedisXReadTimeout = time.Millisecond

// RunHandlerTimeout is the maximum duration to wait for ticker data before
// reading the instruction and report streams.
var RunHandlerTimeout = time.Second

// DefaultDecimalPlaces is the default precision when calculating with
// [decimal.Decimal].
var DefaultDecimalPlaces int32 = 8
