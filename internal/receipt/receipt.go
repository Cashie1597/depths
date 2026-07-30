package receipt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cashie/depths/internal/claim"
	"github.com/cashie/depths/internal/sample"
)

type Receipt struct {
	Version   string         `json:"version"`
	Command   string         `json:"command"`
	Profile   string         `json:"profile"`
	DryRun    bool           `json:"dry_run"`
	Before    sample.Memory  `json:"before"`
	After     *sample.Memory `json:"after,omitempty"`
	Result    claim.Result   `json:"result"`
	WrittenAt time.Time      `json:"written_at"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Logs", "depths")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func Write(r Receipt) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	r.WrittenAt = time.Now()
	if r.Version == "" {
		r.Version = "0.1.0"
	}
	name := fmt.Sprintf("claim-%s.json", r.WrittenAt.Format("20060102-150405"))
	path := filepath.Join(dir, name)
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
