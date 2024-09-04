package dma

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gbkr-com/mkt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRequestLifecycle(t *testing.T) {

	open := &OpenOrder{
		OrderID:     mkt.NewOrderID(),
		Side:        mkt.Buy,
		Symbol:      "A",
		OrderQty:    decimal.New(100, 0),
		Price:       decimal.New(42, 0),
		TimeInForce: mkt.GTC,
	}

	assert.Nil(t, open.MakeReplaceRequest(nil, nil))
	assert.Nil(t, open.MakeCancelRequest())

	nr := open.MakeNewRequest()
	assert.NotNil(t, nr)
	assert.NotEqual(t, "", nr.ClOrdID)
	assert.True(t, nr.OrderQty.Equal(open.OrderQty))
	assert.True(t, nr.Price.Equal(open.Price))
	assert.NotNil(t, open.PendingNew)

	ORDERID := mkt.NewOrderID()
	nr.Accept(ORDERID)

	assert.Equal(t, ORDERID, open.SecondaryOrderID)
	assert.Equal(t, open.ClOrdID, nr.ClOrdID)
	assert.Nil(t, open.PendingNew)

	assert.Nil(t, open.MakeNewRequest())
	assert.Nil(t, open.MakeReplaceRequest(nil, nil))

	PRICE := decimal.New(43, 0)
	rr := open.MakeReplaceRequest(nil, &PRICE)
	assert.NotNil(t, nr)
	assert.NotEqual(t, "", rr.ClOrdID)
	assert.Equal(t, open.ClOrdID, rr.OrigClOrdID)
	assert.Nil(t, rr.OrderQty)
	assert.NotNil(t, rr.Price)
	assert.True(t, PRICE.Equal(*rr.Price))
	assert.NotNil(t, open.PendingReplace)

	rr.Reject()

	assert.Nil(t, open.PendingReplace)

	rr = open.MakeReplaceRequest(nil, &PRICE)
	ORDERID = mkt.NewOrderID()
	rr.Accept(ORDERID)

	assert.Equal(t, ORDERID, open.SecondaryOrderID)
	assert.Equal(t, open.ClOrdID, rr.ClOrdID)
	assert.True(t, open.Price.Equal(*rr.Price))
	assert.Nil(t, open.PendingReplace)

}

func TestNewRequestToFIX(t *testing.T) {

	open := &OpenOrder{
		Side:        mkt.Buy,
		Symbol:      "A",
		OrderQty:    decimal.New(100, 0),
		Price:       decimal.New(42, 0),
		TimeInForce: mkt.GTC,
	}
	request := open.MakeNewRequest()
	assert.NotNil(t, request)

	message := request.AsQuickFIX()

	msg := message.String()
	msg = strings.ReplaceAll(msg, "\u0001", "|")
	fmt.Println(msg)

}
