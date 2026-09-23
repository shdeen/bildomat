package artifact

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Empty artifact batch: Given no artifacts, WriteMedia must return an error.
// 2. Filename claim failure: When the output directory does not exist, WriteMedia must return
//    ErrOutputFile and remove both temporary sources.
// 3. Full destination volume: When the destination volume has insufficient space, WriteMedia must
//    return ErrOutputFileWrite and remove both temporary sources.
// 4. Source removal permission failure: When directory permissions prevent removal of the first
//    source, WriteMedia must return ErrOutputFileRemove and still remove the second source.
// 5. Destination claim permission failure: When directory permissions prevent creation of the
//    destination, WriteMedia must return ErrOutputFileCreate and remove every temporary source.
// 6. Incomplete destination cleanup: When a full volume interrupts writing in-memory media or
//    copying a temporary file, WriteMedia must return ErrOutputFileWrite and remove the incomplete
//    destination.
// 7. Incomplete destination cleanup failure: Given a read-only destination handle and directory
//    permissions that prevent cleanup, writeToReservedFile must return ErrOutputFileWrite with
//    os.ErrPermission and leave the unremovable file in place.
// 8. Later artifact write failure: If the second artifact cannot be written, WriteMedia must return
//    ErrOutputFileWrite, preserve the first file's bytes, and remove the third artifact's unused
//    temporary source.
// 9. Multi-artifact destination collision: When batch-1.jpg already exists, WriteMedia must save
//    both artifacts under batch-02 with their original bytes and preserve the existing file.
// 10. Later artifact exclusive claim: When the requested artifact path exists, writeArtifact must
//     save the new content at the first free numbered path, return that path, and preserve the
//     original content.
// 11. Completed artifacts after cleanup failure: If source removal fails for the first or second
//     artifact, WriteMedia must still save all three artifacts, return their paths and byte counts,
//     and retain the permission error.
// 12. Destination failure after cleanup failure: If the first source cannot be removed and the
//     second source is missing, WriteMedia must retain the first saved file and its result record
//     and return both error causes. If the first source cannot be removed and the second filename
//     exceeds the filesystem limit, WriteMedia must retain the first saved file, its path, and its
//     byte count.

// TestWriteAllEmpty verifies invariant #1: Empty artifact batch.
//
// What is being tested:
// Given no artifacts, WriteMedia must return an error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteAllEmpty(t *testing.T) {
	if _, _, err := WriteMedia(t.TempDir(), "", media.Image, nil, false); err == nil {
		t.Errorf("✗ empty artifact slice should error")
	}

	if !t.Failed() {
		t.Log("✓ empty input errors")
	}
}

