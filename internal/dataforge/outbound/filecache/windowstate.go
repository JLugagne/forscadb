package filecache

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

func LoadWindowState(dir string) (*WindowState, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "window.json"))
	if err != nil {
		return nil, err
	}
	var state WindowState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveWindowState(dir string, state WindowState) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "window.json"), raw, 0o644)
}
