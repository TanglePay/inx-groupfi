package daemon

const (
	PriorityDisconnectINX = iota // no dependencies
	PriorityCloseIMDatabase
	PriorityStopIMInit
	PriorityStopIMTTLCleaning
	PriorityStopIMLedgerConfirmedUpdate
	PriorityStopIMLedgerBlockUpdate
	PriorityStopIMAPI
	PriorityStopIMMQTT
)
