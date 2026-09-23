//go:build linux

package terminal

// Platform support for the prompt tests in prompt_exp_test.go.

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// ptyOpenAttempts and ptyOpenRetryPause bound the wait for a free pseudo-terminal.
const (
	ptyOpenAttempts   = 200
	ptyOpenRetryPause = 5 * time.Millisecond
)

// The pseudo-terminal requests of the linux kernel, as creack/pty names them.
//   - ptyUnlock: unlock the slave
//   - ptyGetNumber: copy the slave's number into the buffer
//   - termiosGet, termiosSet: read and write the terminal settings
//   - inputPendingCount: the count of unread input bytes
const (
	ptyUnlock         = 0x40045431
	ptyGetNumber      = 0x80045430
	termiosGet        = 0x5401
	termiosSet        = 0x5402
	inputPendingCount = 0x541B
)

// openPTY returns a raw pseudo-terminal pair; the caller closes both ends.
func openPTY(test testing.TB) (master, slave *os.File) {
	test.Helper()

	master = openPTYMaster(test)

	var unlock int32

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, master, ptyUnlock, unsafe.Pointer(&unlock)); err != nil {
		test.Fatalf("💣 pseudo-terminal unlock: %v", err)
	}

	var number uint32

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, master, ptyGetNumber, unsafe.Pointer(&number)); err != nil {
		test.Fatalf("💣 pseudo-terminal number: %v", err)
	}

	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		test.Fatalf("💣 open pseudo-terminal slave: %v", err)
	}

	rawInput(test, slave)

	return master, slave
}

// rawInput disables terminal processing so test bytes pass unchanged.
func rawInput(test testing.TB, slave *os.File) {
	test.Helper()

	var settings syscall.Termios

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, slave, termiosGet, unsafe.Pointer(&settings)); err != nil {
		test.Fatalf("💣 pseudo-terminal settings: %v", err)
	}

	settings.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	settings.Oflag &^= syscall.OPOST
	settings.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	settings.Cflag &^= syscall.CSIZE | syscall.PARENB
	settings.Cflag |= syscall.CS8
	settings.Cc[syscall.VMIN] = 1
	settings.Cc[syscall.VTIME] = 0

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, slave, termiosSet, unsafe.Pointer(&settings)); err != nil {
		test.Fatalf("💣 pseudo-terminal raw mode: %v", err)
	}
}

// inputPending reports unread slave input, or false if the terminal query fails.
func inputPending(test testing.TB, slave *os.File) (int32, bool) {
	test.Helper()

	var pending int32

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, slave, inputPendingCount, unsafe.Pointer(&pending)); err != nil {
		return 0, false
	}

	return pending, true
}

// ioctl runs a terminal request through the runtime poller to preserve deadlines. The argument
// stays a pointer until the syscall so stack movement cannot invalidate it.
func ioctl(test testing.TB, file *os.File, request uintptr, argument unsafe.Pointer) error {
	test.Helper()

	conn, err := file.SyscallConn()
	if err != nil {
		return err
	}

	var errno syscall.Errno

	if controlErr := conn.Control(func(fd uintptr) {
		// #nosec G103 -- the pointer is converted in the call expression, which keeps it valid across the call.
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(argument))
	}); controlErr != nil {
		return controlErr
	}

	if errno != 0 {
		return errno
	}

	return nil
}

// openPTYMaster retries opening the master while the kernel reclaims recently closed terminals.
func openPTYMaster(test testing.TB) *os.File {
	test.Helper()

	var lastErr error

	for range ptyOpenAttempts {
		master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
		if err == nil {
			return master
		}

		lastErr = err

		time.Sleep(ptyOpenRetryPause)
	}

	test.Fatalf("💣 open /dev/ptmx: %v", lastErr)

	return nil
}
