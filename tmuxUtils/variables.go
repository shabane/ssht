package tmuxUtils

var tmuxId string

type Mode int

const (
	TailedPane Mode = iota
	SyncedTailedPane
	Window
)
