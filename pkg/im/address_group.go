package im

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/core/kvstore"
)

const (
	GroupQualifyTypeNft = iota
	GroupQualifyTypeToken
)

type AddressGroup struct {
	// address group id
	GroupId          []byte
	AddressSha256    []byte
	GroupName        string
	GroupQualifyType int
	NftLink          string
	TokenType        uint16
	TokenThres       string
}

// new address group from address in bytes not hashed yet and group id
func NewAddressGroup(address []byte, groupId []byte) *AddressGroup {
	return &AddressGroup{
		GroupId:       groupId,
		AddressSha256: Sha256HashBytes(address),
	}
}

// new address group from address in bytes not hashed yet and group id, plus fact that group qualify type is GroupQualifyTypeNft, and nft link and group name is provided
func NewAddressGroupNft(address []byte, groupId []byte, nftLink string, groupName string) *AddressGroup {
	return &AddressGroup{
		GroupId:          groupId,
		AddressSha256:    Sha256HashBytes(address),
		GroupQualifyType: GroupQualifyTypeNft,
		NftLink:          nftLink,
		GroupName:        groupName,
	}
}

// new address group from address in bytes not hashed yet and group id, plus fact that group qualify type is GroupQualifyTypeToken, and token type and token threshold is provided
func NewAddressGroupToken(address []byte, groupId []byte, tokenType uint16, tokenThres string) *AddressGroup {
	return &AddressGroup{
		GroupId:          groupId,
		AddressSha256:    Sha256HashBytes(address),
		GroupQualifyType: GroupQualifyTypeToken,
		TokenType:        tokenType,
		TokenThres:       tokenThres,
	}
}

// serialize address group to bytes, using AppendBytesWithUint16Len, appendLength is false when length is fixed
func (im *Manager) GetAddressGroupValue(addressGroup *AddressGroup) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(addressGroup.GroupName), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{byte(addressGroup.GroupQualifyType)}, false)

	if addressGroup.GroupQualifyType == GroupQualifyTypeNft {
		AppendBytesWithUint16Len(&bytes, &idx, []byte(addressGroup.NftLink), true)
	} else if addressGroup.GroupQualifyType == GroupQualifyTypeToken {
		tmp := make([]byte, 2)
		binary.BigEndian.PutUint16(tmp, addressGroup.TokenType)
		AppendBytesWithUint16Len(&bytes, &idx, tmp, false)
		AppendBytesWithUint16Len(&bytes, &idx, []byte(addressGroup.TokenThres), true)
	}
	return bytes
}

func (im *Manager) ParseAddressGroupValue(bytes []byte) (*AddressGroup, error) {

	idx := 0

	groupName, err := ReadBytesWithUint16Len(bytes, &idx)
	if err != nil {
		return nil, err
	}
	groupQualifyType, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	addressGroup := &AddressGroup{
		GroupName:        string(groupName),
		GroupQualifyType: int(groupQualifyType[0]),
	}
	if addressGroup.GroupQualifyType == GroupQualifyTypeNft {
		nftLink, err := ReadBytesWithUint16Len(bytes, &idx)
		if err != nil {
			return nil, err
		}
		addressGroup.NftLink = string(nftLink)
	} else if addressGroup.GroupQualifyType == GroupQualifyTypeToken {
		TokenTypeBytes, err := ReadBytesWithUint16Len(bytes, &idx, 2)
		if err != nil {
			return nil, err
		}
		tokenType := binary.BigEndian.Uint16(TokenTypeBytes)
		addressGroup.TokenType = tokenType
		tokenThres, err := ReadBytesWithUint16Len(bytes, &idx)
		if err != nil {
			return nil, err
		}
		addressGroup.TokenThres = string(tokenThres)
	}
	return addressGroup, nil
}

// key = prefix + addressSha256Hash + groupid
func (im *Manager) AddressGroupKey(addressGroup *AddressGroup) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressGroup
	index++
	copy(key[index:], addressGroup.AddressSha256)
	index += Sha256HashLen
	copy(key[index:], addressGroup.GroupId)
	return key
}

// get address hash and group id from key
func (im *Manager) AddressGroupKeyToAddressAndGroupId(key []byte) ([]byte, []byte) {
	addressSha256 := key[1 : 1+Sha256HashLen]
	groupId := key[1+Sha256HashLen:]
	return addressSha256, groupId
}

// prefix = prefix + addressSha256Hash
func (im *Manager) AddressGroupKeyPrefix(addressSha256 []byte) []byte {
	key := make([]byte, 1+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressGroup
	index++
	copy(key[index:], addressSha256)
	return key
}
func (im *Manager) StoreAddressGroup(addressGroup *AddressGroup) error {
	key := im.AddressGroupKey(addressGroup)
	value := im.GetAddressGroupValue(addressGroup)
	return im.imStore.Set(key, value)
}

// delete address group
func (im *Manager) DeleteAddressGroup(addressGroup *AddressGroup) error {
	key := im.AddressGroupKey(addressGroup)
	return im.imStore.Delete(key)
}

// get groupIds from address
func (im *Manager) GetGroupIdsFromAddress(addressSha256 []byte) ([][]byte, error) {
	prefix := im.AddressGroupKeyPrefix(addressSha256)
	groupIds := make([][]byte, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupId := key[1+Sha256HashLen:]
		groupIds = append(groupIds, groupId)
		return true
	})
	return groupIds, err
}

// get full data of address group from address
func (im *Manager) GetAddressGroupFromAddress(addressSha256 []byte) ([]*AddressGroup, error) {
	prefix := im.AddressGroupKeyPrefix(addressSha256)
	addressGroups := make([]*AddressGroup, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		addressGroup, err := im.ParseAddressGroupValue(value)
		//call AddressGroupKeyToAddressAndGroupId
		addressSha256, groupId := im.AddressGroupKeyToAddressAndGroupId(key)
		addressGroup.GroupId = groupId
		addressGroup.AddressSha256 = addressSha256
		if err != nil {
			return false
		}
		addressGroups = append(addressGroups, addressGroup)
		return true
	})
	return addressGroups, err
}
