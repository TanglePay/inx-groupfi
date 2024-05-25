package im

import (
	"fmt"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	CONFIG_IN_TEXT = fmt.Sprintf(`[

		    {"groupName":"soon-whale","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "chainId":0,"collectionId":"","tokenThres":"0.00001","tokenId":"0x0884298fe9b82504d26ddb873dbd234a344c120da3a4317d8063dbcf96d356aa9d0100000000"},
			{"groupName":"soon","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "chainId":0,"collectionId":"","tokenThres":"0.0000002","tokenId":"0x0884298fe9b82504d26ddb873dbd234a344c120da3a4317d8063dbcf96d356aa9d0100000000"},
			{"groupName":"smr-whale","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "chainId":0,"collectionId":"","tokenThres":"0.000000636","tokenId":"%s"},
		    {"groupName":"smr","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "chainId":0,"collectionId":"","tokenThres":"0.0000000127","tokenId":"%s"},
		    {"groupName":"staff-marketing","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x3070481ff0e0d96b1b7f6cbf8c2a484c9e7304295b44cd6ff9afe1ecbc4efca9"},
		    {"groupName":"staff-developer","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0xdc6c3b0167af767652567523b9240c86095241c622c37cb726efeeb5e102a93c"},
		    {"groupName":"dapper-groupfi","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x1fd2407145ef147e0b06f835fef0e2059e56899aa8ce80147506893f837ea606"},
			{"groupName":"GroupFi Announcement","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":148,"collectionId":"0x544F353C02363D848dBAC8Dc3a818B36B7f9355e"},
			{"groupName":"EtherVisions","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":148,"collectionId":"0xB85bdaf5eFf3f4c84d565923Eb1D62717dE17297"},
			{"groupName":"TOKEN","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "chainId":148,"collectionId":"","tokenThres":"1","tokenId":"0xfDbc4c5b14A538Aa2F6cD736b525C8e9532C5FA6"},
			{"groupName":"GroupFi Announcement","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x115a9fdca6a2110cc3e4cc7f92555101c3d80addf56984003e51e004ac4f9148"},
			{"groupName":"alpha-test","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x23e5f8500132f9dfa8698e3d352f0d57bd79cf57533f273e87e31b6cd0e0a5ef"},
		   	{"groupName":"iceberg-1","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x064d0eaefb86a94eb326ff633c22cdf744decca954bb93b1572b449d324ae717"},
		   	{"groupName":"iceberg-2","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x451ac5e96cea2ddf399924ce22f0e56a4b485ca417aba1430e9e5ce582d605f2"},
		   	{"groupName":"iceberg-3","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x4136c04d4bc25011f5b03dc9d31f4082bc7c19233cfeb2803aef241b1bb29c92"},
		   	{"groupName":"iceberg-4","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x6ba06fb2371ec3615ff45667a152f729e2c9a24643f4e26e06b297def1e9c4bf"},
		   	{"groupName":"iceberg-5","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x3ba971dbb7bfd6d466835a0c8463169e2b41ad7da26ec7dfcfd77140d0eff4c9"},
		   	{"groupName":"iceberg-6","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x576dcc9c3199650187c981b21b045ef09f56515d7a1c46e9456fa994334f2740"},
		   	{"groupName":"iceberg-7","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0xcf0f598ff3ee378b03906af4de48030bc6082831dfcf67730be780a317d98265"},
		   	{"groupName":"iceberg-8","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x592b20d610ee4618949dd4f969db7ffc81d93486bfe1ab63b9201618b6be3a48"},
			{"groupName":"iceberg-9","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainId":0,"collectionId":"0x560b391d65225d159841ecb6e9a0d60364eac9aab31ced13c27fc47035da785f"}]`, iotago.EncodeHex(im.SmrTokenId), iotago.EncodeHex(im.SmrTokenId))
)
