package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadSnapshotFromFile loads a usage statistics snapshot from a JSON file.
// It returns ok=false when the file does not exist or the path is empty.
func LoadSnapshotFromFile(path string) (snapshot StatisticsSnapshot, ok bool, err error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return snapshot, false, nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, false, nil
		}
		return snapshot, false, fmt.Errorf("open usage snapshot: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&snapshot); err != nil {
		return snapshot, false, fmt.Errorf("decode usage snapshot: %w", err)
	}

	return snapshot, true, nil
}

// SaveSnapshotToFile persists a usage statistics snapshot to a JSON file.
// It writes atomically via a temporary file and rename.
func SaveSnapshotToFile(path string, snapshot StatisticsSnapshot) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create usage snapshot dir: %w", err)
		}
	}

	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create usage snapshot temp file: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encode usage snapshot: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync usage snapshot: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close usage snapshot: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename usage snapshot: %w", err)
	}

	return nil
}
