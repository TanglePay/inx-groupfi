package daemon

const (
	PriorityDisconnectINX = iota // no dependencies
	PriorityCloseIMDatabase
	PriorityStopIM
	PriorityStopIMAPI
)
