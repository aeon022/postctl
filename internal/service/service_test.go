package service

import (
	"strings"
	"testing"
)

func TestPlistAndLabel(t *testing.T) {
	if Label("") != "com.abteilung83.postctl" || Label("bruecke") != "com.abteilung83.postctl.bruecke" {
		t.Fatal("label")
	}
	d := plist("/bin/postctl", "")
	if strings.Contains(d, "--profile") || !strings.Contains(d, "<string>daemon</string>") {
		t.Errorf("default plist:\n%s", d)
	}
	b := plist("/a&b/postctl", "bruecke")
	if !strings.Contains(b, "<string>--profile</string>\n\t\t<string>bruecke</string>\n\t\t<string>daemon</string>") || !strings.Contains(b, "/a&amp;b/postctl") {
		t.Errorf("profile plist:\n%s", b)
	}
}
