package im

import (
	"github.com/mochi-co/mqtt/v2"
	"github.com/mochi-co/mqtt/v2/hooks/auth"
	"github.com/mochi-co/mqtt/v2/listeners"
)

type MQTTOpts struct {
	WebsocketBindAddress string
	qos                  byte
}
type MQTTServer struct {
	server *mqtt.Server
	opts   *MQTTOpts
}

func NewMQTTServer(opts *MQTTOpts) (*MQTTServer, error) {
	server := mqtt.New(nil)
	_ = server.AddHook(new(auth.AllowHook), nil)

	ws := listeners.NewWebsocket("ws1", opts.WebsocketBindAddress, nil)
	err := server.AddListener(ws)
	if err != nil {
		return nil, err
	}

	return &MQTTServer{
		server: server,
		opts:   opts,
	}, nil
}

// publish a message to a topic
func (s *MQTTServer) Publish(topic string, payload []byte) error {
	return s.server.Publish(topic, payload, false, s.opts.qos)
}

// stop the server, server.Close()
func (s *MQTTServer) Close() {
	s.server.Close()
}
