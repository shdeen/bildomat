package output

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
//  1. Cross-device artifact placement: Given temporary artifacts on another device,
//     artifact.WriteMedia must copy their exact bytes and remove each source, including in a batch;
//     the single-file report must name its saved path and size. An unreadable source must produce
//     ErrOutputFileWrite and remove all batch sources. If source removal fails, it must return
//     ErrOutputFile while leaving both the saved copy and source bytes intact.

// TestCrossDevice verifies invariant #1: Cross-device artifact placement.
//
// What is being tested:
// Given temporary artifacts on another device, artifact.WriteMedia must copy their exact bytes and
// remove each source, including in a batch; the single-file report must name its saved path and
// size. An unreadable source must produce ErrOutputFileWrite and remove all batch sources. If
// source removal fails, it must return ErrOutputFile while leaving both the saved copy and source
// bytes intact.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCrossDevice(t *testing.T) {
	ram := ramDiskDir(t, "xdev")
	t.Run("the copy is byte-identical and removes the source", func(t *testing.T) {
		crossDeviceCopies(t, ram)

		if !t.Failed() {
			t.Log("✓ the copy is byte-identical and removes the source")
		}
	})
	t.Run("a multi-file batch copies every file across devices", func(t *testing.T) {
		crossDeviceMultiFile(t, ram)

		if !t.Failed() {
			t.Log("✓ a multi-file batch copies every file across devices")
		}
	})
	t.Run("an unreadable source fails the copy and cleans up", func(t *testing.T) {
		crossDeviceSrcUnreadable(t, ram)

		if !t.Failed() {
			t.Log("✓ an unreadable source fails the copy and cleans up")
		}
	})
	t.Run("an unremovable consumed source surfaces the error with the copy intact", func(t *testing.T) {
		crossDeviceSrcUnremovable(t, ram)

		if !t.Failed() {
			t.Log("✓ an unremovable consumed source surfaces the error with the copy intact")
		}
	})

	if !t.Failed() {
		t.Log("✓ artifacts on another device are copied byte-identical, their sources removed, and copy or cleanup failures classified")
	}
}

// noSrcRemains asserts every given source temp path is gone.
func noSrcRemains(t *testing.T, srcs ...string) {
	t.Helper()

	for _, s := range srcs {
		if _, err := os.Stat(s); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("✗ unconsumed source %s still present after failure: %v", s, err)
		}
	}
}

// ramDiskDir creates, formats, and mounts a dedicated RAM disk so copies to or from its returned
// path cross device boundaries. Cleanup detaches the disk. Any setup failure fails the test.
func ramDiskDir(t *testing.T, tag string) string {
	t.Helper()

	attach := exec.CommandContext(t.Context(), "hdiutil", "attach", "-nomount", "ram://8192")

	devRaw, err := attach.Output()
	if err != nil {
		t.Fatalf("💣 hdiutil attach failed — the cross-device condition cannot be established: %v", err)
	}

	dev := strings.TrimSpace(string(devRaw))

	// #nosec G204 -- the executable and arguments are fixed; dev comes from hdiutil itself.
	t.Cleanup(func() { _ = exec.CommandContext(context.Background(), "hdiutil", "detach", dev, "-force").Run() })

	volume := fmt.Sprintf("bild-%d-%s", os.Getpid(), tag)

	// #nosec G204 -- the executable and operation are fixed; volume and dev are test-owned values.
	erase := exec.CommandContext(t.Context(), "diskutil", "eraseVolume", "HFS+", volume, dev)
	if out, err := erase.CombinedOutput(); err != nil {
		t.Fatalf("💣 diskutil eraseVolume failed: %v (%s)", err, out)
	}

	mount := filepath.Join("/Volumes", volume)
	if fi, err := os.Stat(mount); err != nil || !fi.IsDir() {
		t.Fatalf("💣 ram disk not mounted at %s: %v", mount, err)
	}

	return mount
}

// crossDeviceMultiFile checks that artifact.WriteMedia copies every file in a batch from another
// device, preserves its bytes at the indexed destination, and removes every source.
func crossDeviceMultiFile(t *testing.T, ram string) {
	t.Helper()
	dir := t.TempDir()
	contents := []string{"BATCH-A", "BATCH-B", "BATCH-C"}
	batch := make([]artifact.Media, len(contents))

	sources := make([]string, len(contents))
	for i, data := range contents {
		src := filepath.Join(ram, fmt.Sprintf("bild-dl-batch-%d", i))
		// #nosec G306 -- test artifacts intentionally match generated-file permissions.
		if err := os.WriteFile(src, []byte(data), 0o644); err != nil {
			t.Fatalf("💣 write ram source: %v", err)
		}

		sources[i] = src
		batch[i] = artifact.Media{TmpPath: src, FileExt: ".mp4"}
	}

	stem, _, err := artifact.WriteMedia(dir, "batch", media.Image, batch, false)
	if err != nil {
		t.Fatalf("💣 artifact.WriteMedia(multi-file batch across devices): %v", err)
	}

	for i, data := range contents {
		dst := filepath.Join(dir, fmt.Sprintf("%s-%d.mp4", stem, i))

		// #nosec G304 -- dst is derived from the test-owned temporary output directory.
		b, rErr := os.ReadFile(dst)
		if rErr != nil || string(b) != data {
			t.Errorf("✗ artifact %d = (%q,%v), want the source bytes at %s", i, b, rErr, dst)
		}
	}

	noSrcRemains(t, sources...)

	if !t.Failed() {
		t.Log("✓ a genuine cross-device multi-file batch lands every artifact byte-identical with every source removed")
	}
}

