package market

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestCandles(t *testing.T) {
	ApiAddr := "https://www.okex.com/"
	api, err := NewClientWithResponses(ApiAddr)
	tEnd := time.Now().Add(-time.Hour * 24 * 365)
	tStart := tEnd.Add(-time.Hour)
	startStr := strconv.FormatInt((tStart.Unix() * 1000), 10)
	endStr := strconv.FormatInt((tEnd.Unix() * 1000), 10)
	bSize := "1m"
	var params = GetApiV5MarketHistoryCandlesParams{InstId: "ETH-USDT-SWAP", Bar: &bSize, After: &endStr, Before: &startStr}
	resp, err := api.GetApiV5MarketHistoryCandlesWithResponse(context.Background(), &params)
	if err != nil {
		t.Fatal(err.Error())

	}
	t.Log(string(resp.Body), resp.JSON200)
}
