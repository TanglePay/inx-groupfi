package im

const (
	// Holds the database status.
	ImStoreKeyPrefixStatus byte = 0

	ImStoreKeyPrefixInitStatusFinish byte = 9
	ImStoreKeyPrefixInitStatusOffset byte = 10

	// Holds the Message.
	ImStoreKeyPrefixMessage byte = 11

	// Holds the NFT.
	ImStoreKeyPrefixNFT byte = 12

	// Holds the Shared
	ImStoreKeyPrefixShared byte = 13

	// Holds the Inbox
	ImStoreKeyPrefixInbox byte = 14

	// Holds token
	ImStoreKeyPrefixToken byte = 15

	ImStoreKeyPrefixAddressGroup = 16

	// Holds the Shared for consolidation
	ImStoreKeyPrefixSharedForConsolidation byte = 17

	// Holds the Message for consolidation
	ImStoreKeyPrefixMessageForConsolidation byte = 18

	// holds the group config
	ImStoreKeyPrefixGroupConfig byte = 19

	// holds address public key
	ImStoreKeyPrefixAddressPublicKey byte = 20

	ImStoreKeyPrefixGroupPublicKeyCount byte = 21

	// holds mark group address
	ImStoreKeyPrefixGroupMark byte = 22

	// holds the group address muted by address
	ImStoreKeyPrefixUserMuteGroupMember byte = 23

	// holds the group blacklist
	ImStoreKeyPrefixGroupBlacklist byte = 24

	ImStoreKeyPrefixVote byte = 25

	ImStoreKeyPrefixGroupMember byte = 26

	ImStoreKeyPrefixGroupQualification byte = 27

	ImStoreKeyPrefixMemberGroup byte = 28

	ImStoreKeyPrefixGroupUserReputation byte = 29

	ImStoreKeyPrefixUserGroupReputation byte = 30

	ImStoreKeyPrefixAddressMark byte = 31

	ImStoreKeyPrefixDid byte = 32

	ImStoreKeyPrefixPairXData byte = 33

	ImStoreKeyPrefixPairXEvmAddressSceneryProxyAddress byte = 34

	ImStoreKeyPrefixPairXProxyAddressEvmAddress byte = 35
)
