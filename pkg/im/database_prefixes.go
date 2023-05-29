package im

const (
	// Holds the database status.
	ImStoreKeyPrefixStatus byte = 0

	// Holds the events.
	ImStoreKeyPrefixEvents byte = 1

	// Holds the blocks containing participations.
	ImStoreKeyPrefixBlocks byte = 2

	// Tracks all active and past participations.
	ImStoreKeyPrefixTrackedOutputs         byte = 3
	ImStoreKeyPrefixTrackedSpentOutputs    byte = 4
	ImStoreKeyPrefixTrackedOutputByAddress byte = 5
)
