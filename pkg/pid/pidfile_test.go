package pid

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// tmpDir returns a clean temporary directory for a test, setting SYLASTRACLAWS_HOME
// so that config.GetStateDir() resolves inside dir.
func tmpDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pidtest-*")
	if err != nil {
		t.Fatal(err)
	}
	oldHome := os.Getenv("SYLASTRACLAWS_HOME")
	os.Setenv("SYLASTRACLAWS_HOME", dir)
	t.Cleanup(func() { os.RemoveAll(dir) })
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("SYLASTRACLAWS_HOME", oldHome)
		} else {
			os.Unsetenv("SYLASTRACLAWS_HOME")
		}
	})
	return dir
}

// testPidPath returns the expected PID file path under the current test's state dir.
func testPidPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(os.Getenv("SYLASTRACLAWS_HOME"), "state", pidFileName)
}

// TestGenerateToken verifies that generateToken produces a 32-character hex string.
func TestGenerateToken(t *testing.T) {
	token := generateToken()
	if len(token) != 32 {
		t.Errorf("expected token length 32, got %d (token: %q)", len(token), token)
	}
	// Verify all characters are valid hex.
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains non-hex character: %c", c)
		}
	}
}

// TestGenerateTokenUniqueness checks that two consecutive tokens differ.
func TestGenerateTokenUniqueness(t *testing.T) {
	a := generateToken()
	b := generateToken()
	if a == b {
		t.Error("two consecutive tokens should not be equal")
	}
}

// TestPidFilePath verifies that pidFilePath() resolves to the correct path
// when SYLASTRACLAWS_HOME is set.
func TestPidFilePath(t *testing.T) {
	_ = tmpDir(t)
	got := pidFilePath()
	want := testPidPath(t)
	if got != want {
		t.Errorf("pidFilePath() = %q, want %q", got, want)
	}
}

// TestWritePidFile creates a PID file and verifies its contents.
func TestWritePidFile(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	data, err := WritePidFile("127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
	if data.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", data.Host, "127.0.0.1")
	}
	if data.Port != 18790 {
		t.Errorf("Port = %d, want %d", data.Port, 18790)
	}
	if len(data.Token) != 32 {
		t.Errorf("Token length = %d, want 32", len(data.Token))
	}

	// Verify the file exists and can be unmarshalled.
	raw, err := os.ReadFile(pf)
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}

	var fileData PidFileData
	if err = json.Unmarshal(raw, &fileData); err != nil {
		t.Fatalf("failed to unmarshal pid file: %v", err)
	}
	if fileData.PID != data.PID || fileData.Token != data.Token {
		t.Error("file data mismatch")
	}

	// Verify file permissions (owner-only read/write).
	info, err := os.Stat(pf)
	if err != nil {
		t.Fatalf("failed to stat pid file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permission = %o, want 0600", perm)
	}
}

// TestWritePidFileOverwrite writes twice and verifies the PID file is replaced.
func TestWritePidFileOverwrite(t *testing.T) {
	_ = tmpDir(t)

	data1, err := WritePidFile("0.0.0.0", 18790)
	if err != nil {
		t.Fatalf("first WritePidFile failed: %v", err)
	}

	// Second write should succeed because the PID matches our process.
	data2, err := WritePidFile("0.0.0.0", 18800)
	if err != nil {
		t.Fatalf("second WritePidFile failed: %v", err)
	}

	if data2.Token == data1.Token {
		t.Error("token should change on re-write")
	}
	if data2.Port != 18800 {
		t.Errorf("Port = %d, want 18800", data2.Port)
	}
}

