// Package main is an example.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/dma/coinbase"
	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
)

func main() {

	url, rate, symbol := configure()

	ctx, cxl := context.WithCancel(context.Background())
	var shutdown sync.WaitGroup

	instructions := make(chan *mkt.Order, 1)

	// Market data.
	quoteQueue := utl.NewConflatingQueue(mkt.QuoteKey)
	onQuote := run.SubscriberQuoteQueueConnector(quoteQueue)
	tradeQueue := utl.NewConflatingQueue(mkt.TradeKey, utl.WithConflateOption[string](run.ConflateTrade))
	onTrade := run.SubscriberTradeQueueConnector(tradeQueue)
	subscriber := dma.NewSubscriber(
		url,
		coinbase.Factory,
		onQuote,
		onTrade,
		func(x error) { os.Stderr.WriteString(x.Error()) },
		utl.NewRateLimiter(rate, time.Second),
		time.Hour,
	)

	dispatcher := run.NewDispatcher(instructions, &delegateFactory{}, run.ConflateComposite, subscriber, quoteQueue, tradeQueue)

	shutdown.Add(1)
	go dispatcher.Run(ctx, &shutdown)

	// Submit an order.
	orderID := mkt.NewOrderID()
	side := mkt.Buy
	fmt.Printf("%s %s\n", side.String(), symbol)
	instructions <- &mkt.Order{
		MsgType: mkt.OrderNew,
		OrderID: orderID,
		Side:    mkt.Buy,
		Symbol:  symbol,
	}

	<-time.After(10 * time.Second)

	instructions <- &mkt.Order{
		MsgType: mkt.OrderCancel,
		OrderID: orderID,
		Side:    mkt.Buy,
		Symbol:  symbol,
	}

	<-time.After(3 * time.Second)
	fmt.Println("shutdown")
	cxl()
	shutdown.Wait()
	fmt.Println("done")

}

func configure() (url string, rate int, symbol string) {
	url = os.Getenv("URL")
	if url == "" {
		os.Stderr.WriteString("missing URL")
		os.Exit(1)
	}
	x := os.Getenv("RATE")
	if x == "" {
		os.Stderr.WriteString("missing RATE")
		os.Exit(1)
	}
	var err error
	rate, err = strconv.Atoi(x)
	if err != nil {
		os.Stderr.WriteString("bad RATE")
		os.Exit(1)
	}
	symbol = os.Getenv("SYMBOL")
	if symbol == "" {
		os.Stderr.WriteString("missing SYMBOL")
		os.Exit(1)
	}
	return
}
