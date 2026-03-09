package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	stateVersion           = "v1"
	defaultMemoryStateFile = "./data/memory_state.json"
)

type State struct {
	Version      string    `json:"version"`
	UpdatedAt    string    `json:"updated_at"`
	SystemMemory string    `json:"system_memory"`
	UserProfile  []History `json:"user_profile"`
	ToolsMemory  []History `json:"tools_memory"`
	ShortTerm    []History `json:"short_term"`
	MessageCount int       `json:"message_count"`
	ContextCoeff []float32 `json:"context_coeff"`
}

func (m *Memory) resolveStatePath(path string) string {
	if path != "" {
		return path
	}
	if m != nil && m.Cfg.MemoryStateFile != "" {
		return m.Cfg.MemoryStateFile
	}
	return defaultMemoryStateFile
}

func (m *Memory) snapshotState() State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := State{
		Version:      stateVersion,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		SystemMemory: m.SystemMemory,
		UserProfile:  append([]History(nil), m.UserProfile...),
		ToolsMemory:  append([]History(nil), m.ToolsMemory...),
		ShortTerm:    append([]History(nil), m.ShortTerm...),
		MessageCount: m.Tokens.MessageCount,
		ContextCoeff: m.Tokens.ContextCoeffSnapshot(),
	}
	return state
}

func (m *Memory) applyState(state State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SystemMemory = state.SystemMemory
	m.UserProfile = append([]History(nil), state.UserProfile...)
	m.ToolsMemory = append([]History(nil), state.ToolsMemory...)
	m.ShortTerm = append([]History(nil), state.ShortTerm...)
	m.Tokens.MessageCount = max(state.MessageCount, 0)
	if len(state.ContextCoeff) > 0 {
		m.Tokens.SetContextCoeffSnapshot(state.ContextCoeff)
	}
}

func (m *Memory) LoadState(path string) error {
	if m == nil {
		return errors.New("memory is nil")
	}

	statePath := m.resolveStatePath(path)
	raw, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read memory state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("unmarshal memory state: %w", err)
	}
	if state.Version != "" && state.Version != stateVersion {
		return fmt.Errorf("unsupported memory state version: %s", state.Version)
	}

	m.applyState(state)
	return nil
}

func (m *Memory) SaveState(path string) error {
	if m == nil {
		return errors.New("memory is nil")
	}

	statePath := m.resolveStatePath(path)
	state := m.snapshotState()

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal memory state: %w", err)
	}

	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir memory state dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(statePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp memory state file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp memory state file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp memory state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("replace memory state file: %w", err)
	}
	return nil
}

func (m *Memory) FlushState() error {
	return m.SaveState("")
}