// TestFailureClaim verifies invariant #2: Filename claim failure.
//
// What is being tested:
// When the output directory does not exist, WriteMedia must return ErrOutputFile and remove both
// temporary sources.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFailureClaim(t *testing.T) {
	src1 := srcFile(t, "S1")
	src2 := srcFile(t, "S2")
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	_, _, err := WriteMedia(missing, "x", media.Image, []Media{
		{TmpPath: src1, FileExt: ".mp4"},
		{TmpPath: src2, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ claim failure = %v, want a categorized ErrOutputFile", err)
	}

	noSrcRemains(t, src1, src2)

	if !t.Failed() {
		t.Log("✓ a claim failure returns the disk error and removes every unconsumed source")
	}
}

// TestFullVolumeWrite verifies invariant #3: Full destination volume.
//
// What is being tested:
// When the destination volume has insufficient space, WriteMedia must return ErrOutputFileWrite and
// remove both temporary sources.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFullVolumeWrite(t *testing.T) {
	out := fullVolumeOutDir(t, "full")
	src1 := srcFile(t, strings.Repeat("A", 1024*1024))
	src2 := srcFile(t, "S2")

	_, _, err := WriteMedia(out, "x", media.Image, []Media{
		{TmpPath: src1, FileExt: ".mp4"},
		{TmpPath: src2, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ full-volume destination write = %v, want ErrOutputFileWrite", err)
	}

	noSrcRemains(t, src1, src2)

	if !t.Failed() {
		t.Log("✓ a genuinely full destination volume fails the write with the disk error and removes every remaining source")
	}
}

// TestSourceRemovalDenied verifies invariant #4: Source removal permission failure.
//
// What is being tested:
// When directory permissions prevent removal of the first source, WriteMedia must return
// ErrOutputFileRemove and still remove the second source.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSourceRemovalDenied(t *testing.T) {
	holder := filepath.Join(t.TempDir(), "holder")
	// #nosec G301 -- permissions are part of this source-directory failure test.
	if err := os.Mkdir(holder, 0o755); err != nil {
		t.Fatalf("💣 mkdir holder: %v", err)
	}

	src1 := filepath.Join(holder, "bild-dl-held")
	// #nosec G306 -- permissions are part of this source-directory failure test.
	if err := os.WriteFile(src1, []byte("S1"), 0o644); err != nil {
		t.Fatalf("💣 write source: %v", err)
	}

	// #nosec G302 -- permissions are part of this source-directory failure test.
	if err := os.Chmod(holder, 0o500); err != nil {
		t.Fatalf("💣 chmod holder: %v", err)
	}

	// #nosec G302 -- cleanup restores access to the test-owned directory.
	t.Cleanup(func() { _ = os.Chmod(holder, 0o755) })
	src2 := srcFile(t, "S2")
	dir := t.TempDir()

	_, _, err := WriteMedia(dir, "x", media.Image, []Media{
		{TmpPath: src1, FileExt: ".mp4"},
		{TmpPath: src2, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileRemove) {
		t.Errorf("✗ a genuinely denied source removal = %v, want ErrOutputFileRemove", err)
	}

	noSrcRemains(t, src2)

	if !t.Failed() {
		t.Log("✓ a source removal denied by real source-directory permissions returns the disk error and removes the removable source")
	}
}

// TestClaimDenied verifies invariant #5: Destination claim permission failure.
//
// What is being tested:
// When directory permissions prevent creation of the destination, WriteMedia must return
// ErrOutputFileCreate and remove every temporary source.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestClaimDenied(t *testing.T) {
	locked := filepath.Join(t.TempDir(), "locked")
	// #nosec G301 -- permissions are part of this destination-directory failure test.
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatalf("💣 mkdir locked: %v", err)
	}

	// #nosec G302 -- permissions are part of this destination-directory failure test.
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatalf("💣 chmod locked: %v", err)
	}

	// #nosec G302 -- cleanup restores access to the test-owned directory.
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	src1 := srcFile(t, "S1")
	src2 := srcFile(t, "S2")

	_, _, err := WriteMedia(locked, "x", media.Image, []Media{
		{TmpPath: src1, FileExt: ".mp4"},
		{TmpPath: src2, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileCreate) {
		t.Errorf("✗ a genuinely denied exclusive create = %v, want ErrOutputFileCreate", err)
	}

	noSrcRemains(t, src1, src2)

	if !t.Failed() {
		t.Log("✓ an exclusive create denied by real directory permissions returns the disk error and removes every source")
	}
}

// TestWriteFailureRemovesUnusable verifies invariant #6: Incomplete destination cleanup.
//
// What is being tested:
// When a full volume interrupts writing in-memory media or copying a temporary file, WriteMedia
// must return ErrOutputFileWrite and remove the incomplete destination. The copy case must also
// remove its source.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteFailureRemovesUnusable(t *testing.T) {
	t.Run("a failed bytes write removes the claimed destination", bytesWriteFailureRemoval)
	t.Run("a failed copy removes the partial destination", copyFailureRemoval)

	if !t.Failed() {
		t.Log("✓ a partially written destination is removed after either a direct write failure or a copy failure")
	}
}

// TestWriteFailureRemovalDenied verifies invariant #7: Incomplete destination cleanup failure.
//
// What is being tested:
// Given a read-only destination handle and directory permissions that prevent cleanup,
// writeToReservedFile must return ErrOutputFileWrite with os.ErrPermission and leave the
// unremovable file in place.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteFailureRemovalDenied(t *testing.T) {
	holder := t.TempDir()
	dstPath := filepath.Join(holder, "stuck.png")
	// A read-only handle makes the write itself fail; the read-only holding directory then
	// denies the removal of the just-created destination.
	// #nosec G302 G304 -- dstPath is test-owned and the permissions create the failure fixture.
	f, err := os.OpenFile(dstPath, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		t.Fatalf("💣 create read-only destination: %v", err)
	}

	// #nosec G302 -- permissions are part of this cleanup-failure test.
	if err := os.Chmod(holder, 0o500); err != nil {
		t.Fatalf("💣 chmod holder: %v", err)
	}

	// #nosec G302 -- cleanup restores access to the test-owned directory.
	t.Cleanup(func() { _ = os.Chmod(holder, 0o755) })

	_, err = writeToReservedFile(f, dstPath, Media{Data: []byte("DATA")})
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ a failed write with denied cleanup = %v, want the classified ErrOutputFileWrite", err)
	}

	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("✗ the failure does not carry the denied removal's cause: %v", err)
	}

	if _, sErr := os.Stat(dstPath); sErr != nil {
		t.Errorf("✗ the unremovable destination vanished or is unreadable: %v", sErr)
	}

	if !t.Failed() {
		t.Log("✓ a denied cleanup after a failed write keeps the classified failure and reports the leftover file's cause")
	}
}

