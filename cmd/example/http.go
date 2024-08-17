package main

import (
	"net/http"
	"strings"

	"github.com/gbkr-com/mkt"
	"github.com/gin-gonic/gin"
)

const (
	basePath = "/v1/orders"
)

// A Handler for HTTP traffic.
type Handler struct {
	orders       map[string]*mkt.Order // Temporary
	instructions chan *mkt.Order
}

// Bind this [Handler] to [*gin.Engine].
func (x *Handler) Bind(router *gin.Engine) {
	router.POST(basePath, x.postOrder)
	router.DELETE(basePath+"/:id", x.deleteOrder)
}

func (x *Handler) postOrder(ctx *gin.Context) {
	//
	// Body.
	//
	body := struct {
		Side   string `json:"side"`
		Symbol string `json:"symbol"`
	}{}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//
	// Content.
	//
	body.Side = strings.ToUpper(body.Side)
	side := mkt.SideFromString(body.Side)
	if side == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unrecognised side"})
		return
	}
	//
	// Forward.
	//
	orderID := mkt.NewOrderID()
	order := &mkt.Order{
		MsgType: mkt.OrderNew,
		OrderID: orderID,
		Side:    side,
		Symbol:  body.Symbol,
	}
	x.instructions <- order
	x.orders[orderID] = order

	ctx.JSON(http.StatusAccepted, gin.H{"orderID": orderID})
}

func (x *Handler) deleteOrder(ctx *gin.Context) {
	//
	// URI.
	//
	uri := struct {
		OrderID string `uri:"id" binding:"required"`
	}{}
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//
	// Content.
	//
	order := x.orders[uri.OrderID]
	if order == nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	//
	// Forward.
	//
	order.MsgType = mkt.OrderCancel
	x.instructions <- order
	delete(x.orders, uri.OrderID)

	ctx.JSON(http.StatusAccepted, gin.H{"orderID": uri.OrderID})
}
