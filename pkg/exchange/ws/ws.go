package ws

import (
	"bytes"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	pongMsg = []byte("pong")
)

type WSInitFn func(ws *WSConn) error
type MessageFn func(message []byte) error

type WSConn struct {
	addr           string
	ws             *websocket.Conn
	initFn         WSInitFn
	messageFn      MessageFn
	pongFn         MessageFn
	closeCh        chan int
	writeMuetx     sync.Mutex
	wg             sync.WaitGroup
	disablePingMsg bool
}

func NewWSConnWithoutPing(addr string, initFn WSInitFn, messageFn MessageFn) (conn *WSConn, err error) {
	conn = new(WSConn)
	conn.addr = addr
	conn.initFn = initFn
	conn.messageFn = messageFn
	conn.closeCh = make(chan int, 1)
	conn.disablePingMsg = true
	err = conn.connect()
	return
}

func NewWSConn(addr string, initFn WSInitFn, messageFn MessageFn) (conn *WSConn, err error) {
	conn = new(WSConn)
	conn.addr = addr
	conn.initFn = initFn
	conn.messageFn = messageFn
	conn.closeCh = make(chan int, 1)
	err = conn.connect()
	if err != nil {
		return
	}
	return
}

func (conn *WSConn) SetDisablePingMsg(disablePingMsg bool) {
	conn.disablePingMsg = disablePingMsg
}

func (conn *WSConn) SetPongMsgFn(pong MessageFn) {
	conn.pongFn = pong
}

func (conn *WSConn) Close() (err error) {
	close(conn.closeCh)
	conn.wg.Wait()
	return
}

func (conn *WSConn) WriteMsg(value interface{}) (err error) {
	conn.writeMuetx.Lock()
	if conn.ws != nil {
		err = conn.ws.WriteJSON(value)
	} else {
		log.Warnf("WriteMsg ignore conn of %s not init", conn.addr)
	}
	conn.writeMuetx.Unlock()
	return
}

func (conn *WSConn) connect() (err error) {
	u, err := url.Parse(conn.addr)
	if err != nil {
		return
	}
	conn.ws, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		err = fmt.Errorf("connect to %s failed: %w", conn.addr, err)
		return
	}
	if conn.initFn != nil {
		err = conn.initFn(conn)
		if err != nil {
			conn.ws.Close()
			return
		}
	}
	go conn.loop()
	return
}

func (conn *WSConn) loop() {
	ws := conn.ws
	ch := make(chan []byte, 1024)
	needReconn := make(chan bool, 1)
	go conn.readLoop(ws, ch, needReconn)
	var msg []byte
	var err error
	var lastMsgTime time.Time
	ticker := time.NewTicker(time.Second * 5)

	conn.wg.Add(1)
	defer conn.wg.Done()
	var ok bool
	defer ticker.Stop()
Out:
	for {
		select {
		case msg, ok = <-ch:
			if !ok {
				break Out
			}
			lastMsgTime = time.Now()
			if bytes.Equal(msg, pongMsg) {
				if conn.pongFn != nil {
					conn.pongFn(pongMsg)
				}
				continue
			}
			err = conn.messageFn(msg)
			if err != nil {
				break Out
			}
		case <-ticker.C:
			dur := time.Since(lastMsgTime)
			if dur > time.Second*5 {
				if !conn.disablePingMsg {
					conn.writeMuetx.Lock()
					ws.WriteMessage(websocket.TextMessage, []byte("ping"))
					conn.writeMuetx.Unlock()
				}
			}
		case <-conn.closeCh:
			return
		}
	}
	reConn := <-needReconn
	if reConn {
		for i := 0; i != 100; i++ {
			err = conn.connect()
			if err == nil {
				break
			}
			log.Errorf("ws reconnect %d to failed: %s", i, err.Error())
			time.Sleep(time.Second)
		}
	}
}

func (conn *WSConn) readLoop(ws *websocket.Conn, ch chan []byte, needConn chan bool) {
	defer func() {
		ws.Close()
		close(ch)
		close(needConn)
	}()
	var message []byte
	var err error
	for {
		select {
		case <-conn.closeCh:
			needConn <- false
			return
		default:
		}
		_, message, err = ws.ReadMessage()
		if err != nil {
			log.Printf("%s ws read error: %s", conn.addr, err.Error())
			needConn <- true
			return
		}
		ch <- message
	}
}
