package main

import "strings"

const (
	bookTableREST = "market_order_book"
	bookTableWS   = "market_order_book_ws"
)

// orderBookTable 当前扫描的订单簿表（仅允许白名单表名）。
var orderBookTable = bookTableREST

func setOrderBookTable(flag string) {
	switch strings.ToLower(strings.TrimSpace(flag)) {
	case "ws", bookTableWS:
		orderBookTable = bookTableWS
	default:
		orderBookTable = bookTableREST
	}
}

func orderBookFromWS() bool {
	return orderBookTable == bookTableWS
}

func bookCSVFileSuffix() string {
	if orderBookFromWS() {
		return "_订单簿WS.csv"
	}
	return "_订单簿REST.csv"
}

func bookSourceLabelCN() string {
	if orderBookFromWS() {
		return "WS"
	}
	return "REST"
}