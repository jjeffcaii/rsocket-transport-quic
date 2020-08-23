package rsocket_transport_quic_test

import (
	"context"
	"log"
	"testing"

	rsocket_transport_quic "github.com/jjeffcaii/rsocket-transport-quic"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
)

var fakeRequest = payload.NewString("fake data", "fake metadata")

func TestQUICServerTransport_Accept(t *testing.T) {
	started := make(chan struct{})

	go func() {
		err := rsocket.Receive().
			OnStart(func() {
				close(started)
			}).
			Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (responder rsocket.RSocket, err error) {
				responder = rsocket.NewAbstractSocket(
					rsocket.RequestResponse(func(request payload.Payload) mono.Mono {
						return mono.Just(request)
					}),
				)
				return
			}).
			Transport(rsocket_transport_quic.Server().SetAddr(":443").Build()).
			Serve(context.Background())
		log.Fatalln(err)
	}()

	client, err := rsocket.Connect().
		Transport(rsocket_transport_quic.Client().SetAddr("127.0.0.1:443").Build()).
		Start(context.Background())
	assert.NoError(t, err, "connect failed")
	defer client.Close()

	response, err := client.RequestResponse(fakeRequest).Block(context.Background())
	assert.NoError(t, err, "request failed")
	assert.True(t, payload.Equal(fakeRequest, response))
}
