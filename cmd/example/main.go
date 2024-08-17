// Package main is an example.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/dma/coinbase"
	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/gin-gonic/gin"
)

func main() {

	url, rate, address := configure()

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

	handler := &Handler{
		orders:       map[string]*mkt.Order{},
		instructions: instructions,
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	handler.Bind(router)
	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}
	go srv.ListenAndServe()

	<-Signal()
	fmt.Println("")
	cxl()
	shutdown.Wait()
	fmt.Println("done")

}

func configure() (url string, rate int, address string) {
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
	address = os.Getenv("HTTP")
	if address == "" {
		os.Stderr.WriteString("missing HTTP")
		os.Exit(1)
	}
	return
}

// Signal for termination.
func Signal() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}
