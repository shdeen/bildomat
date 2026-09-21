//go:build darwin

package main

import (
	"bytes"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// The pseudo-terminal requests of the darwin kernel, as creack/pty names them.
//   - ptyGrant: grant access to the slave
//   - ptyUnlock: unlock the slave
//   - ptyGetName: copy the slave's device path into the buffer
//   - termiosGet, termiosSet: read and write the terminal settings
//   - inputPendingCount: the count of unread input bytes
//
// ptyOpenAttempts and ptyOpenRetryPause bound the wait for a free pseudo-terminal.
const (
	ptyOpenAttempts   = 200
	ptyOpenRetryPause = 5 * time.Millisecond
)

const (
	ptyGrant          = 0x20007454
	ptyUnlock         = 0x20007452
	ptyGetName        = 0x40807453
	termiosGet        = 0x40487413
	termiosSet        = 0x80487414
	inputPendingCount = 0x4004667f
)

// openPTY opens a pseudo-terminal pair and returns its master and slave ends,
// with input and output processing off; the caller closes them. What a
// test writes to the master, a program reads from the slave as terminal input; what the program writes to the
// slave, the test reads from the master.
//
// Test class: Core: Helper.
func openPTY(test testing.TB) (master, slave *os.File) {
	test.Helper()

	master = openPTYMaster(test)

	for _, request := range []uintptr{ptyGrant, ptyUnlock} {
		if err := ioctl(test, master, request, nil); err != nil {
			test.Fatalf("💣 pseudo-terminal request %#x: %v", request, err)
		}
	}

	var name [128]byte

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, master, ptyGetName, unsafe.Pointer(&name[0])); err != nil {
		test.Fatalf("💣 pseudo-terminal name: %v", err)
	}

	slave, err := os.OpenFile(string(name[:bytes.IndexByte(name[:], 0)]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		test.Fatalf("💣 open pseudo-terminal slave: %v", err)
	}

	rawInput(test, slave)

	return master, slave
}

// rawInput takes the slave end of a pseudo-terminal and turns off its input
// and output processing, so a test's bytes reach the program unchanged
// (no carriage-return conversion, echo, line editing, or signal keys) and
// the program's bytes reach the test unchanged.
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

// inputPending takes the slave end of a pseudo-terminal and returns how many
// bytes written to its master the program has not read yet, and false once
// the terminal is closed.
func inputPending(test testing.TB, slave *os.File) (int32, bool) {
	test.Helper()

	var pending int32

	// #nosec G103 -- the address stays a pointer until the call expression inside ioctl converts it.
	if err := ioctl(test, slave, inputPendingCount, unsafe.Pointer(&pending)); err != nil {
		return 0, false
	}

	return pending, true
}

// ioctl takes a terminal file, a request, and the request's argument (a
// pointer to the value the request reads or writes, or nil), and runs the
// request through the file's raw connection, so the file stays with the
// runtime's poller and its write deadline keeps working. The argument stays a
// pointer until the call expression itself, as the unsafe rules require, so
// a moved stack cannot leave the kernel writing to a stale address.
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

// openPTYMaster opens the pseudo-terminal master, retrying for up to a second
// while the system has no free pseudo-terminal: a fuzz run opens and closes
// one per iteration faster than the kernel reclaims them.
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
