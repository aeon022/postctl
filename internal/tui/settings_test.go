package tui

import (
	"fmt"
	"testing"

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsEnterKey(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	m := NewModel(s)
	m.activeTab = 4 // Settings
	m.loading = false

	// Test case for Bluesky (cursor 9)
	m.cursor = 9
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	if !updatedModel.loading {
		t.Errorf("expected loading to be true after hitting Enter on Bluesky, got false")
	}

	if cmd == nil {
		t.Errorf("expected cmd to be returned, got nil")
	}

	// Trigger the command to see if it executes without crashing
	// (it should run in background and return authResultMsg)
	resMsg := cmd()
	authRes, ok := resMsg.(authResultMsg)
	if !ok {
		t.Fatalf("expected command result to be authResultMsg, got %T", resMsg)
	}

	if authRes.platform != models.PlatformBluesky {
		t.Errorf("expected platform to be %s, got %s", models.PlatformBluesky, authRes.platform)
	}

	if authRes.err == nil {
		t.Errorf("expected error since config is empty, got nil")
	} else {
		fmt.Printf("Success: Got expected error for Bluesky Auth: %v\n", authRes.err)
	}
}
