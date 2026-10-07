package tui

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Every Tr("key") used in the package must exist in translationsMap with an
// English text — a missing key otherwise renders as the raw key (the
// dashboard once showed "stats_posted0").
func TestEveryTrKeyIsTranslated(t *testing.T) {
	// Tr("key"), hint("k", "hint_key") and the {"k", "help_key"} rows of the help panel
	re := regexp.MustCompile(`\bTr\("([a-zA-Z0-9_]+)"\)|\bhint\("[^"]*", "([a-zA-Z0-9_]+)"\)|\{"[^"]*", "((?:help|hint)_[a-zA-Z0-9_]+)"\}`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, _ := os.ReadFile(e.Name())
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			for _, k := range m[1:] {
				if k != "" {
					used[k] = e.Name()
				}
			}
		}
	}
	if len(used) < 20 {
		t.Fatalf("scan found only %d keys — regexp or layout changed", len(used))
	}
	for key, file := range used {
		tr, ok := translationsMap[key]
		if !ok || tr["en"] == "" {
			t.Errorf("Tr(%q) in %s has no English translation", key, file)
		}
	}
}
