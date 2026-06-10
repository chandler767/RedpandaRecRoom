// internal/schedule/schedule_test.go
package schedule_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/schedule"
)

func TestInstallAt(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "com.redpanda.llm-tracker.plist")

	err := schedule.InstallAt("/usr/local/bin/myapp", plistPath)
	if err != nil {
		t.Fatalf("InstallAt() error: %v", err)
	}

	data, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "/usr/local/bin/myapp") {
		t.Error("plist missing binary path")
	}
	if !strings.Contains(content, "com.redpanda.llm-tracker") {
		t.Error("plist missing label")
	}
	if !strings.Contains(content, "<integer>8</integer>") {
		t.Error("plist missing 08:00 schedule")
	}
}
