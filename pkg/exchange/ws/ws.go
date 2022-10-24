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
type PingFn func(ws *WSConn) error
type PongFn func(message []byte) bool

func defaultPingFn(ws *WSConn) error {
	return ws.WriteText("ping")
}

func defaultPongFn(message []byte) bool {
	return bytes.Equal(pongMsg, message)
}

type WSConn struct {
	addr       string
	ws         *websocket.Conn
	initFn     WSInitFn
	messageFn  MessageFn
	pingFn     PingFn
	pongFn     PongFn
	closeCh    chan int
	writeMuetx sync.Mutex
	wg         sync.WaitGroup
}

func NewWSConnWithPingPong(addr string, initFn WSInitFn, messageFn MessageFn, ping PingFn, pong PongFn) (conn *WSConn, err error) {
	conn = new(WSConn)
	conn.addr = addr
	conn.initFn = initFn
	conn.messageFn = messageFn
	conn.closeCh = make(chan int, 1)
	conn.pingFn = ping
	conn.pongFn = pong
	err = conn.connect()
	return
}

func NewWSConn(addr string, initFn WSInitFn, messageFn MessageFn) (conn *WSConn, err error) {
	conn = new(WSConn)
	conn.addr = addr
	conn.initFn = initFn
	conn.pingFn = defaultPingFn
	conn.pongFn = defaultPongFn
	conn.messageFn = messageFn
	conn.closeCh = make(chan int, 1)
	err = conn.connect()
	if err != nil {
		return
	}
	return
}

func (conn *WSConn) SetPingPongFn(ping PingFn, pong PongFn) {
	conn.pingFn = ping
	conn.pongFn = pong
}

func (conn *WSConn) Close() (err error) {
	close(conn.closeCh)
	conn.wg.Wait()
	return
}

func (conn *WSConn) WriteText(value string) (err error) {
	conn.writeMuetx.Lock()
	if conn.ws != nil {
		err = conn.ws.WriteMessage(websocket.TextMessage, []byte(value))
	} else {
		log.Warnf("WriteText ignore conn of %s not init", conn.addr)
	}
	conn.writeMuetx.Unlock()
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

			if conn.pongFn != nil && conn.pongFn(msg) {
				continue
			}
			err = conn.messageFn(msg)
			if err != nil {
				break Out
			}
		case <-ticker.C:
			dur := time.Since(lastMsgTime)
			if dur > time.Second*5 {
				if conn.pingFn != nil {
					err1 := conn.pingFn(conn)
					if err1 != nil {
						log.Errorf("ws pingFn failed: %s", err1.Error())
					}
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
