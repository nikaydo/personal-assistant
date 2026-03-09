package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadState_MissingFileIsNoop(t *testing.T) {
	m := newTestMemory()
	err := m.LoadState(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadState returned error for missing file: %v", err)
	}
}

func TestSaveAndLoadState_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory_state.json")

	m1 := newTestMemory()
	m1.SystemMemory = "system"
	m1.UserProfile = []History{{Question: ShotTermQuestion{Text: "uq"}, Answer: ShotTermAnswer{Text: "ua"}}}
	m1.ToolsMemory = []History{{Question: ShotTermQuestion{Text: "tq"}, Answer: ShotTermAnswer{Text: "ta"}}}
	m1.ShortTerm = []History{{Question: ShotTermQuestion{Text: "q1"}, Answer: ShotTermAnswer{Text: "a1"}, Model: "m1", Id: "id1", Created: 123}}
	m1.Tokens.MessageCount = 3
	m1.Tokens.SetContextCoeffSnapshot([]float32{1.25, 2.5})

	if err := m1.SaveState(path); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `"version": "v1"`) {
		t.Fatalf("saved state does not contain version field: %s", text)
	}

	m2 := newTestMemory()
	m2.Tokens.SetContextCoeffSnapshot([]float32{9})
	if err := m2.LoadState(path); err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	if m2.SystemMemory != "system" {
		t.Fatalf("unexpected system memory: %q", m2.SystemMemory)
	}
	if len(m2.UserProfile) != 1 || m2.UserProfile[0].Question.Text != "uq" {
		t.Fatalf("unexpected user profile: %#v", m2.UserProfile)
	}
	if len(m2.ToolsMemory) != 1 || m2.ToolsMemory[0].Question.Text != "tq" {
		t.Fatalf("unexpected tools memory: %#v", m2.ToolsMemory)
	}
	if len(m2.ShortTerm) != 1 || m2.ShortTerm[0].Question.Text != "q1" {
		t.Fatalf("unexpected short term: %#v", m2.ShortTerm)
	}
	if m2.Tokens.MessageCount != 3 {
		t.Fatalf("unexpected message count: %d", m2.Tokens.MessageCount)
	}
	coeff := m2.Tokens.ContextCoeffSnapshot()
	if len(coeff) != 2 || coeff[0] != 1.25 || coeff[1] != 2.5 {
		t.Fatalf("unexpected context coeff snapshot: %#v", coeff)
	}
}

func TestLoadState_CorruptedJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	m := newTestMemory()
	err := m.LoadState(path)
	if err == nil {
		t.Fatalf("expected error for corrupted json")
	}
	if !strings.Contains(err.Error(), "unmarshal memory state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveState_AtomicWriteLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory_state.json")

	m := newTestMemory()
	m.ShortTerm = []History{{Question: ShotTermQuestion{Text: "q"}, Answer: ShotTermAnswer{Text: "a"}}}
	if err := m.SaveState(path); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, filepath.Base(path)+".tmp-*"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no temporary files after atomic write, got: %v", matches)
	}
}

func TestFlushState_UsesDefaultPathWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	m := newTestMemory()
	m.Cfg.MemoryStateFile = ""
	m.ShortTerm = []History{{Question: ShotTermQuestion{Text: "q"}, Answer: ShotTermAnswer{Text: "a"}}}
	if err := m.FlushState(); err != nil {
		t.Fatalf("FlushState returned error: %v", err)
	}

	_, err = os.Stat(filepath.Join(dir, "data", "memory_state.json"))
	if err != nil {
		t.Fatalf("expected default state file to exist: %v", err)
	}
}

func TestLoadState_UnsupportedVersionReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	payload := `{"version":"v2"}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	m := newTestMemory()
	err := m.LoadState(path)
	if err == nil {
		t.Fatalf("expected version error")
	}
	if !strings.Contains(err.Error(), "unsupported memory state version") {
		t.Fatalf("unexpected error: %v", err)
	}
}
