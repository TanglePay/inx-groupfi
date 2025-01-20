package im

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

/*
import { Converter, ReadStream, WriteStream } from "@iota/util.js";
import { GroupStateSyncSchemaVersion, GroupStateSync, GroupStateSyncItem, GroupIDLength } from "./types";

// serialize group state sync
export function serializeGroupStateSync(items: GroupStateSyncItem[]): Uint8Array {
    const writer = new WriteStream();
    writer.writeUInt8("schema_version", GroupStateSyncSchemaVersion);
    writer.writeUInt16("items_length", items.length);
    for (const item of items) {
        writer.writeBytes("group_id", GroupIDLength,Converter.hexToBytes(item.groupId));
        writer.writeUInt32("lastTimeReadLatestMessageTimestamp", item.lastTimeReadLatestMessageTimestamp);
    }
    return writer.finalBytes();
}

// deserialize group state sync
export function deserializeGroupStateSync(bytes: Uint8Array): GroupStateSync {
    const reader = new ReadStream(bytes);
    const schemaVersion = reader.readUInt8("schema_version");
    const itemsLength = reader.readUInt16("items_length");
    const items = [];
    for (let i = 0; i < itemsLength; i++) {
        const groupId = Converter.bytesToHex(reader.readBytes("group_id", GroupIDLength), true);
        const lastTimeReadLatestMessageTimestamp = reader.readUInt32("lastTimeReadLatestMessageTimestamp");
        items.push({
            groupId,
            lastTimeReadLatestMessageTimestamp
        });
    }
    return {
        schemaVersion,
        items
    };
}*/

// tag
var groupStateSyncTagRawStr = "GROUPFIGROUPSTATESYNCV1"
var groupStateSyncTag = []byte(groupStateSyncTagRawStr)
var GroupStateSyncTagStr = iotago.EncodeHex(groupStateSyncTag)

// GroupStateSyncItem struct
type GroupStateSyncItem struct {
	GroupId                            [GroupIdLen]byte
	LastTimeReadLatestMessageTimestamp uint32
}

// GroupStateSync struct
type GroupStateSync struct {
	SchemaVersion uint8
	Items         []*GroupStateSyncItem
}

// GroupStateSyncForStorage struct
type GroupStateSyncForStorage struct {
	OutputId      [OutputIdLen]byte
	SchemaVersion uint8
	Items         []*GroupStateSyncItem
}

// NewGroupStateSync creates a new GroupStateSync.
func NewGroupStateSync(schemaVersion uint8, items []*GroupStateSyncItem) *GroupStateSync {
	return &GroupStateSync{
		SchemaVersion: schemaVersion,
		Items:         items,
	}
}

// NewGroupStateSyncStorage creates a new GroupStateSyncForStorage.
func NewGroupStateSyncStorage(outputId [OutputIdLen]byte, schemaVersion uint8, items []*GroupStateSyncItem) *GroupStateSyncForStorage {
	return &GroupStateSyncForStorage{
		OutputId:      outputId,
		SchemaVersion: schemaVersion,
		Items:         items,
	}
}

// new GroupStateSyncItem creates a new GroupStateSyncItem.
func NewGroupStateSyncItem(groupId [GroupIdLen]byte, lastTimeReadLatestMessageTimestamp uint32) *GroupStateSyncItem {
	return &GroupStateSyncItem{
		GroupId:                            groupId,
		LastTimeReadLatestMessageTimestamp: lastTimeReadLatestMessageTimestamp,
	}
}

// UnmarshalGroupStateSync parses the serialized data and returns a GroupStateSync.
func UnmarshalGroupStateSync(bytes []byte) (*GroupStateSync, error) {
	idx := 0

	// Read schema version (uint8)
	schemaVersionBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema version: %w", err)
	}
	schemaVersion := schemaVersionBytes[0]

	// Read items length (uint16)
	itemsLengthBytes, err := ReadBytesWithUint16Len(bytes, &idx, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to read items length: %w", err)
	}
	itemsLength := int(BytesToUint16(itemsLengthBytes))
	// log itemsLength, total bytes
	Logger.Infof("UnmarshalGroupStateSync itemsLength: %d, total bytes: %d", itemsLength, len(bytes))
	items := make([]*GroupStateSyncItem, itemsLength)
	for i := 0; i < itemsLength; i++ {
		// Read group ID
		groupIdBytes, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
		if err != nil {
			return nil, fmt.Errorf("failed to read group ID: %w", err)
		}
		var groupId [GroupIdLen]byte
		copy(groupId[:], groupIdBytes)

		// Read timestamp
		timestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
		if err != nil {
			return nil, fmt.Errorf("failed to read timestamp: %w", err)
		}
		lastTimeReadLatestMessageTimestamp := BytesToUint32(timestampBytes)

		items[i] = NewGroupStateSyncItem(groupId, lastTimeReadLatestMessageTimestamp)
	}

	return NewGroupStateSync(schemaVersion, items), nil
}

