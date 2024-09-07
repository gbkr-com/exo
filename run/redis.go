package run

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gbkr-com/mkt"
	"github.com/redis/go-redis/v9"
)

// Prefixes for order ID based keys. The first part of the prefix is the type
// of Redis data structure, to assist when working with the Redis command line.
// The colon separator is a Redis idiom.
const (
	OrderInstructionsStreamPrefix = "stream:instructions:"
	OrderReportsStreamPrefix      = "stream:reports:"
	OrderHashPrefix               = "hash:order:"
)

// MakeOrderInstructionsStreamName is a convenience function.
func MakeOrderInstructionsStreamName(order *mkt.Order) string {
	return OrderInstructionsStreamPrefix + order.OrderID
}

// MakeOrderReportsStreamName is a convenience function.
func MakeOrderReportsStreamName(order *mkt.Order) string {
	return OrderReportsStreamPrefix + order.OrderID
}

// MakeOrderHashKey is a convenience function.
func MakeOrderHashKey(order *mkt.Order) string {
	return OrderHashPrefix + order.OrderID
}

// WriteOrderInstructions to the stream named with [MakeOrderInstructionsStreamName].
func WriteOrderInstructions[T mkt.AnyOrder](ctx context.Context, rdb *redis.Client, order T) error {
	b, err := json.Marshal(order)
	if err != nil {
		return err
	}
	args := &redis.XAddArgs{
		Stream: MakeOrderInstructionsStreamName(order.Definition()),
		Values: []any{"json", string(b)},
	}
	_, err = rdb.XAdd(ctx, args).Result()
	return err
}

// WriteOrderReport to the stream named with [MakeOrderReportsStreamName].
func WriteOrderReport(ctx context.Context, rdb *redis.Client, report *mkt.Report) error {
	b, err := json.Marshal(report)
	if err != nil {
		return err
	}
	args := &redis.XAddArgs{
		Stream: OrderReportsStreamPrefix + report.OrderID,
		Values: []any{"json", string(b)},
	}
	_, err = rdb.XAdd(ctx, args).Result()
	return err
}

// UnmarshalOrderReport translates the stream message into a [*mkt.Report].
func UnmarshalOrderReport(message redis.XMessage) (*mkt.Report, error) {

	v := message.Values
	s, ok := v["json"]
	if !ok {
		return nil, fmt.Errorf("UnmarshalOrderReport: message does not contain the 'json' field")
	}
	j, ok := s.(string)
	if !ok {
		return nil, fmt.Errorf("UnmarshalOrderReport: 'json' value is not a string")
	}
	var report mkt.Report
	if err := json.Unmarshal([]byte(j), &report); err != nil {
		return nil, fmt.Errorf("UnmarshalOrderReport: %w", err)
	}
	return &report, nil

}
