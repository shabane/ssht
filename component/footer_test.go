package component

import (
	"strings"
	"testing"
)

func TestUpdateFooter_K9sShortcuts(t *testing.T) {
	newFooter()

	UpdateFooter(0)
	text := Footer.GetText(true)
	expectedShortcuts := []string{"<s> Shell", "<d> Describe", "<e> Edit", "<w> Direct", "<a> Tiled", "<o> Sync", "</> Search", "<:> Cmd"}
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
}
