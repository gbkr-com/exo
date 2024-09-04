package main

import (
	"testing"

	"github.com/gbkr-com/mkt"
	"github.com/quickfixgo/field"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDecimals(t *testing.T) {

	qty := decimal.New(1, -2)
	field := field.NewOrderQty(qty, mkt.Precision(qty))

	assert.Equal(t, "0.01", field.String())

}
