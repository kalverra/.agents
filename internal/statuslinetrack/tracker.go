// Package statuslinetrack provides tracking capabilities for the status line.
package statuslinetrack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ActiveTask is a running background task from the statusline payload.
type ActiveTask struct {
	Key  string
	Name string
	Sort int
}

// Entry is a tracked task with a persisted start time.
type Entry struct {
	Key       string
	Name      string
	StartedAt time.Time
}

type fileState struct {
	Tasks map[string]taskRecord `json:"tasks"`
}

type taskRecord struct {
	StartedAt time.Time `json:"started_at"`
	Name      string    `json:"name"`
}

// DefaultRoot returns the directory for per-session tracker files.
func DefaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agents-statusline-tracker")
	}
	return filepath.Join(home, ".agents", "statusline-tracker")
}

// Sync updates persisted task start times for sessionID and returns active entries in sort order.
func Sync(root, sessionID string, active []ActiveTask, now time.Time) ([]Entry, error) {
	if sessionID == "" {
		sessionID = "unknown"
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, err
	}

	path := filepath.Join(root, sessionID+".json")
	state, err := loadState(path)
	if err != nil {
		return nil, err
	}
	if state.Tasks == nil {
		state.Tasks = make(map[string]taskRecord)
	}

	activeKeys := make(map[string]struct{}, len(active))
	for _, task := range active {
		activeKeys[task.Key] = struct{}{}
		rec, exists := state.Tasks[task.Key]
		if !exists {
			rec = taskRecord{StartedAt: now, Name: task.Name}
		} else if task.Name != "" {
			rec.Name = task.Name
		}
		state.Tasks[task.Key] = rec
	}

	for key := range state.Tasks {
		if _, ok := activeKeys[key]; !ok {
			delete(state.Tasks, key)
		}
	}

	if err := saveState(path, state); err != nil {
		return nil, err
	}

	sort.Slice(active, func(i, j int) bool {
		return active[i].Sort < active[j].Sort
	})

	entries := make([]Entry, 0, len(active))
	for _, task := range active {
		rec := state.Tasks[task.Key]
		entries = append(entries, Entry{
			Key:       task.Key,
			Name:      rec.Name,
			StartedAt: rec.StartedAt,
		})
	}
	return entries, nil
}

func loadState(path string) (fileState, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is under controlled tracker root
	if err != nil {
		if os.IsNotExist(err) {
			return fileState{Tasks: map[string]taskRecord{}}, nil
		}
		return fileState{}, err
	}
	var state fileState
	if err := json.Unmarshal(data, &state); err != nil {
		return fileState{}, fmt.Errorf("parse tracker %s: %w", path, err)
	}
	if state.Tasks == nil {
		state.Tasks = map[string]taskRecord{}
	}
	return state, nil
}

func saveState(path string, state fileState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
