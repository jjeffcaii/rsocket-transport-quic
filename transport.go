package rsocket_transport_quic

import (
	"context"
	"crypto/tls"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/rsocket/rsocket-go/core/transport"
)

type listenerFactory func(context.Context) (quic.Listener, error)

type quicServerTransport struct {
	mu       sync.Mutex
	f        listenerFactory
	l        quic.Listener
	acceptor transport.ServerTransportAcceptor
	done     chan struct{}
	m        map[*transport.Transport]struct{}
}

func (qu *quicServerTransport) Close() (err error) {
	qu.mu.Lock()
	defer qu.mu.Unlock()

	select {
	case <-qu.done:
		// already closed
		break
	default:
		close(qu.done)
		for k := range qu.m {
			_ = k.Close()
		}
		err = qu.l.Close()
		break
	}
	return
}

func (qu *quicServerTransport) Accept(acceptor transport.ServerTransportAcceptor) {
	qu.acceptor = acceptor
}

func (qu *quicServerTransport) Listen(ctx context.Context, notifier chan<- bool) error {
	listener, err := qu.f(ctx)
	if err != nil {
		notifier <- false
		return err
	}
	qu.l = listener
	notifier <- true

	go func() {
		select {
		case <-ctx.Done():
			_ = qu.Close()
			break
		case <-qu.done:
			// already closed
			break
		}
	}()

	for {
		session, err := listener.Accept(ctx)
		if err != nil {
			return err
		}
		stream, err := session.AcceptStream(ctx)
		if err != nil {
			return err
		}

		tp := transport.NewTransport(newQUICConnection(session, stream))

		if qu.putTransport(tp) {
			go qu.acceptor(ctx, tp, func(tp *transport.Transport) {
				qu.removeTransport(tp)
			})
		} else {
			_ = tp.Close()
		}
	}
}

func (qu *quicServerTransport) removeTransport(tp *transport.Transport) {
	qu.mu.Lock()
	defer qu.mu.Unlock()
	delete(qu.m, tp)
}

func (qu *quicServerTransport) putTransport(tp *transport.Transport) bool {
	qu.mu.Lock()
	defer qu.mu.Unlock()
	select {
	case <-qu.done:
		// already closed
		return false
	default:
		if qu.m == nil {
			return false
		}
		qu.m[tp] = struct{}{}
		return true
	}
}

func newServerTransport(addr string, c *tls.Config) *quicServerTransport {
	f := func(ctx context.Context) (quic.Listener, error) {
		return quic.ListenAddr(addr, c, nil)
	}
	return &quicServerTransport{
		f:    f,
		m:    make(map[*transport.Transport]struct{}),
		done: make(chan struct{}),
	}
}

func newClientTransport(addr string, c *tls.Config) (*transport.Transport, error) {
	session, err := quic.DialAddr(addr, c, nil)
	if err != nil {
		return nil, err
	}
	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}
	return transport.NewTransport(newQUICConnection(session, stream)), nil
}