// TestWriteAllLaterBytesFailure verifies invariant #8: Later artifact write failure.
//
// What is being tested:
// If the second artifact cannot be written, WriteMedia must return ErrOutputFileWrite, preserve the
// first file's bytes, and remove the third artifact's unused temporary source.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteAllLaterBytesFailure(t *testing.T) {
	dir := t.TempDir()
	src := srcFile(t, "UNCONSUMED")
	// The second extension names a missing subdirectory, so saving the first artifact succeeds
	// before the second fails.
	artifacts := []Media{
		{Data: []byte("OK"), FileExt: ".png"},
		{Data: []byte("FAILS"), FileExt: ".png/x"},
		{TmpPath: src, FileExt: ".mp4"},
	}

	_, _, err := WriteMedia(dir, "batch", media.Image, artifacts, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ a later bytes-write failure = %v, want ErrOutputFileWrite", err)
	}

	// #nosec G304 -- dir is a test-owned temporary directory.
	if b, rerr := os.ReadFile(filepath.Join(dir, "batch-0.png")); rerr != nil || string(b) != "OK" {
		t.Errorf("✗ the already-written first file = (%q, %v), want it kept in place", b, rerr)
	}

	noSrcRemains(t, src)

	if !t.Failed() {
		t.Log("✓ a later bytes-write failure returns the disk error, keeps written files, and removes the unconsumed sources")
	}
}

