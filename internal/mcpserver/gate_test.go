package mcpserver

import "testing"

func TestPublishIsOptIn(t *testing.T) {
	t.Setenv(allowPublishEnv, "")
	if publishAllowed() {
		t.Fatal("publishing through MCP must be off by default")
	}
	t.Setenv(allowPublishEnv, "1")
	if !publishAllowed() {
		t.Fatal("explicit opt-in enables it")
	}
}
