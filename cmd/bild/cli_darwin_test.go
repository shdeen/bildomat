//go:build darwin

package main

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"bytes"
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

// Darwin ioctl request codes used to open and configure the pseudo-terminal.
//   - ptyGrant: grant access to the slave
//   - ptyUnlock: unlock the slave
//   - ptyGetName: copy the slave's device path into the buffer
//   - termiosGet, termiosSet: read and write the terminal settings
//   - inputPendingCount: the count of unread input bytes
const (
	ptyGrant          = 0x20007454
	ptyUnlock         = 0x20007452
	ptyGetName        = 0x40807453
	termiosGet        = 0x40487413
	termiosSet        = 0x80487414
	inputPendingCount = 0x4004667f
)

// openPTY returns a raw pseudo-terminal pair; the caller closes both ends.
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

// rawInput disables terminal processing so test bytes pass unchanged.
//
// Test class: Core: Helper.
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

// inputPending returns the number of unread bytes on the slave and whether the query succeeded.
//
// Test class: Core: Helper.
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
//
// Test class: Core: Helper.
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
//
// Test class: Core: Helper.
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
