package tmuxUtils

var SelectedHosts []string

var tmuxId string

type Mode int

const (
	TailedPane Mode = iota
	SyncedTailedPane
	Window
)
