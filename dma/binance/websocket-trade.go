package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gbkr-com/exo/dma"
	"github.com/gbkr-com/mkt"
)

// NewRequestFrame returns the web socket frame for a [dma.NewRequest].
func NewRequestFrame(request *dma.NewRequest, apiKey, secret string) ([]byte, error) {

	now := time.Now().UnixMilli()
	payload := newRequestPayloadForSignature(request, now, apiKey)
	signature := sign(payload, secret)

	frame := struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Params struct {
			Symbol           string `json:"symbol"`
			Side             string `json:"side"`
			Type             string `json:"type"`
			TimeInForce      string `json:"timeInForce"`
			Quantity         string `json:"quantity"`
			Price            string `json:"price"`
			NewClientOrderID string `json:"newClientOrderId"`
			NewOrderRespType string `json:"newOrderRespType"`
			RecvWindow       int64  `json:"recvWindow"`
			Timestamp        int64  `json:"timestamp"`
			APIKey           string `json:"apiKey"`
			Signature        string `json:"signature"`
		}
	}{}
	frame.ID = request.ClOrdID
	frame.Method = "order.place"
	frame.Params.Symbol = request.Symbol
	frame.Params.Side = request.Side.String()
	if request.TimeInForce == mkt.GTC {
		frame.Params.Type = "LIMIT_MAKER"
	} else {
		frame.Params.Type = "LIMIT"
	}
	frame.Params.Quantity = request.OrderQty.String()
	frame.Params.Price = request.Price.String()
	frame.Params.NewClientOrderID = request.ClOrdID
	frame.Params.NewOrderRespType = "ACK"
	frame.Params.RecvWindow = RecvWindow
	frame.Params.Timestamp = now
	frame.Params.APIKey = apiKey
	frame.Params.Signature = signature

	return json.Marshal(&frame)
}

func newRequestPayloadForSignature(request *dma.NewRequest, unixMillis int64, apiKey string) string {

	var builder strings.Builder

	builder.WriteString("apiKey=")
	builder.WriteString(apiKey)
	builder.WriteString("&")
	builder.WriteString("newClientOrderId=")
	builder.WriteString(request.ClOrdID)
	builder.WriteString("&")
	builder.WriteString("newOrderRespType=ACK")
	builder.WriteString("&")
	builder.WriteString("price=")
	builder.WriteString(request.Price.String())
	builder.WriteString("&")
	builder.WriteString("quantity=")
	builder.WriteString(request.OrderQty.String())
	builder.WriteString("&")
	builder.WriteString("recvWindow=")
	builder.WriteString(strconv.Itoa(RecvWindow))
	builder.WriteString("&")
	builder.WriteString("side=")
	builder.WriteString(request.Side.String())
	builder.WriteString("&")
	builder.WriteString("symbol=")
	builder.WriteString(request.Symbol)
	builder.WriteString("&")
	builder.WriteString("timeInForce=")
	builder.WriteString(request.TimeInForce.String())
	builder.WriteString("&")
	builder.WriteString("timestamp=")
	builder.WriteString(strconv.FormatInt(unixMillis, 10))
	builder.WriteString("&")
	if request.TimeInForce == mkt.GTC {
		builder.WriteString("type=LIMIT_MAKER")
	} else {
		builder.WriteString("type=LIMIT")
	}

	return builder.String()

}

func sign(payload string, secret string) string {
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(payload))
	return hex.EncodeToString(hash.Sum(nil))
}

// CancelRequestFrame returns a web socket frame for a [dma.CancelRequest].
func CancelRequestFrame(request *dma.CancelRequest, apiKey, secret string) ([]byte, error) {

	now := time.Now().UnixMilli()
	payload := cancelRequestPayloadForSignature(request, now, apiKey)
	signature := sign(payload, secret)

	frame := struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Params struct {
			Symbol            string `json:"symbol"`
			OrigClientOrderID string `json:"origClientOrderId"`
			RecvWindow        int64  `json:"recvWindow"`
			Timestamp         int64  `json:"timestamp"`
			APIKey            string `json:"apiKey"`
			Signature         string `json:"signature"`
		}
	}{}
	frame.ID = request.ClOrdID
	frame.Method = "order.cancel"
	frame.Params.Symbol = request.OpenOrder.Symbol
	frame.Params.OrigClientOrderID = request.OrigClOrdID
	frame.Params.RecvWindow = RecvWindow
	frame.Params.Timestamp = now
	frame.Params.APIKey = apiKey
	frame.Params.Signature = signature

	return json.Marshal(&frame)
}

func cancelRequestPayloadForSignature(request *dma.CancelRequest, unixMillis int64, apiKey string) string {

	var builder strings.Builder

	builder.WriteString("apiKey=")
	builder.WriteString(apiKey)
	builder.WriteString("&")
	builder.WriteString("origClientOrderId=")
	builder.WriteString(request.OrigClOrdID)
	builder.WriteString("&")
	builder.WriteString("recvWindow=")
	builder.WriteString(strconv.Itoa(RecvWindow))
	builder.WriteString("&")
	builder.WriteString("symbol=")
	builder.WriteString(request.OpenOrder.Symbol)
	builder.WriteString("&")
	builder.WriteString("timestamp=")
	builder.WriteString(strconv.FormatInt(unixMillis, 10))

	return builder.String()

}
