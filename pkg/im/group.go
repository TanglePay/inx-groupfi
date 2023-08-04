package im

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
)

const IcebergGroup = "iceberg"

func IssuerBech32AddressToGroupId(address string) []byte {
	iceberg := map[string]string{
		"smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp": "iceberg-collection-1",
		"smr1zpz3430fdn4zmheenyjvughsu44ykjzu5st6hg2rp609eevz6czlye60pe7": "iceberg-collection-2",
		"smr1zpqndszdf0p9qy04kq7un5clgzptclqeyv70av5q8thjgxcmk2wfy7pspe5": "iceberg-collection-3",
		"smr1zp46qmajxu0vxc2l73tx0g2j7u579jdzgeplfcnwq6ef0hh3a8zt7wnx9dv": "iceberg-collection-4",
		"smr1zqa6juwmk7lad4rxsddqeprrz60zksdd0k3xa37lelthzsxsal6vjygkl9e": "iceberg-collection-5",
		"smr1zptkmnyuxxvk2qv8exqmyxcytmcf74j3t4apc3hfg4h6n9pnfun5q26j6w4": "iceberg-collection-6",
		"smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf": "iceberg-collection-7",
		"smr1zpvjkgxkzrhyvxy5nh20j6wm0l7grkf5s6l7r2mrhyspvx9khcaysmam589": "iceberg-collection-8",
	}
	groupName := iceberg[address]
	groupId := GroupNameToGroupId(groupName)

	return groupId
}

func GroupNameToGroupMeta(group string) map[string]string {
	m := map[string]map[string]string{
		"iceberg": {
			"groupName":     "iceberg",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-1": {
			"groupName":     "iceberg-collection-1",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-2": {
			"groupName":     "iceberg-collection-2",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-3": {
			"groupName":     "iceberg-collection-3",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-4": {
			"groupName":     "iceberg-collection-4",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-5": {
			"groupName":     "iceberg-collection-5",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-6": {
			"groupName":     "iceberg-collection-6",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-7": {
			"groupName":     "iceberg-collection-7",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-8": {
			"groupName":     "iceberg-collection-8",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"smr-whale": {
			"groupName":     "smr-whale",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
	}
	return m[group]
}
func sortAndSha256Map(m map[string]string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedMap := make(map[string]string)
	for _, k := range keys {
		sortedMap[k] = m[k]
	}
	b, _ := json.Marshal(sortedMap)
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}
func GroupNameToGroupId(group string) []byte {
	return sortAndSha256Map(GroupNameToGroupMeta(group))
}

func (im *Manager) GroupNameToGroupId(group string) []byte {
	return sortAndSha256Map(GroupNameToGroupMeta(group))
}
