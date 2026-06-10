// internal/schedule/schedule.go
package schedule

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const plistLabel = "com.redpanda.llm-tracker"

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.BinaryPath}}</string>
	</array>
	<key>StartCalendarInterval</key>
	<array>
		<dict><key>Hour</key><integer>8</integer><key>Minute</key><integer>0</integer></dict>
		<dict><key>Hour</key><integer>14</integer><key>Minute</key><integer>0</integer></dict>
		<dict><key>Hour</key><integer>20</integer><key>Minute</key><integer>0</integer></dict>
	</array>
	<key>StandardOutPath</key>
	<string>/tmp/redpanda-llm-tracker.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/redpanda-llm-tracker.err</string>
</dict>
</plist>
`))

// InstallAt writes the launchd plist to plistPath (used in tests and production).
func InstallAt(binaryPath, plistPath string) error {
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, map[string]string{
		"Label":      plistLabel,
		"BinaryPath": binaryPath,
	}); err != nil {
		return err
	}
	return os.WriteFile(plistPath, buf.Bytes(), 0644)
}

// Install writes the plist to ~/Library/LaunchAgents/ and loads it.
func Install(binaryPath string) error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", plistLabel+".plist")
	if err := InstallAt(binaryPath, plistPath); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

// Uninstall unloads and removes the plist.
func Uninstall() error {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", plistLabel+".plist")
	exec.Command("launchctl", "unload", plistPath).Run() // ignore — may not be loaded
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}
