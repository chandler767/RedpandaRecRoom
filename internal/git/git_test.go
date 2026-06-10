// internal/git/git_test.go
package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/chandler767/redpanda-rec-room/internal/git"
)

func TestAddCommitPush_DryRun(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "init")

	reportsDir := filepath.Join(dir, "reports")
	os.MkdirAll(reportsDir, 0755)
	os.WriteFile(filepath.Join(reportsDir, "test.json"), []byte(`{"id":"test"}`), 0644)

	err := git.AddCommitPush(dir, "reports/", "test", true)
	if err != nil {
		t.Fatalf("AddCommitPush() error: %v", err)
	}
}
