package im

import (
	"bytes"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/hooks/debug"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/rs/zerolog"
)

type MQTTOpts struct {
	WebsocketBindAddress string
	qos                  byte
}
type MQTTServer struct {
	server *mqtt.Server
	opts   *MQTTOpts
}
type ExampleHook struct {
	mqtt.HookBase
}

func (h *ExampleHook) ID() string {
	return "events-example"
}

func (h *ExampleHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
		mqtt.OnPublish,
	}, []byte{b})
}

func (h *ExampleHook) Init(config any) error {
	h.Log.Info().Msg("initialised")
	return nil
}

func (h *ExampleHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.Log.Info().Str("client", cl.ID).Msgf("client connected")
	return nil
}

func (h *ExampleHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	h.Log.Info().Str("client", cl.ID).Bool("expire", expire).Err(err).Msg("client disconnected")
}

func (h *ExampleHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msgf("subscribed qos=%v", reasonCodes)
}

func (h *ExampleHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msg("unsubscribed")
}

func (h *ExampleHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Msg("received from client")

	pkx := pk
	if string(pk.Payload) == "hello" {
		pkx.Payload = []byte("hello world")
		h.Log.Info().Str("client", cl.ID).Str("payload", string(pkx.Payload)).Msg("received modified packet from client")
	}

	return pkx, nil
}

func (h *ExampleHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Msg("published to client")
}
func NewMQTTServer(opts *MQTTOpts) (*MQTTServer, error) {
	server := mqtt.New(nil)
	server.Options.Capabilities.Compatibilities.ObscureNotAuthorized = true
	server.Options.Capabilities.Compatibilities.PassiveClientDisconnect = true
	server.Options.Capabilities.Compatibilities.NoInheritedPropertiesOnAck = true
	l := server.Log.Level(zerolog.DebugLevel)
	server.Log = &l
	_ = server.AddHook(new(debug.Hook), &debug.Options{
		ShowPacketData: true,
	})
	_ = server.AddHook(new(auth.AllowHook), nil)
	_ = server.AddHook(new(ExampleHook), map[string]any{})
	ws := listeners.NewWebsocket("ws1", "localhost:1888", nil)
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

// serve
func (s *MQTTServer) Serve() error {
	return s.server.Serve()
}

// stop the server, server.Close()
func (s *MQTTServer) Close() {
	s.server.Close()
}
