package exchange

import (
	"errors"
	"time"
)

var (
	ErrCanRetry = errors.New("error but can retry")
)

type dofn func() (interface{}, error)

func doOrderWithRetry(nRetry int, fn dofn) (ret interface{}, err error) {
	ret, err = fn()
	if err == nil {
		return
	}
	for n := 0; n != nRetry; n++ {
		if errors.Is(err, ErrCanRetry) {
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
