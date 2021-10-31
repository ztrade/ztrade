package event

import (
	"testing"

	"github.com/ztrade/ztrade/pkg/core"
)

func TestUnmarshalEvent(t *testing.T) {
	// buf := `{"data":{"type":"balance","data":{"Balance":100000}},"Name":"BTCUSDT","Time":"2021-10-31T11:15:49.131137699+08:00","From":"VExchange"}`
	var e = Event{Data: core.EventData{Type: "balance", Data: &core.BalanceInfo{Balance: 100}}, Name: "BTCUSDT"}
	buf, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(string(buf))
	err = json.Unmarshal(buf, &e)
	if err != nil {
		t.Fatal(err.Error())
	}
}
