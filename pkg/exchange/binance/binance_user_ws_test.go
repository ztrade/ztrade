package binance

import (
	"fmt"
	"testing"
	"time"
)

func TestUserWS(t *testing.T) {
	err := testClt.startUserWS()
	if err != nil {
		t.Fatal(err.Error())
	}

	datas := testClt.GetDataChan()
	tm := time.After(time.Minute)
Out:
	for {
		select {
		case o, ok := <-datas:
			fmt.Println("datas:", o, ok)
		case <-tm:
			break Out
		}
	}
}
