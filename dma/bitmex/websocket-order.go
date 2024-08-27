package bitmex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/utl"
	"github.com/gorilla/websocket"
)

// OrderConnection is a web socket connection for order data on BitMex.
type OrderConnection struct {
	url      string
	apiKey   string
	secret   string
	onError  func(error)
	limiter  *utl.RateLimiter
	lifetime time.Duration

	conn *websocket.Conn
	ctx  context.Context
	cxl  context.CancelFunc
	exit *sync.WaitGroup
}

// OpenWebSocket opens the connection.
func (x *OrderConnection) OpenWebSocket() {

	x.limiter.Block()

	x.ctx, x.cxl = context.WithCancel(context.Background())
	x.exit = &sync.WaitGroup{}

	if err := x.connect(); err != nil {
		x.onError(err)
		return
	}

	b, err := x.subscribeRequest()
	if err != nil {
		x.onError(err)
		return
	}
	if err = x.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		x.onError(err)
		return
	}

	x.exit.Add(1)
	go x.listen()

}

// CloseWebSocket closes the connection.
func (x *OrderConnection) CloseWebSocket() {

	x.limiter.Block()

	x.cxl()
	x.exit.Wait()

}

func (x *OrderConnection) connect() error {
	var (
		response *http.Response
		err      error
	)

	dialer := &websocket.Dialer{}
	x.conn, response, err = dialer.Dial(x.url, http.Header{})
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("Connection: StatusCode: %d", response.StatusCode)
	}

	expires := strconv.FormatInt(time.Now().Unix()+RequestExpirySeconds, 10)
	signature := sign(http.MethodGet, x.url, expires, []byte(""), x.secret)

	msg := &Command{
		Op:   "authKeyExpires",
		Args: []string{x.apiKey, expires, signature},
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return x.conn.WriteMessage(websocket.TextMessage, b)

}

func (x *OrderConnection) subscribeRequest() ([]byte, error) {
	msg := &Command{
		Op:   "subscribe",
		Args: []string{"order", "execution"},
	}
	return json.Marshal(&msg)
}

func (x *OrderConnection) unsubscribeRequest() ([]byte, error) {
	msg := &Command{
		Op:   "unsubscribe",
		Args: []string{"quote", "trade"},
	}
	return json.Marshal(&msg)
}

func (x *OrderConnection) listen() {

	var reconnecting bool

	defer func() {

		b, _ := x.unsubscribeRequest()
		x.conn.WriteMessage(websocket.TextMessage, b)

		x.conn.Close()
		x.exit.Done()

		if reconnecting {
			x.OpenWebSocket()
		}

	}()

	c := time.After(x.lifetime)

	messages := make(chan []byte, 16)
	go dma.ReadWebSocket(x.conn, messages)

	for {

		select {
		case <-x.ctx.Done():
			return
		case <-c:
			reconnecting = true
			return
		case b := <-messages:
			if bytes.HasPrefix(b, []byte(`{"table":"order"`)) {
				fmt.Println(string(b))
			}
			if bytes.HasPrefix(b, []byte(`{"table":"execution"`)) {
				fmt.Println(string(b))
			}
		}

	}

}