// TestWriteAllChecksEveryDestination verifies invariant #9: Multi-artifact destination collision.
//
// What is being tested:
// When batch-1.jpg already exists, WriteMedia must save both artifacts under batch-02 with their
// original bytes and preserve the existing file.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWriteAllChecksEveryDestination(t *testing.T) {
	dir := t.TempDir()
	existingPath := filepath.Join(dir, "batch-1.jpg")
	// #nosec G306 -- permissions are sufficient for a test-owned file.
	if err := os.WriteFile(existingPath, []byte("existing image"), 0o644); err != nil {
		t.Fatalf("💣 seed the second destination: %v", err)
	}

	artifacts := []Media{
		{Data: []byte("first image"), FileExt: ".png"},
		{Data: []byte("second image"), FileExt: ".jpg"},
	}

	stem, _, err := WriteMedia(dir, "batch", media.Image, artifacts, false)
	if err != nil {
		t.Errorf("✗ write after a collision on the second destination: %v", err)

		return
	}

	if stem != "batch-02" {
		t.Errorf("✗ resolved stem = %q, want batch-02", stem)
	}

	// #nosec G304 -- the seeded path belongs to this test's temporary directory.
	existingContent, err := os.ReadFile(existingPath)
	if err != nil {
		t.Errorf("✗ read the existing second destination: %v", err)
	} else if string(existingContent) != "existing image" {
		t.Errorf("✗ existing second destination changed to %q", existingContent)
	}

	for artifactIndex, generatedMedia := range artifacts {
		writtenPath := filepath.Join(dir, fmt.Sprintf("batch-02-%d%s", artifactIndex, generatedMedia.FileExt))
		// #nosec G304 -- the expected path belongs to this test's temporary directory.
		writtenContent, readErr := os.ReadFile(writtenPath)
		if readErr != nil {
			t.Errorf("✗ read %s: %v", filepath.Base(writtenPath), readErr)
		} else if string(writtenContent) != string(generatedMedia.Data) {
			t.Errorf("✗ %s content = %q, want %q", filepath.Base(writtenPath), writtenContent, generatedMedia.Data)
		}
	}

	if !t.Failed() {
		t.Log("✓ a collision on any multi-artifact destination moves the whole result without overwriting the existing file")
	}
}

// TestLaterArtifactNeverReplaces verifies invariant #10: Later artifact exclusive claim.
//
// What is being tested:
// When the requested artifact path exists, writeArtifact must save the new content at the first
// free numbered path, return that path, and preserve the original content.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestLaterArtifactNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	takenPath := filepath.Join(dir, "batch-1.png")
	// #nosec G306 -- permissions are sufficient for a test-owned file.
	if err := os.WriteFile(takenPath, []byte("another run's image"), 0o644); err != nil {
		t.Fatalf("💣 seed the taken destination: %v", err)
	}

	savedFile, err := writeArtifact(dir, "batch-1", Media{Data: []byte("this run's image"), FileExt: ".png"})
	if err != nil {
		t.Fatalf("💣 writeArtifact over a taken destination: %v", err)
	}

	wantPath := filepath.Join(dir, "batch-1-02.png")
	if savedFile.Path != wantPath {
		t.Errorf("✗ save report names %q, want %q", savedFile.Path, wantPath)
	}

	// #nosec G304 -- both paths belong to this test's temporary directory.
	if got, readErr := os.ReadFile(takenPath); readErr != nil || string(got) != "another run's image" {
		t.Errorf("✗ the taken destination = (%q, %v), want its own content kept", got, readErr)
	}

	// #nosec G304 -- both paths belong to this test's temporary directory.
	if got, readErr := os.ReadFile(wantPath); readErr != nil || string(got) != "this run's image" {
		t.Errorf("✗ the suffixed destination = (%q, %v), want this run's content", got, readErr)
	}

	if !t.Failed() {
		t.Log("✓ a later artifact never replaces a file that appeared after the stem was resolved")
	}
}

