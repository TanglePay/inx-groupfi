package im

type CommonHeader struct {
	SchemaVersion uint8
	IsActAsSelf   bool
}

func DeserializeCommonHeader(data []byte, idx *int) (*CommonHeader, error) {
	schemaBytes, err := ReadBytesWithUint16Len(data, idx, 1)
	if err != nil {
		return nil, err
	}
	isActAsSelf := false
	schemaVersion := schemaBytes[0]
	if schemaVersion > 4 {
		isActAsSelfBytes, err := ReadBytesWithUint16Len(data, idx, 1)
		if err != nil {
			return nil, err
		}
		isActAsSelf = BytesToBool(isActAsSelfBytes)
	}
	return &CommonHeader{
		SchemaVersion: schemaVersion,
		IsActAsSelf:   isActAsSelf,
	}, nil

}
