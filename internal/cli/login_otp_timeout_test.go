//go:build darwin || linux

package cli

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

func TestOTPTimeoutRestoresTerminalEcho(t *testing.T) {
	master, slave := openPTY(t)
	t.Cleanup(func() { _ = slave.Close() })
	t.Cleanup(func() { _ = master.Close() })

	old := os.Stdin
	os.Stdin = slave
	t.Cleanup(func() { os.Stdin = old })

	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = readPasswordWithTimeout(context.Background(), 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout error = %v", err)
	}
	after, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("terminal state changed after timeout\nbefore %#v\nafter %#v", before, after)
	}
}