// TestCleanupFailureContinuesArtifacts verifies invariant #11: Completed artifacts after cleanup
// failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// If source removal fails for the first or second artifact, WriteMedia must still save all three
// artifacts, return their paths and byte counts, and retain the permission error. It must remove
// sources whose permissions allow cleanup and leave the blocked source.
//
// Kind: permanent.
func TestCleanupFailureContinuesArtifacts(t *testing.T) {
	for _, lockedIndex := range []int{0, 1} {
		t.Run(fmt.Sprintf("source-%d", lockedIndex), func(t *testing.T) {
			lockedDirectory := t.TempDir()
			ordinaryDirectory := t.TempDir()
			destination := t.TempDir()
			artifacts := make([]Media, 3)

			sourcePaths := make([]string, len(artifacts))
			for artifactIndex := range artifacts {
				sourceDirectory := ordinaryDirectory
				if artifactIndex == lockedIndex {
					sourceDirectory = lockedDirectory
				}

				sourcePath := filepath.Join(sourceDirectory, fmt.Sprintf("source-%d", artifactIndex))
				if err := os.WriteFile(sourcePath, []byte{byte(artifactIndex)}, 0o600); err != nil {
					t.Fatalf("💣 source fixture: %v", err)
				}

				sourcePaths[artifactIndex] = sourcePath
				artifacts[artifactIndex] = Media{TmpPath: sourcePath, FileExt: ".bin"}
			}

			denyArtifactRemoval(t, lockedDirectory)

			_, savedFiles, err := WriteMedia(destination, "boat", media.Image, artifacts, false)
			if !errors.Is(err, os.ErrPermission) || len(savedFiles) != len(artifacts) {
				t.Errorf("✗ cleanup failure lost its cause or completed facts: %+v, %v", savedFiles, err)
			}

			for artifactIndex := range artifacts {
				path := filepath.Join(destination, fmt.Sprintf("boat-%d.bin", artifactIndex))

				// #nosec G304 -- this path belongs to the test temporary directory.
				content, readErr := os.ReadFile(path)
				if readErr != nil || len(content) != 1 || content[0] != byte(artifactIndex) {
					t.Errorf("✗ remaining artifact %d was not persisted intact: %v, %v", artifactIndex, content, readErr)
				}

				if artifactIndex < len(savedFiles) && (savedFiles[artifactIndex].Path != path || savedFiles[artifactIndex].Bytes != 1) {
					t.Errorf("✗ inaccurate completed fact: %+v", savedFiles[artifactIndex])
				}

				_, sourceErr := os.Stat(sourcePaths[artifactIndex])
				if artifactIndex == lockedIndex && sourceErr != nil {
					t.Errorf("✗ permission-denied source unexpectedly absent: %v", sourceErr)
				}

				if artifactIndex != lockedIndex && !errors.Is(sourceErr, os.ErrNotExist) {
					t.Errorf("✗ removable source remains: %v", sourceErr)
				}
			}

			if !t.Failed() {
				t.Log("✓ cleanup failure preserves all completed facts and remaining writes")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ first and later cleanup failures do not stop artifact persistence")
	}
}

// TestCleanupThenDestinationFailure verifies invariant #12: Destination failure after cleanup
// failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// If the first source cannot be removed and the second source is missing, WriteMedia must retain
// the first saved file and its result record and return both error causes. It must remove the
// incomplete second destination, skip the third write, and remove the third source.
//
// Kind: permanent.
func TestCleanupThenDestinationFailure(t *testing.T) {
	lockedDirectory := t.TempDir()
	ordinaryDirectory := t.TempDir()
	destination := t.TempDir()
	firstSource := filepath.Join(lockedDirectory, "first")

	lastSource := filepath.Join(ordinaryDirectory, "last")
	for _, sourcePath := range []string{firstSource, lastSource} {
		if err := os.WriteFile(sourcePath, []byte("media"), 0o600); err != nil {
			t.Fatalf("💣 source fixture: %v", err)
		}
	}

	denyArtifactRemoval(t, lockedDirectory)

	_, savedFiles, err := WriteMedia(destination, "boat", media.Image, []Media{
		{TmpPath: firstSource, FileExt: ".bin"},
		{TmpPath: filepath.Join(ordinaryDirectory, "missing"), FileExt: ".bin"},
		{TmpPath: lastSource, FileExt: ".bin"},
	}, false)
	if !errors.Is(err, os.ErrPermission) || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ cleanup or subsequent source-read cause lost: %v", err)
	}

	if len(savedFiles) != 1 || savedFiles[0].Path != filepath.Join(destination, "boat-0.bin") || savedFiles[0].Bytes != 5 {
		t.Errorf("✗ first completed artifact fact lost: %+v", savedFiles)
	}

	// #nosec G304 -- this path belongs to the test temporary directory.
	content, readErr := os.ReadFile(filepath.Join(destination, "boat-0.bin"))
	if readErr != nil || string(content) != "media" {
		t.Errorf("✗ first completed artifact changed: %q, %v", content, readErr)
	}

	for _, absentPath := range []string{filepath.Join(destination, "boat-1.bin"), filepath.Join(destination, "boat-2.bin"), lastSource} {
		if _, statErr := os.Stat(absentPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("✗ stopped write or removable source remains at %s: %v", absentPath, statErr)
		}
	}

	if !t.Failed() {
		t.Log("✓ later destination failure retains earlier facts and all failure causes")
	}
}

// TestCleanupThenCreateFailure verifies invariant #12: Destination failure after cleanup failure.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// If the first source cannot be removed and the second filename exceeds the filesystem limit,
// WriteMedia must retain the first saved file, its path, and its byte count. The error must match
// os.ErrPermission, ENAMETOOLONG, and ErrOutputFileCreate; the third artifact must remain unwritten
// and its source must be removed.
//
// Kind: permanent.
func TestCleanupThenCreateFailure(t *testing.T) {
	lockedDirectory := t.TempDir()
	ordinaryDirectory := t.TempDir()
	destination := t.TempDir()
	firstSource := filepath.Join(lockedDirectory, "first")

	lastSource := filepath.Join(ordinaryDirectory, "last")
	for _, sourcePath := range []string{firstSource, lastSource} {
		if err := os.WriteFile(sourcePath, []byte("media"), 0o600); err != nil {
			t.Fatalf("💣 generated source fixture: %v", err)
		}
	}

	oversizedExtension := "." + strings.Repeat("x", 256)
	if _, err := os.Stat(filepath.Join(destination, "boat-1"+oversizedExtension)); !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Fatalf("💣 destination fixture did not exceed its filename limit: %v", err)
	}

	denyArtifactRemoval(t, lockedDirectory)

	_, savedFiles, err := WriteMedia(destination, "boat", media.Image, []Media{
		{TmpPath: firstSource, FileExt: ".bin"},
		{Data: []byte("media"), FileExt: oversizedExtension},
		{TmpPath: lastSource, FileExt: ".bin"},
	}, false)
	if !errors.Is(err, os.ErrPermission) || !errors.Is(err, syscall.ENAMETOOLONG) || !errors.Is(err, errs.ErrOutputFileCreate) {
		t.Errorf("✗ source cleanup or later destination-create failure lost: %v", err)
	}

	if len(savedFiles) != 1 || savedFiles[0].Path != filepath.Join(destination, "boat-0.bin") || savedFiles[0].Bytes != 5 {
		t.Errorf("✗ completed first file fact lost: %+v", savedFiles)
	}

	// #nosec G304 -- this path belongs to the test temporary directory.
	content, readErr := os.ReadFile(filepath.Join(destination, "boat-0.bin"))
	if readErr != nil || string(content) != "media" {
		t.Errorf("✗ completed destination changed: %q, %v", content, readErr)
	}

	for _, absentPath := range []string{filepath.Join(destination, "boat-2.bin"), lastSource} {
		if _, statErr := os.Stat(absentPath); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("✗ unattempted destination or unconsumed source remains: %s, %v", absentPath, statErr)
		}
	}

	if !t.Failed() {
		t.Log("✓ destination create failure stops writes and retains earlier facts and cleanup causes")
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

// ramDiskDir mounts a separate RAM disk for filesystem failure tests and schedules its removal.
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

// fillDisk writes zeros into dir until the volume returns ENOSPC.
func fillDisk(t *testing.T, dir string) {
	t.Helper()

	// #nosec G304 -- dir is the dedicated test ram disk created by ramDiskDir.
	f, err := os.Create(filepath.Join(dir, "fill"))
	if err != nil {
		t.Fatalf("💣 create fill file: %v", err)
	}

	defer func() { _ = f.Close() }()

	chunk := make([]byte, 64*1024)
	for {
		if _, err := f.Write(chunk); err != nil {
			if errors.Is(err, syscall.ENOSPC) {
				return
			}

			t.Fatalf("💣 filling the volume failed before it was full: %v", err)
		}
	}
}

// fullVolumeOutDir returns a RAM disk directory with space to create a file but not finish writing
// the test payload.
func fullVolumeOutDir(t *testing.T, tag string) string {
	t.Helper()
	ram := ramDiskDir(t, tag)

	out := filepath.Join(ram, "out")
	// #nosec G301 -- permissions are part of this full-volume failure test.
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatalf("💣 mkdir out: %v", err)
	}

	spacer := filepath.Join(ram, "spacer")
	// #nosec G306 -- permissions are part of this full-volume failure test.
	if err := os.WriteFile(spacer, make([]byte, 128*1024), 0o644); err != nil {
		t.Fatalf("💣 write spacer: %v", err)
	}

	fillDisk(t, ram)

	if err := os.Remove(spacer); err != nil {
		t.Fatalf("💣 release spacer: %v", err)
	}

	return out
}

// denyArtifactRemoval restricts directory permissions and verifies that file removal is denied.
func denyArtifactRemoval(t *testing.T, directory string) {
	t.Helper()

	probePath := filepath.Join(directory, "removal-probe")
	if err := os.WriteFile(probePath, []byte("probe"), 0o600); err != nil {
		t.Fatalf("💣 permission probe fixture: %v", err)
	}

	// #nosec G302 -- owner directory permissions establish and restore the test failure.
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatalf("💣 restrict source directory: %v", err)
	}

	t.Cleanup(func() {
		// #nosec G302 -- owner directory permissions establish and restore the test failure.
		if err := os.Chmod(directory, 0o700); err != nil {
			t.Errorf("✗ restore source-directory permissions: %v", err)
		}
	})

	if err := os.Remove(probePath); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("💣 fixture did not establish denied source removal: %v", err)
	}
}

