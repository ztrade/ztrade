package bitmex

import (
	"strings"
	"time"
)

type dofn func() (interface{}, error)

func doOrderWithRetry(nRetry int, fn dofn) (ret interface{}, err error) {
	ret, err = fn()
	if err == nil {
		return
	}
	var msg string
	for n := 0; n != nRetry; n++ {
		msg = err.Error()
		if strings.Contains(msg, "status 503") ||
			strings.Contains(msg, "context deadline exceeded") {
			time.Sleep(time.Millisecond * 500)
			ret, err = fn()
			if err == nil {
				break
			}
		} else {
			break
		}
	}
	return
}
