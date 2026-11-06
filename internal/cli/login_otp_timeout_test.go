package cli

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func ptySlave(t *testing.T, master *os.File) *os.File {
	t.Helper()
	fd := uintptr(master.Fd())
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGRANT), 0); errno != 0 {
		t.Fatal(errno)
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYUNLK), 0); errno != 0 {
		t.Fatal(errno)
	}
	var name [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0]))); errno != 0 {
		t.Fatal(errno)
	}
	n := bytes.IndexByte(name[:], 0)
	if n < 0 {
		n = len(name)
	}
	slave, err := os.OpenFile(string(name[:n]), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	return slave
}

func TestOTPTimeoutRestoresTerminalEcho(t *testing.T) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	slave := ptySlave(t, master)
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
