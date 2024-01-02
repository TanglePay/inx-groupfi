package im

var (
	NFT_ISSUER_LIST = []string{
		"smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp",
		"smr1zpz3430fdn4zmheenyjvughsu44ykjzu5st6hg2rp609eevz6czlye60pe7",
		"smr1zpqndszdf0p9qy04kq7un5clgzptclqeyv70av5q8thjgxcmk2wfy7pspe5",
		"smr1zp46qmajxu0vxc2l73tx0g2j7u579jdzgeplfcnwq6ef0hh3a8zt7wnx9dv",
		"smr1zqa6juwmk7lad4rxsddqeprrz60zksdd0k3xa37lelthzsxsal6vjygkl9e",
		"smr1zptkmnyuxxvk2qv8exqmyxcytmcf74j3t4apc3hfg4h6n9pnfun5q26j6w4",
		"smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf",
		"smr1zpvjkgxkzrhyvxy5nh20j6wm0l7grkf5s6l7r2mrhyspvx9khcaysmam589",
		"smr1zq37t7zsqye0nhagdx8r6df0p4tm67w02afn7fe7sl33kmxsuzj77nyms34",
		"smr1zqc8qjql7rsdj6cm0aktlrp2fpxfuucy99d5fnt0lxh7rm9ufm72jmlnz9e",
		"smr1zrwxcwcpv7hhvajj2e6j8wfypjrqj5jpcc3vxl9hymh7ad0pq25nc462rvv",
	}
	/*
		NFT_ISSUER_LIST = []string{
			"rms1zq3653f3s6yt7f29ztetv7zd4q5gx4d28acklfl4q6arjaghdqujzdqlzs5",
			"rms1zqga89r0vaqea6hljnkcxms3m9ld2zvg0zcd4q00e0ej6ttwztzpy2lnys8",
		}
	*/

	CONFIG_IN_TEXT = `[
		{"groupName":"staff-marketing","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x3070481ff0e0d96b1b7f6cbf8c2a484c9e7304295b44cd6ff9afe1ecbc4efca9"]},
	    {"groupName":"staff-developer","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0xdc6c3b0167af767652567523b9240c86095241c622c37cb726efeeb5e102a93c"]},
	    {"groupName":"alpha-test","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x23e5f8500132f9dfa8698e3d352f0d57bd79cf57533f273e87e31b6cd0e0a5ef"]},
	   	{"groupName":"iceberg-1","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x064d0eaefb86a94eb326ff633c22cdf744decca954bb93b1572b449d324ae717"]},
	   	{"groupName":"iceberg-2","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x451ac5e96cea2ddf399924ce22f0e56a4b485ca417aba1430e9e5ce582d605f2"]},
	   	{"groupName":"iceberg-3","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x4136c04d4bc25011f5b03dc9d31f4082bc7c19233cfeb2803aef241b1bb29c92"]},
	   	{"groupName":"iceberg-4","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x6ba06fb2371ec3615ff45667a152f729e2c9a24643f4e26e06b297def1e9c4bf"]},
	   	{"groupName":"iceberg-5","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x3ba971dbb7bfd6d466835a0c8463169e2b41ad7da26ec7dfcfd77140d0eff4c9"]},
	   	{"groupName":"iceberg-6","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x576dcc9c3199650187c981b21b045ef09f56515d7a1c46e9456fa994334f2740"]},
	   	{"groupName":"iceberg-7","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0xcf0f598ff3ee378b03906af4de48030bc6082831dfcf67730be780a317d98265"]},
	   	{"groupName":"iceberg-8","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x592b20d610ee4618949dd4f969db7ffc81d93486bfe1ab63b9201618b6be3a48"]}]`
	/*
		CONFIG_IN_TEXT = `[
		{"groupName":"test-1","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"rms","collectionIds":["0x23aa45318688bf254512f2b6784da8288355aa3f716fa7f506ba397517683921"]},
		{"groupName":"test-2","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"rms","collectionIds":["0x11d3946f67419eeaff94ed836e11d97ed5098878b0da81efcbf32d2d6e12c412"]}]`
	*/
)
