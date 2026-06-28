package tui

import (
	"fmt"
	"testing"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsEnterKeyWithConfig(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	// Vorbereitung der Credentials, damit direkt der Auth-Flow gestartet wird
	config.ActiveConfig.Bluesky.Handle = "test-handle"
	config.ActiveConfig.Bluesky.AppPassword = "test-password"

	m := NewModel(s)
	m.activeTab = 5 // Settings
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
		t.Errorf("expected error since handle/password are fake, got nil")
	} else {
		fmt.Printf("Success: Got expected error for Bluesky Auth: %v\n", authRes.err)
	}
}

func TestSettingsEnterKeyNeedsSetup(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	// Credentials leeren, damit der Setup-Wizard gestartet wird
	config.ActiveConfig.Bluesky.Handle = ""
	config.ActiveConfig.Bluesky.AppPassword = ""

	m := NewModel(s)
	m.activeTab = 5 // Settings
	m.loading = false

	// Test case for Bluesky (cursor 9)
	m.cursor = 9
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	
	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	// Beim Setup-Wizard wird loading NICHT auf true gesetzt (die TUI wird stattdessen pausiert/suspendiert)
	if updatedModel.loading {
		t.Errorf("expected loading to remain false when starting setup wizard, got true")
	}

	if cmd == nil {
		t.Errorf("expected cmd to be returned, got nil")
	}

	// Das Kommando sollte vom Typ tea.execMsg sein, da tea.ExecProcess zurückgegeben wird
	// Da tea.execMsg im bubbletea-Paket nicht exportiert ist, können wir den Typ nicht direkt prüfen,
	// aber wir können sicherstellen, dass cmd nicht nil ist und das Modell im korrekten Zustand bleibt.
}

