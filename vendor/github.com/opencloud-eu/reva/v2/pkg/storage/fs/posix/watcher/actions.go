package watcher

type EventAction int

const (
	ActionCreate EventAction = iota
	ActionUpdate
	ActionMove
	ActionDelete
	ActionMoveFrom
)
