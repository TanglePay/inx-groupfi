package im

import (
	"bytes"

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

// UnmarshalGroupStateSyncItem parses the serialized data and returns a GroupStateSyncItem.
func UnmarshalGroupStateSync(bytes []byte) (*GroupStateSync, error) {
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
	return NewGroupStateSync(schemaVersion, items), nil
}

// marshalGroupStateSyncStorage serializes the GroupStateSyncForStorage.
func marshalGroupStateSyncStorage(groupStateSync *GroupStateSyncForStorage) []byte {
	idx := 0
	var bytes []byte
	AppendBytesWithUint16Len(&bytes, &idx, []byte{groupStateSync.SchemaVersion}, false)
	// output id
	AppendBytesWithUint16Len(&bytes, &idx, groupStateSync.OutputId[:], false)
	// items length
	AppendBytesWithUint16Len(&bytes, &idx, Uint16ToBytes(uint16(len(groupStateSync.Items))), false)
	for _, item := range groupStateSync.Items {
		// group id
		AppendBytesWithUint16Len(&bytes, &idx, item.GroupId[:], false)
		// last time read latest message timestamp
		AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(item.LastTimeReadLatestMessageTimestamp), false)
	}
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
	outputIdBytes, err := ReadBytesWithUint16Len(bytes, &idx, OutputIdLen)
	if err != nil {
		return nil, err
	}
	var outputId [OutputIdLen]byte
	copy(outputId[:], outputIdBytes)
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
	return NewGroupStateSyncStorage(outputId, schemaVersion, items), nil
}

// key = prefix + address hash + output id
func GetGroupStateSyncKey(addressHash [Sha256HashLen]byte, outputId [OutputIdLen]byte) []byte {
	idx := 0
	var key []byte
	AppendBytesWithUint16Len(&key, &idx, []byte{ImStoreKeyPrefixGroupStateSync}, false)
	AppendBytesWithUint16Len(&key, &idx, addressHash[:], false)
	AppendBytesWithUint16Len(&key, &idx, outputId[:], false)
	return key
}

// store group state sync
func StoreGroupStateSync(groupStateSync *GroupStateSyncForStorage,
	address string,
	im *Manager) error {
	addressHash := Sha256HashFixedAddress(address)
	key := GetGroupStateSyncKey(addressHash, groupStateSync.OutputId)
	value := marshalGroupStateSyncStorage(groupStateSync)
	return im.imStore.Set(key, value)
}

// delete group state sync
func DeleteGroupStateSync(address string, im *Manager) error {
	keyPrefix := GetGroupStateSyncKeyPrefix(address)
	err := im.imStore.DeletePrefix(keyPrefix)
	return err
}

// keyprefix = prefix + address hash
func GetGroupStateSyncKeyPrefix(address string) []byte {
	addressHash := Sha256HashFixedAddress(address)
	idx := 0
	var key []byte
	AppendBytesWithUint16Len(&key, &idx, []byte{ImStoreKeyPrefixGroupStateSync}, false)
	AppendBytesWithUint16Len(&key, &idx, addressHash[:], false)
	return key
}

// get group state sync from address
func GetGroupStateSyncFromAddress(address string, im *Manager) (*GroupStateSyncForStorage, error) {
	keyPrefix := GetGroupStateSyncKeyPrefix(address)
	// iterate all group state sync
	var groupStateSyncs []*GroupStateSyncForStorage
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupStateSync, err := unmarshalGroupStateSyncStorage(value)
		if err != nil {
			Logger.Errorf("GetGroupStateSyncFromAddress unmarshalGroupStateSyncStorage error: %s", err)
		}
		groupStateSyncs = append(groupStateSyncs, groupStateSync)
		return true
	})
	if err != nil {
		return nil, err
	}
	if len(groupStateSyncs) == 0 {
		return nil, nil
	}
	return groupStateSyncs[0], nil
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

	if output.FeatureSet().MetadataFeature() == nil {
		return nil, "", false
	}

	// unmarshal group state sync
	groupStateSync, err := UnmarshalGroupStateSync(output.FeatureSet().MetadataFeature().Data)
	if err != nil {
		return nil, "", false
	}
	unlockConditionSet := output.UnlockConditionSet()
	smrAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	evmAddress := Im.ConvertAddressToActualAddress(smrAddress)

	return NewGroupStateSyncStorage(outputID, groupStateSync.SchemaVersion, groupStateSync.Items), evmAddress, true
}
