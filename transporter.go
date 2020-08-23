package rsocket_transport_quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/rsocket/rsocket-go/core/transport"
)

const tlsProtoQUIC = "quic-rsocket"

type ServerBuilder struct {
	addr string
	c    *tls.Config
}

type ClientBuilder struct {
	addr string
	c    *tls.Config
}

func (q *ServerBuilder) SetAddr(addr string) *ServerBuilder {
	q.addr = addr
	return q
}

func (q *ServerBuilder) SetTLSConfig(c *tls.Config) *ServerBuilder {
	q.c = c
	return q
}

func (q *ServerBuilder) SetHostAndPort(host string, port int) *ServerBuilder {
	q.addr = fmt.Sprintf("%s:%d", host, port)
	return q
}

func (q *ServerBuilder) Build() transport.ServerTransporter {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return newServerTransport(q.addr, q.c), nil
	}
}

func (q *ClientBuilder) SetAddr(addr string) *ClientBuilder {
	q.addr = addr
	return q
}

func (q *ClientBuilder) SetTLSConfig(c *tls.Config) *ClientBuilder {
	q.c = c
	return q
}

func (q *ClientBuilder) SetHostAndPort(host string, port int) *ClientBuilder {
	q.addr = fmt.Sprintf("%s:%d", host, port)
	return q
}

func (q *ClientBuilder) Build() transport.ClientTransporter {
	return func(ctx context.Context) (*transport.Transport, error) {
		return newClientTransport(q.addr, q.c)
	}
}

func Server() *ServerBuilder {
	return &ServerBuilder{addr: ":7878", c: generateTLSConfig()}
}

func Client() *ClientBuilder {
	c := &tls.Config{
		InsecureSkipVerify: true,
	}
	c.NextProtos = append(c.NextProtos, tlsProtoQUIC)
	return &ClientBuilder{addr: "127.0.0.1:7878", c: c}
}

func generateTLSConfig() (tlsConf *tls.Config) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	tlsConf = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
	tlsConf.NextProtos = append(tlsConf.NextProtos, tlsProtoQUIC)
	return tlsConf
}
