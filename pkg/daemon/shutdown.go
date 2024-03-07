package daemon

const (
	PriorityDisconnectINX = iota // no dependencies
	PriorityCloseIMDatabase
	PriorityStopIMInit
	PriorityStopIMLedgerConfirmedUpdate
	PriorityStopIMLedgerBlockUpdate
	PriorityStopIMAPI
	PriorityStopIMMQTT
)
