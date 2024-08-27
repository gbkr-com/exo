package bitmex

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
)

// NewOrder translates a [*dma.NewRequest] into a BitMex new order.
func NewOrder(request *dma.NewRequest, url, apiKey, secret string) (*http.Request, error) {

	body := struct {
		Symbol      string  `json:"symbol"`
		OrderQty    float64 `json:"orderQty"`
		Price       float64 `json:"price"`
		ClOrdID     string  `json:"clOrdID"`
		OrdType     string  `json:"ordType"`     // "Limit"
		TimeInForce string  `json:"timeInForce"` // "GoodTillCancel", "ImmediateOrCancel"
	}{}
	body.Symbol = request.Symbol
	body.OrderQty = request.OrderQty.InexactFloat64()
	body.Price = request.Price.InexactFloat64()
	body.ClOrdID = request.ClOrdID
	body.OrdType = "Limit"
	if request.TimeInForce == mkt.IOC {
		body.TimeInForce = "ImmediateOrCancel"
	} else {
		body.TimeInForce = "GoodTillCancel"
	}
	b, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}

	expires := strconv.FormatInt(time.Now().Unix()+RequestExpirySeconds, 10)
	signature := sign(http.MethodPost, url, expires, b, secret)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req, expires, apiKey, signature)

	return req, nil

}

// ReplaceOrder trnalsates a [*dma.ReplaceRequest] into a BitMex amendment.
func ReplaceOrder(request *dma.ReplaceRequest, url, apiKey, secret string) (*http.Request, error) {

	body := struct {
		OrigClOrdID string   `json:"origClOrdID"`
		ClOrdID     string   `json:"clOrdID"`
		OrderQty    *float64 `json:"orderQty,omitempty"`
		Price       *float64 `json:"price,omitempty"`
	}{}
	body.OrigClOrdID = request.OrigClOrdID
	body.ClOrdID = request.ClOrdID
	if request.OrderQty != nil {
		x := request.OrderQty.InexactFloat64()
		body.OrderQty = &x
	}
	if request.Price != nil {
		x := request.Price.InexactFloat64()
		body.Price = &x
	}
	b, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}

	expires := strconv.FormatInt(time.Now().Unix()+RequestExpirySeconds, 10)
	signature := sign(http.MethodPost, url, expires, b, secret)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req, expires, apiKey, signature)

	return req, nil

}

// CancelOrder translates a [*dma.CancelRequest] into a BitMex cancellation.
func CancelOrder(request *dma.CancelRequest, url, apiKey, secret string) (*http.Request, error) {

	body := struct {
		ClOrdID string `json:"clOrdID"`
	}{}
	body.ClOrdID = request.OrigClOrdID // BitMex breaks with FIX here.
	b, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}

	expires := strconv.FormatInt(time.Now().Unix()+RequestExpirySeconds, 10)
	signature := sign(http.MethodPost, url, expires, b, secret)

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req, expires, apiKey, signature)

	return req, nil

}

func sign(verb, path, expires string, body []byte, secret string) string {

	var buffer bytes.Buffer
	buffer.WriteString(verb)
	buffer.WriteString(path)
	buffer.WriteString(expires)
	buffer.Write(body)

	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write(buffer.Bytes())
	return hex.EncodeToString(hash.Sum(nil))

}

func setRequestHeaders(request *http.Request, expires, apiKey, signature string) {
	request.Header.Set("api-expires", expires)
	request.Header.Set("api-key", apiKey)
	request.Header.Set("api-signature", signature)

}
