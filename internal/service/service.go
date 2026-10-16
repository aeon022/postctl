// Package service installs the postctl scheduler daemon as a macOS launchd
// agent, one per profile, so scheduled posts go out without a terminal open.
package service

import (
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Label is the launchd label for a profile ("" = default profile).
func Label(profile string) string {
	if profile == "" {
		return "com.abteilung83.postctl"
	}
	return "com.abteilung83.postctl." + profile
}

func plistPath(profile string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", Label(profile)+".plist")
}

func logPath(profile string) string {
	home, _ := os.UserHomeDir()
	name := "postctl"
	if profile != "" {
		name += "-" + profile
	}
	return filepath.Join(home, "Library", "Logs", name+".log")
}

func domain() string { return "gui/" + strconv.Itoa(os.Getuid()) }

// plist renders the launch agent: start at login, restart if it dies.
func plist(exe, profile string) string {
	args := []string{exe}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	args = append(args, "daemon")
	var sb strings.Builder
	for _, a := range args {
		sb.WriteString("\t\t<string>" + html.EscapeString(a) + "</string>\n")
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>` + Label(profile) + `</string>
	<key>ProgramArguments</key>
	<array>
` + sb.String() + `	</array>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><true/>
	<key>StandardOutPath</key><string>` + html.EscapeString(logPath(profile)) + `</string>
	<key>StandardErrorPath</key><string>` + html.EscapeString(logPath(profile)) + `</string>
</dict>
</plist>
`
}

func requireMac() error {
	if runtime.GOOS != "darwin" {
		return errors.New("scheduler service is only supported on macOS (launchd)")
	}
	return nil
}

func launchctl(args ...string) error {
	if out, err := exec.Command("launchctl", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl %s: %v: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Running reports whether the agent for the profile is loaded in launchd.
func Running(profile string) bool {
	return requireMac() == nil && exec.Command("launchctl", "print", domain()+"/"+Label(profile)).Run() == nil
}

// Install writes the plist for the current executable and (re)loads it.
func Install(profile string) error {
	if err := requireMac(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p := plistPath(profile)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath(profile)), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(plist(exe, profile)), 0o644); err != nil {
		return err
	}
	_ = launchctl("bootout", domain()+"/"+Label(profile)) // reload with the new plist; ok if not loaded
	return launchctl("bootstrap", domain(), p)
}

// Uninstall stops the agent and removes its plist.
func Uninstall(profile string) error {
	if err := requireMac(); err != nil {
		return err
	}
	if Running(profile) {
		if err := launchctl("bootout", domain()+"/"+Label(profile)); err != nil {
			return err
		}
	}
	if err := os.Remove(plistPath(profile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LogPath is where the daemon writes its output.
func LogPath(profile string) string { return logPath(profile) }