// marshalGroupStateSyncStorage serializes the GroupStateSyncForStorage.
func marshalGroupStateSyncStorage(groupStateSync *GroupStateSyncForStorage) []byte {
	idx := 0
	var bytes []byte
	AppendBytesWithUint16Len(&bytes, &idx, []byte{groupStateSync.SchemaVersion}, false)
	// items length
	AppendBytesWithUint16Len(&bytes, &idx, Uint16ToBytes(uint16(len(groupStateSync.Items))), false)
	for _, item := range groupStateSync.Items {
		// group id
		AppendBytesWithUint16Len(&bytes, &idx, item.GroupId[:], false)
		// last time read latest message timestamp
		AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(item.LastTimeReadLatestMessageTimestamp), false)
	}
	// output id at the end
	AppendBytesWithUint16Len(&bytes, &idx, groupStateSync.OutputId[:], false)
	return bytes
}

// unmarshalGroupStateSyncStorage parses the serialized data and returns a GroupStateSyncForStorage.
func unmarshalGroupStateSyncStorage(bytes []byte) (*GroupStateSyncForStorage, error) {
	idx := 0
	schemaVersionBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	schemaVersion := schemaVersionBytes[0]

	itemsLengthBytes, err := ReadBytesWithUint16Len(bytes, &idx, 2)
	if err != nil {
		return nil, err
	}
	itemsLength := int(BytesToUint16(itemsLengthBytes))
	items := make([]*GroupStateSyncItem, itemsLength)
	for i := 0; i < itemsLength; i++ {
		groupIdBytes, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
		if err != nil {
			return nil, err
		}
		var groupId [GroupIdLen]byte
		copy(groupId[:], groupIdBytes)
		lastTimeReadLatestMessageTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
		if err != nil {
			return nil, err
		}
		lastTimeReadLatestMessageTimestamp := BytesToUint32(lastTimeReadLatestMessageTimestampBytes)
		items[i] = NewGroupStateSyncItem(groupId, lastTimeReadLatestMessageTimestamp)
	}

	// Read output id from the end
	outputIdBytes, err := ReadBytesWithUint16Len(bytes, &idx, OutputIdLen)
	if err != nil {
		return nil, err
	}
	var outputId [OutputIdLen]byte
	copy(outputId[:], outputIdBytes)

	return NewGroupStateSyncStorage(outputId, schemaVersion, items), nil
}

// key = prefix + address hash
func GetGroupStateSyncKey(addressHash [Sha256HashLen]byte) []byte {
	idx := 0
	var key []byte
	AppendBytesWithUint16Len(&key, &idx, []byte{ImStoreKeyPrefixGroupStateSync}, false)
	AppendBytesWithUint16Len(&key, &idx, addressHash[:], false)
	return key
}

// store group state sync
func StoreGroupStateSync(groupStateSync *GroupStateSyncForStorage,
	address string,
	im *Manager) error {
	// log store
	Logger.Infof("StoreGroupStateSync store group state sync: %s", iotago.EncodeHex(groupStateSync.OutputId[:]))
	addressHash := Sha256HashFixedAddress(address)
	key := GetGroupStateSyncKey(addressHash)
	value := marshalGroupStateSyncStorage(groupStateSync)
	return im.imStore.Set(key, value)
}

// delete group state sync
func DeleteGroupStateSync(address string, im *Manager) error {
	addressHash := Sha256HashFixedAddress(address)
	key := GetGroupStateSyncKey(addressHash)
	return im.imStore.Delete(key)
}

// get group state sync from address
func GetGroupStateSyncFromAddress(address string, im *Manager) (*GroupStateSyncForStorage, error) {
	addressHash := Sha256HashFixedAddress(address)
	key := GetGroupStateSyncKey(addressHash)

	value, err := im.imStore.Get(key)
	if err != nil {
		if err == kvstore.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	return unmarshalGroupStateSyncStorage(value)
}

// filter output, check if output is group state sync
func FilterGroupStateSyncOutput(output iotago.Output,
	outputID [OutputIdLen]byte,
	im *Manager) (*GroupStateSyncForStorage, string, bool) {
	if output.FeatureSet().TagFeature() == nil ||
		output.FeatureSet().TagFeature().Tag == nil ||
		!bytes.Equal(output.FeatureSet().TagFeature().Tag, groupStateSyncTag) {
		return nil, "", false
	}
	// log found
	Logger.Infof("FilterGroupStateSyncOutput found group state sync output: %s", iotago.EncodeHex(outputID[:]))
	if output.FeatureSet().MetadataFeature() == nil {
		return nil, "", false
	}
	// log has metadata
	Logger.Infof("FilterGroupStateSyncOutput has metadata: %s", iotago.EncodeHex(output.FeatureSet().MetadataFeature().Data))
	// unmarshal group state sync
	groupStateSync, err := UnmarshalGroupStateSync(output.FeatureSet().MetadataFeature().Data)
	if err != nil {
		Logger.Errorf("FilterGroupStateSyncOutput unmarshalGroupStateSync error: %s", err)
		return nil, "", false
	}
	// log unmarshal group state sync success
	Logger.Infof("FilterGroupStateSyncOutput unmarshal group state sync success: %s", iotago.EncodeHex(output.FeatureSet().MetadataFeature().Data))
	unlockConditionSet := output.UnlockConditionSet()
	smrAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	evmAddress := Im.ConvertAddressToActualAddress(smrAddress)
	// log got evm address
	Logger.Infof("FilterGroupStateSyncOutput got evm address: %s", evmAddress)
	return NewGroupStateSyncStorage(outputID, groupStateSync.SchemaVersion, groupStateSync.Items), evmAddress, true
}
