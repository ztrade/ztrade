package ws

import (
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWsConn(t *testing.T) {
	initFn := func(ws *WSConn) error {
		var p = map[string]interface{}{
			"op": "subscribe",
			"args": []interface{}{
				map[string]interface{}{"channel": "trades", "instType": "SWAP", "instId": "BTC-USDT-SWAP"},
			},
		}
		return ws.WriteMsg(p)
	}
	messageFn := func(msg []byte) error {
		fmt.Println("msg:", string(msg))
		return nil
	}
	conn, err := NewWSConn("wss://wsaws.okx.com:8443/ws/v5/public", initFn, messageFn)
	if err != nil {
		t.Fatal(err.Error())
	}
	time.Sleep(time.Second * 5)
	conn.Close()
}

func TestWsPing(t *testing.T) {
	initFn := func(ws *WSConn) error {
		return nil
	}
	messageFn := func(msg []byte) error {
		fmt.Println("msg:", string(msg))
		return nil
	}
	pongFn := func(msg []byte) error {
		fmt.Println("recv pong msg:", string(msg))
		return nil
	}
	conn, err := NewWSConn("wss://wsaws.okx.com:8443/ws/v5/public", initFn, messageFn)
	if err != nil {
		t.Fatal(err.Error())
	}
	conn.SetPongMsgFn(pongFn)
	conn.ws.WriteMessage(websocket.TextMessage, []byte("ping"))
	time.Sleep(time.Second * 15)
	conn.Close()
}
