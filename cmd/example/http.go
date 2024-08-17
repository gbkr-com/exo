package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gbkr-com/mkt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	basePath = "/v1/orders"
)

// A Handler for HTTP traffic.
type Handler struct {
	rdb          *redis.Client
	key          string
	instructions chan *mkt.Order
}

// Bind this [Handler] to [*gin.Engine].
func (x *Handler) Bind(router *gin.Engine) {
	router.POST(basePath, x.postOrder)
	router.GET(basePath+"/:id", x.getOrder)
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

	ctx.JSON(http.StatusAccepted, gin.H{"orderID": orderID})
}

func (x *Handler) getOrder(ctx *gin.Context) {
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
	order, err := x.rdb.HGet(context.Background(), x.key, uri.OrderID).Result()
	if err == redis.Nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx.String(http.StatusOK, order)
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
	str, err := x.rdb.HGet(context.Background(), x.key, uri.OrderID).Result()
	if err == redis.Nil {
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}
	var order mkt.Order
	if err := json.Unmarshal([]byte(str), &order); err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	//
	// Forward.
	//
	order.MsgType = mkt.OrderCancel
	x.instructions <- &order
	x.rdb.HDel(context.Background(), x.key, uri.OrderID)

	ctx.JSON(http.StatusAccepted, gin.H{"orderID": uri.OrderID})
}
