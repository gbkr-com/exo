package binance

import (
	"fmt"
	"os"
	"testing"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {

	err := env.Load("test.env")
	assert.Nil(t, err)

	open := &dma.OpenOrder{
		Side:        mkt.Sell,
		Symbol:      "BTCUSDT",
		OrderQty:    decimal.New(1, -2),
		Price:       decimal.New(52000, 0),
		TimeInForce: mkt.GTC,
	}
	request := open.MakeNewRequest()

	b, err := NewRequestFrame(request, os.Getenv("APIKEY"), os.Getenv("SECRET"))
	assert.Nil(t, err)

	fmt.Println(string(b))

	request.Accept("ABC")
	cxl := open.MakeCancelRequest()

	b, err = CancelRequestFrame(cxl, os.Getenv("APIKEY"), os.Getenv("SECRET"))
	assert.Nil(t, err)

	fmt.Println(string(b))
}
