package sxbet

import "strings"

func buildSxbetEventURL(eventID string) string {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return "https://sx.bet"
	}
	return "https://sx.bet/event/" + eventID
}

func buildSxbetMarketURL(marketHash, eventID string) string {
	hash := strings.TrimSpace(marketHash)
	if hash == "" {
		return buildSxbetEventURL(eventID)
	}
	return "https://sx.bet/market/" + hash
}