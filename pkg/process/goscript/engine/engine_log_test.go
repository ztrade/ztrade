package engine

import (
	"io"
	"os"
	"reflect"
	"testing"
)

func withDiscardedStdout(t *testing.T, fn func()) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe failed: %v", err)
	}

	os.Stdout = w
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	<-done
	_ = r.Close()
}

func TestEngineImplGetLogReturnsLoggedLines(t *testing.T) {
	e := NewEngineImpl(nil, "")

	withDiscardedStdout(t, func() {
		e.Log("alpha", 1)
		e.Log("beta")
	})

	want := []string{"alpha 1", "beta"}
	got := e.GetLog()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetLog() = %#v, want %#v", got, want)
	}
}

func TestEngineImplGetLogReturnsCopy(t *testing.T) {
	e := NewEngineImpl(nil, "")

	withDiscardedStdout(t, func() {
		e.Log("immutable")
	})

	logs := e.GetLog()
	logs[0] = "changed"

	got := e.GetLog()
	want := []string{"immutable"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetLog() after mutate = %#v, want %#v", got, want)
	}
}
