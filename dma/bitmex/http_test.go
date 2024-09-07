package bitmex

import (
	"net/http/httputil"
	"os"
	"testing"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRequests(t *testing.T) {

	t.Skip()

	err := env.Load("test.env")
	assert.Nil(t, err)

	open := &dma.OpenOrder{
		Side:        mkt.Sell,
		Symbol:      "XBTUSD",
		OrderQty:    decimal.New(1, -2),
		Price:       decimal.New(52000, 0),
		TimeInForce: mkt.GTC,
	}
	request := open.MakeNewRequest()

	req, err := NewOrder(request, OrderTestURL, os.Getenv("APIKEY"), os.Getenv("SECRET"))
	assert.Nil(t, err)

	_, err = httputil.DumpRequest(req, true)
	assert.Nil(t, err)
	// fmt.Println(string(b))

}
