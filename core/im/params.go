package im

import (
	"github.com/iotaledger/hive.go/core/app"
)

type ParametersIM struct {
	Database struct {
		// Engine defines the used database engine (pebble/rocksdb/mapdb).
		Engine string `default:"rocksdb" usage:"the used database engine (pebble/rocksdb/mapdb)"`
		// Path defines the path to the database folder.
		Path string `default:"database" usage:"the path to the database folder"`
	} `name:"db"`
}

/*
"restAPI": {
    "bindAddress": "localhost:9892",
    "advertiseAddress": "",
    "debugRequestLoggerEnabled": false
  },
*/
// ParametersRestAPI contains the definition of the parameters used by the IM HTTP server.
type ParametersRestAPI struct {
	// BindAddress defines the bind address on which the IM HTTP server listens.
	BindAddress string `default:"localhost:9892" usage:"the bind address on which the IM HTTP server listens"`

	// AdvertiseAddress defines the address of the Participation HTTP server which is advertised to the INX Server (optional).
	AdvertiseAddress string `default:"" usage:"the address of the IOTACAT HTTP server which is advertised to the INX Server (optional)"`
	// DebugRequestLoggerEnabled defines whether the debug logging for requests should be enabled
	DebugRequestLoggerEnabled bool `default:"false" usage:"whether the debug logging for requests should be enabled"`
}

/*
	"mqtt": {
	  "websocket": {
	    "bindAddress": "localhost:1888"
		"advertiseAddress" : 'inx-iotacat:1888'
	  }
	},
*/
type ParametersMQTT struct {
	Websocket struct {
		BindAddress      string `default:"localhost:1888" usage:"the bind address on which the MQTT server listens"`
		AdvertiseAddress string `default:"" usage:"the address of the MQTT server which is advertised to the INX Server (optional)"`
	} `name:"websocket"`
}

var ParamsIM = &ParametersIM{}
var ParamsRestAPI = &ParametersRestAPI{}
var ParamsMQTT = &ParametersMQTT{}
var params = &app.ComponentParams{
	Params: map[string]any{
		"im":      ParamsIM,
		"restAPI": ParamsRestAPI,
		"mqtt":    ParamsMQTT,
	},
	Masked: nil,
}