// TestWritePidFileStalePID writes a PID file with a non-running PID, then
// verifies WritePidFile cleans it up and writes a new one.
func TestWritePidFileStalePID(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	// Write a PID file with a PID that almost certainly doesn't exist.
	stale := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	data, err := WritePidFile("127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile with stale PID failed: %v", err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
}

// TestReadPidFileWithCheck verifies reading a valid PID file for the current process.
func TestReadPidFileWithCheck(t *testing.T) {
	_ = tmpDir(t)

	// Some sandboxed environments (e.g. macOS test runner) may restrict
	// signal(0), causing isProcessRunning(getpid()) to return false.
	if !isProcessRunning(os.Getpid()) {
		t.Skip("skipping: isProcessRunning(getpid()) is false in this environment")
	}

	written, err := WritePidFile("127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	read := ReadPidFileWithCheck()
	if read == nil {
		t.Fatal("ReadPidFileWithCheck returned nil for current process")
	}
	if read.PID != written.PID || read.Token != written.Token {
		t.Error("read data doesn't match written data")
	}
}

// TestReadPidFileWithCheckNonexistent returns nil for missing file.
func TestReadPidFileWithCheckNonexistent(t *testing.T) {
	_ = tmpDir(t)
	data := ReadPidFileWithCheck()
	if data != nil {
		t.Error("expected nil for nonexistent PID file")
	}
}

// TestReadPidFileWithCheckStalePID auto-cleans a PID file whose process is dead.
func TestReadPidFileWithCheckStalePID(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	stale := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	data := ReadPidFileWithCheck()
	if data != nil {
		t.Error("expected nil for stale PID")
	}

	// File should be cleaned up.
	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		t.Error("stale PID file should be removed")
	}
}

// TestReadPidFileWithCheckInvalidFile auto-cleans malformed PID file.
func TestReadPidFileWithCheckInvalidFile(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, []byte("not json"), 0o600)

	data := ReadPidFileWithCheck()
	if data != nil {
		t.Error("expected nil for malformed pid file")
	}

	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		t.Error("malformed PID file should be removed")
	}
}

// TestRemovePidFile removes the PID file for the current process.
func TestRemovePidFile(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	if _, err := WritePidFile("127.0.0.1", 18790); err != nil {
		t.Fatalf("WritePidFile failed: %v", err)
	}

	RemovePidFile()

	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		t.Error("PID file should be removed")
	}
}

// TestRemovePidFileDifferentPID does not remove a PID file owned by another process.
func TestRemovePidFileDifferentPID(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	RemovePidFile()

	if _, err := os.Stat(pf); os.IsNotExist(err) {
		t.Error("PID file should NOT be removed (different PID)")
	}
}

// TestRemovePidFileNonexistent does not error on missing file.
func TestRemovePidFileNonexistent(t *testing.T) {
	_ = tmpDir(t)
	// Should not panic or error.
	RemovePidFile()
}

func TestRemovePidFileIfPID(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	removed := RemovePidFileIfPID(99999999)
	if !removed {
		t.Fatal("expected RemovePidFileIfPID to remove matching pid file")
	}
	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		t.Error("PID file should be removed for matching expected PID")
	}
}

func TestRemovePidFileIfPIDMismatch(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	other := PidFileData{PID: 99999999, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(other, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	removed := RemovePidFileIfPID(88888888)
	if removed {
		t.Fatal("expected RemovePidFileIfPID to keep non-matching pid file")
	}
	if _, err := os.Stat(pf); os.IsNotExist(err) {
		t.Error("PID file should NOT be removed for mismatching expected PID")
	}
}

// TestWritePidFileContainerPID1 verifies that a leftover PID file with PID 1
// (typical container entrypoint) is treated as stale and overwritten.
func TestWritePidFileContainerPID1(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)

	stale := PidFileData{PID: 1, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	data, err := WritePidFile("127.0.0.1", 18790)
	if err != nil {
		t.Fatalf("WritePidFile should treat PID 1 as stale, got error: %v", err)
	}
	if data.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", data.PID, os.Getpid())
	}
}

// TestReadPidFileWithCheckContainerPID1 verifies that a leftover PID file
// with PID 1 is treated as stale and cleaned up.
func TestReadPidFileWithCheckContainerPID1(t *testing.T) {
	if os.Getpid() == 1 {
		t.Skip("test not meaningful when running as PID 1")
	}
	_ = tmpDir(t)
	pf := testPidPath(t)

	stale := PidFileData{PID: 1, Token: "deadbeef12345678deadbeef12345678"}
	raw, _ := json.MarshalIndent(stale, "", "  ")
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, raw, 0o600)

	data := ReadPidFileWithCheck()
	if data != nil {
		t.Error("expected nil for PID 1 leftover")
	}

	if _, err := os.Stat(pf); !os.IsNotExist(err) {
		t.Error("PID 1 leftover file should be removed")
	}
}

// TestReadPidFileUnlockedInvalidJSON returns error for malformed content.
func TestReadPidFileUnlockedInvalidJSON(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, []byte("not json"), 0o600)

	_, err := readPidFileUnlocked(pf)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestReadPidFileUnlockedInvalidPID returns error for non-positive PID.
func TestReadPidFileUnlockedInvalidPID(t *testing.T) {
	_ = tmpDir(t)
	pf := testPidPath(t)
	os.MkdirAll(filepath.Dir(pf), 0o755)
	os.WriteFile(pf, []byte(`{"pid": -1, "token": "a"}`), 0o600)

	_, err := readPidFileUnlocked(pf)
	if err == nil {
		t.Error("expected error for invalid PID")
	}
}