// bytesWriteFailureRemoval checks that a failed write to a full volume removes the partial file.
func bytesWriteFailureRemoval(t *testing.T) {
	t.Helper()
	out := fullVolumeOutDir(t, "bytesfail")

	_, _, err := WriteMedia(out, "bytesfail", media.Image, []Media{
		{Data: make([]byte, 1024*1024), FileExt: ".png"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ a full-volume bytes write = %v, want ErrOutputFileWrite", err)
	}

	if _, sErr := os.Stat(filepath.Join(out, "bytesfail.png")); !errors.Is(sErr, os.ErrNotExist) {
		t.Errorf("✗ the partly written destination remains after the failed write: %v", sErr)
	}

	if !t.Failed() {
		t.Log("✓ a bytes write that fails partway removes the unusable destination and returns the disk error")
	}
}

// copyFailureRemoval checks that a failed copy to a full volume removes the partial file and
// source.
func copyFailureRemoval(t *testing.T) {
	t.Helper()
	out := fullVolumeOutDir(t, "copyfail")
	src := srcFile(t, strings.Repeat("A", 1024*1024))

	_, _, err := WriteMedia(out, "copyfail", media.Image, []Media{
		{TmpPath: src, FileExt: ".mp4"},
	}, false)
	if err == nil || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ a full-volume copy = %v, want ErrOutputFileWrite", err)
	}

	if _, sErr := os.Stat(filepath.Join(out, "copyfail.mp4")); !errors.Is(sErr, os.ErrNotExist) {
		t.Errorf("✗ the partial copy destination remains after the failed copy: %v", sErr)
	}

	noSrcRemains(t, src)

	if !t.Failed() {
		t.Log("✓ a copy that fails partway removes the unusable destination and returns the disk error")
	}
}
