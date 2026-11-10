package component

import (
	"strings"
	"testing"
)

func TestUpdateFooter_K9sShortcuts(t *testing.T) {
	newFooter()

	UpdateFooter(0)
	text := Footer.GetText(true)
	expectedShortcuts := []string{"<s> Shell", "<d> Describe", "<e> Edit", "<w> Direct", "<a> Tiled", "<o> Sync", "</> Search", "<:> Cmd", "<<> Run"}
	for _, sc := range expectedShortcuts {
		if !strings.Contains(text, sc) {
			t.Errorf("expected footer to contain %q, got: %q", sc, text)
		}
	}
	if strings.Contains(text, "selected") {
		t.Errorf("footer should not contain 'selected' when count is 0, got: %q", text)
	}

	UpdateFooter(3)
	textWithSelection := Footer.GetText(true)
	if !strings.Contains(textWithSelection, "(3 selected) |") {
		t.Errorf("expected footer to contain '(3 selected) |', got: %q", textWithSelection)
	}
	for _, sc := range expectedShortcuts {
		if !strings.Contains(textWithSelection, sc) {
			t.Errorf("expected footer with selection to contain %q, got: %q", sc, textWithSelection)
		}
	}

	// Test command mode hints
	ShowCommandFooterHints()
	cmdText := Footer.GetText(true)
	if !strings.Contains(cmdText, ":tag <name>") || !strings.Contains(cmdText, ":proxy") || !strings.Contains(cmdText, ":badkeys") {
		t.Errorf("expected command footer hints to contain directives, got: %q", cmdText)
	}

	// Test search mode hints
	ShowSearchFooterHints()
	searchText := Footer.GetText(true)
	if !strings.Contains(searchText, "Filter:") || !strings.Contains(searchText, "#tag") {
		t.Errorf("expected search footer hints, got: %q", searchText)
	}

	// Test remote command hints
	ShowRemoteCommandFooterHints(3, "")
	remoteTextMulti := Footer.GetText(true)
	if !strings.Contains(remoteTextMulti, "3 selected hosts") {
		t.Errorf("expected multi-target remote command hints, got: %q", remoteTextMulti)
	}

	ShowRemoteCommandFooterHints(1, "my-server")
	remoteTextSingle := Footer.GetText(true)
	if !strings.Contains(remoteTextSingle, "my-server") {
		t.Errorf("expected single-target remote command hints, got: %q", remoteTextSingle)
	}

	// Test restore
	RestoreNormalFooter()
	restoredText := Footer.GetText(true)
	if !strings.Contains(restoredText, "<s> Shell") {
		t.Errorf("expected restored footer to contain '<s> Shell', got: %q", restoredText)
	}
}