// crossDeviceCopies checks that artifact.WriteMedia copies a file from another device, preserves
// its bytes at the returned destination, removes the source, and produces the exact save report.
func crossDeviceCopies(t *testing.T, ram string) {
	t.Helper()
	dir := t.TempDir()

	src := filepath.Join(ram, "bild-dl-real")
	// #nosec G306 -- test artifacts intentionally match generated-file permissions.
	if err := os.WriteFile(src, []byte("CROSS-DEVICE-REAL"), 0o644); err != nil {
		t.Fatalf("💣 write ram source: %v", err)
	}

	stem, savedFiles, err := artifact.WriteMedia(dir, "clip", media.Image, []artifact.Media{{TmpPath: src, FileExt: ".mp4"}}, false)

	out := savedFileReports(t, savedFiles)
	if err != nil {
		t.Fatalf("💣 artifact.WriteMedia across devices: %v", err)
	}

	// #nosec G304 -- dir is a test-owned temporary directory and stem is the returned output name.
	b, rerr := os.ReadFile(filepath.Join(dir, stem+".mp4"))
	if rerr != nil || string(b) != "CROSS-DEVICE-REAL" {
		t.Errorf("✗ cross-device copy = (%q,%v), want the source bytes at the claimed path", b, rerr)
	}

	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ source temp file still present after the cross-device copy: %v", err)
	}

	if want := fmt.Sprintf(SavedReport+"\n", filepath.Join(dir, stem+".mp4"), 17); out != want {
		t.Errorf("✗ save report = %q, want %q", out, want)
	}

	if !t.Failed() {
		t.Log("✓ a genuine cross-device artifact is copied and its source removed, byte-identical with its exact save report")
	}
}

// crossDeviceSrcUnreadable checks that an unreadable source on another device makes
// artifact.WriteMedia return ErrOutputFileWrite and remove all batch sources.
func crossDeviceSrcUnreadable(t *testing.T, ram string) {
	t.Helper()
	dir := t.TempDir()

	src1 := filepath.Join(ram, "bild-dl-unreadable")
	// #nosec G306 -- test artifacts intentionally match generated-file permissions.
	if err := os.WriteFile(src1, []byte("S1"), 0o644); err != nil {
		t.Fatalf("💣 write ram source: %v", err)
	}

	if err := os.Chmod(src1, 0o000); err != nil {
		t.Fatalf("💣 chmod source: %v", err)
	}

	src2 := srcFile(t, "S2")

	_, _, err := artifact.WriteMedia(dir, "x", media.Image, []artifact.Media{
		{TmpPath: src1, FileExt: ".mp4"},
		{TmpPath: src2, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ copy source-open failure = %v, want ErrOutputFileWrite", err)
	}

	noSrcRemains(t, src1, src2)

	if !t.Failed() {
		t.Log("✓ a genuinely unreadable cross-device source returns the disk error and removes every source")
	}
}

// crossDeviceSrcUnremovable checks that a source-removal failure makes artifact.WriteMedia return
// ErrOutputFile while leaving both the copied destination and source bytes intact.
func crossDeviceSrcUnremovable(t *testing.T, ram string) {
	t.Helper()

	holder := filepath.Join(ram, "holder")
	// #nosec G301 -- permissions are part of this ram-disk failure test.
	if err := os.Mkdir(holder, 0o755); err != nil {
		t.Fatalf("💣 mkdir holder: %v", err)
	}

	src := filepath.Join(holder, "bild-dl-pinned")
	// #nosec G306 -- permissions are part of this ram-disk failure test.
	if err := os.WriteFile(src, []byte("PINNED"), 0o644); err != nil {
		t.Fatalf("💣 write source: %v", err)
	}

	// #nosec G302 -- permissions are part of this ram-disk failure test.
	if err := os.Chmod(holder, 0o555); err != nil {
		t.Fatalf("💣 chmod holder: %v", err)
	}

	// #nosec G302 -- cleanup restores access to the test-owned directory.
	t.Cleanup(func() { _ = os.Chmod(holder, 0o755) })
	dir := t.TempDir()

	_, _, err := artifact.WriteMedia(dir, "x", media.Image, []artifact.Media{{TmpPath: src, FileExt: ".mp4"}}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ an unremovable consumed source = %v, want a categorized ErrOutputFile", err)
	}

	// #nosec G304 -- dir is a test-owned temporary directory.
	if b, rerr := os.ReadFile(filepath.Join(dir, "x.mp4")); rerr != nil || string(b) != "PINNED" {
		t.Errorf("✗ the copied destination = (%q, %v), want the intact PINNED copy", b, rerr)
	}

	// #nosec G304 -- src is a test-owned path created above.
	if b, rerr := os.ReadFile(src); rerr != nil || string(b) != "PINNED" {
		t.Errorf("✗ the unremovable source = (%q, %v), want it untouched", b, rerr)
	}

	if !t.Failed() {
		t.Log("✓ a source the copy genuinely cannot remove surfaces the categorized disk error with the copy intact")
	}
}
