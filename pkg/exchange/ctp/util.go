package ctp

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

type SafeWait struct {
	value uint32
	err   error
}

func (w *SafeWait) Wait(ctx context.Context) error {
Out:
	for {
		select {
		case _ = <-ctx.Done():
			return errors.New("deadline")
		default:
			v := atomic.LoadUint32(&w.value)
			if v != 0 {
				break Out
			}
			time.Sleep(time.Millisecond)
		}
	}
	return w.err
}

func (w *SafeWait) Done(err error) {
	atomic.StoreUint32(&w.value, 1)
	if err == nil {
		return
	}
	w.err = err
}
