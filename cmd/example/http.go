package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gbkr-com/mkt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

const (
	basePath = "/v1/orders"
)

// A Handler for HTTP traffic.
type Handler struct {
	rdb          *redis.Client
	key          string
	instructions chan *Order
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
		Side     string `json:"side"`
		Symbol   string `json:"symbol"`
		OrderQty string `json:"orderQty"`
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
	if body.OrderQty == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing orderQty"})
		return
	}
	orderQty, err := decimal.NewFromString(body.OrderQty)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "orderQty"})
		return
	}
	if !orderQty.IsPositive() {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "orderQty"})
		return
	}
	//
	// Forward.
	//
	orderID := mkt.NewOrderID()
	order := &Order{
		Order: mkt.Order{
			MsgType: mkt.OrderNew,
			OrderID: orderID,
			Side:    side,
			Symbol:  body.Symbol,
		},
		OrderQty: &orderQty,
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
	var memo DelegateMemo
	if err := json.Unmarshal([]byte(str), &memo); err != nil {
		ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	//
	// Forward.
	//
	memo.Instructions.MsgType = mkt.OrderCancel
	x.instructions <- &memo.Instructions.Order
	x.rdb.HDel(context.Background(), x.key, uri.OrderID)

	ctx.JSON(http.StatusAccepted, gin.H{"orderID": uri.OrderID})
}
