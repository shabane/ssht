package component

import (
	"sort"
	"strings"

	"ssht/sshUtils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var baseCommands = []string{
	":proxy", ":bastion", ":fwd", ":forward", ":tunnel",
	":badkeys", ":keys", ":cmd", ":reload", ":quit",
	":help", ":ping", ":select all", ":select ", ":clear",
	":deselect", ":sort name", ":sort ip", ":sort status",
	":all", ":tag ",
}

// AutocompleteInputField embeds tview.InputField with inline ghost-text
// suggestions and Tab autocompletion for colon (:) mode commands.
type AutocompleteInputField struct {
	*tview.InputField
	ghostSuffix      string
	suggestions      []string
	suggIndex        int
	isAutocompleting bool
}

var SearchBox *AutocompleteInputField
var prevSearchText string

func init() {
	newSearchBox()
}

func newSearchBox() *AutocompleteInputField {
	if SearchBox == nil {
		field := tview.NewInputField().SetPlaceholder("Search host, #tag, IP, user... | Esc to cancel")
		field.SetBorder(true).SetBorderColor(tcell.GetColor("#42f5aa"))
		field.SetFieldBackgroundColor(tcell.ColorDefault)
		SearchBox = &AutocompleteInputField{
			InputField: field,
		}
	}

	return SearchBox
}

func getUniqueTags() []string {
	seen := make(map[string]string)
	for _, meta := range sshUtils.HostsMeta {
		for _, tag := range meta.Tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				lower := strings.ToLower(trimmed)
				if _, exists := seen[lower]; !exists {
					seen[lower] = trimmed
				}
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for _, tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// GetCommandSuggestions returns matching command or parameter suggestions
// for an input string starting with ':'.
func GetCommandSuggestions(input string) []string {
	if !strings.HasPrefix(input, ":") {
		return nil
	}

	lower := strings.ToLower(input)

	// Dynamic tag completion: if input starts with :tag or :t
	if strings.HasPrefix(lower, ":tag ") || strings.HasPrefix(lower, ":t ") {
		var prefix string
		if strings.HasPrefix(lower, ":tag ") {
			prefix = strings.TrimSpace(input[5:])
		} else {
			prefix = strings.TrimSpace(input[3:])
		}
		var suggs []string
		for _, tag := range getUniqueTags() {
			if prefix == "" || strings.HasPrefix(strings.ToLower(tag), strings.ToLower(prefix)) {
				suggs = append(suggs, ":tag "+tag)
			}
		}
		return suggs
	}

	// Dynamic select completion: if input starts with :select
	if strings.HasPrefix(lower, ":select ") {
		prefix := strings.TrimSpace(input[8:])
		var suggs []string
		if prefix == "" || strings.HasPrefix("all", strings.ToLower(prefix)) {
			suggs = append(suggs, ":select all")
		}
		for _, tag := range getUniqueTags() {
			if strings.ToLower(tag) == "all" {
				continue
			}
			if prefix == "" || strings.HasPrefix(strings.ToLower(tag), strings.ToLower(prefix)) {
				suggs = append(suggs, ":select "+tag)
			}
		}
		return suggs
	}

	// Dynamic sort completion: if input starts with :sort
	if strings.HasPrefix(lower, ":sort ") {
		prefix := strings.TrimSpace(input[6:])
		sortOptions := []string{":sort name", ":sort ip", ":sort status"}
		var suggs []string
		for _, opt := range sortOptions {
			sub := opt[6:]
			if prefix == "" || strings.HasPrefix(sub, strings.ToLower(prefix)) {
				suggs = append(suggs, opt)
			}
		}
		return suggs
	}

	// Base commands: case-insensitive prefix match
	var suggs []string
	for _, cmd := range baseCommands {
		if strings.HasPrefix(strings.ToLower(cmd), lower) {
			suggs = append(suggs, cmd)
		}
	}
	return suggs
}

// UpdateSuggestions computes matching suggestions and updates the ghost suffix.
func (a *AutocompleteInputField) UpdateSuggestions(text string) {
	if a.isAutocompleting {
		return
	}
	if !strings.HasPrefix(text, ":") {
		a.suggestions = nil
		a.ghostSuffix = ""
		a.suggIndex = 0
		return
	}

	a.suggestions = GetCommandSuggestions(text)
	a.suggIndex = 0

	if len(a.suggestions) > 0 {
		first := a.suggestions[0]
		if strings.HasPrefix(strings.ToLower(first), strings.ToLower(text)) && len(first) >= len(text) {
			a.ghostSuffix = first[len(text):]
		} else {
			a.ghostSuffix = ""
		}
	} else {
		a.ghostSuffix = ""
	}
}

// AutocompleteNext advances to the next suggestion in the cycle and updates input text.
func (a *AutocompleteInputField) AutocompleteNext() bool {
	if len(a.suggestions) == 0 {
		return false
	}
	nextText := a.suggestions[a.suggIndex]
	a.suggIndex = (a.suggIndex + 1) % len(a.suggestions)
	a.isAutocompleting = true
	a.SetText(nextText)
	a.isAutocompleting = false
	a.ghostSuffix = ""
	return true
}

// AcceptGhost accepts the visible inline ghost suffix and appends it to the input text.
func (a *AutocompleteInputField) AcceptGhost() bool {
	if a.ghostSuffix != "" {
		a.isAutocompleting = true
		a.SetText(a.GetText() + a.ghostSuffix)
		a.isAutocompleting = false
		a.ghostSuffix = ""
		return true
	}
	return false
}

// Draw overrides InputField.Draw to render the inline ghost-text placeholder.
func (a *AutocompleteInputField) Draw(screen tcell.Screen) {
	if screen == nil {
		return
	}
	a.InputField.Draw(screen)
	if a.ghostSuffix != "" && strings.HasPrefix(a.GetText(), ":") {
		x, y, width, _ := a.GetInnerRect()
		labelWidth := tview.TaggedStringWidth(a.GetLabel())
		textWidth := tview.TaggedStringWidth(a.GetText())
		startX := x + labelWidth + textWidth
		available := width - labelWidth - textWidth
		if available > 0 && startX < x+width {
			tview.Print(screen, a.ghostSuffix, startX, y, available, tview.AlignLeft, tcell.GetColor("#38bdf8"))
		}
	}
}

func init() {
	SearchBox.SetChangedFunc(func(text string) {
		SearchBox.UpdateSuggestions(text)

		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, ":") {
			prevSearchText = trimmed
			return
		}
		if strings.HasPrefix(prevSearchText, ":") && trimmed == "" {
			prevSearchText = ""
			return
		}
		prevSearchText = trimmed

		Clear()
		for _, host := range sshUtils.SearchHostname(sshUtils.AllHosts, text) {
			h := host
			var fn func()
			if DefaultSelectFunc != nil {
				fn = DefaultSelectFunc(h)
			}
			AddItem(h, "", 0, fn)
		}
		RequestPing()
	})
}
