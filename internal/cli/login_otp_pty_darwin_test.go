//go:build darwin

package cli

import (
	"bytes"
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openPTY(t *testing.T) (master *os.File, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	fd := uintptr(master.Fd())
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGRANT), 0); errno != 0 {
		_ = master.Close()
		t.Fatal(errno)
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYUNLK), 0); errno != 0 {
		_ = master.Close()
		t.Fatal(errno)
	}
	var name [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0]))); errno != 0 {
		_ = master.Close()
		t.Fatal(errno)
	}
	n := bytes.IndexByte(name[:], 0)
	if n < 0 {
		n = len(name)
	}
	slave, err = os.OpenFile(string(name[:n]), os.O_RDWR, 0)
	if err != nil {
		_ = master.Close()
		t.Fatal(err)
	}
	return master, slave
}
