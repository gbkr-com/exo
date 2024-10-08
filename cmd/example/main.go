// Package main is an example.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/exo/dma/coinbase"
	"github.com/gbkr-com/exo/env"
	"github.com/gbkr-com/exo/run"
	"github.com/gbkr-com/mkt"
	"github.com/gbkr-com/utl"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {

	url, rate, address, redisAddress := configure()

	ctx, cxl := context.WithCancel(context.Background())
	var shutdown sync.WaitGroup

	//
	// Delegates.
	//
	rdb := redis.NewClient(
		&redis.Options{
			Addr: redisAddress,
		},
	)
	factory := &DelegateFactory{rdb: rdb}

	instructions := make(chan *Order, 1)
	reports := make(chan *mkt.Report, 1)

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

	dispatcher := run.NewDispatcher[*Order](
		instructions,
		factory,
		run.ConflateTicker,
		reports,
		subscriber,
		quoteQueue,
		tradeQueue,
		func(orderID string, err error) {
			os.Stderr.WriteString(fmt.Sprintf("OrderID %s error %s", orderID, err.Error()))
		},
		rdb,
	)

	shutdown.Add(1)
	go dispatcher.Run(ctx, &shutdown)

	handler := &Handler{
		rdb:          rdb,
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

	<-env.Signal()
	fmt.Println("")
	cxl()
	shutdown.Wait()
	fmt.Println("done")

}

func configure() (url string, rate int, address string, redisAddress string) {
	url = env.MustHave("URL")
	x := env.MustHave("RATE")
	var err error
	rate, err = strconv.Atoi(x)
	if err != nil {
		os.Stderr.WriteString("bad RATE")
		os.Exit(1)
	}
	address = env.MustHave("HTTP")
	redisAddress = env.MustHave("REDIS")
	return
}
