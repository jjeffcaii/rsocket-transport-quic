package rsocket_transport_quic

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
)

type quicConn struct {
	counter *core.TrafficCounter
	session quic.Session
	stream  quic.Stream
	writer  *bufio.Writer
	decoder *transport.LengthBasedFrameDecoder
}

func (p *quicConn) Close() (err error) {
	return p.stream.Close()
}

func (p *quicConn) SetDeadline(deadline time.Time) (err error) {
	err = p.stream.SetReadDeadline(deadline)
	return
}

func (p *quicConn) SetCounter(c *core.TrafficCounter) {
	p.counter = c
}

func (p *quicConn) Read() (f core.Frame, err error) {
	raw, err := p.decoder.Read()
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}

	f, err = framing.FromBytes(raw)
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}

	if p.counter != nil && f.Header().Resumable() {
		p.counter.IncReadBytes(f.Len())
	}

	err = f.Validate()
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	return
}

func (p *quicConn) Write(frame core.WriteableFrame) (err error) {
	size := frame.Len()
	if p.counter != nil && frame.Header().Resumable() {
		p.counter.IncWriteBytes(size)
	}

	b := toUint24Bytes(size)
	_, err = p.writer.Write(b[:])
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	_, err = frame.WriteTo(p.writer)
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	return
}

func (p *quicConn) Flush() (err error) {
	err = p.writer.Flush()
	if err != nil {
		err = errors.Wrap(err, "flush failed")
	}
	return
}

func newQUICConnection(session quic.Session, stream quic.Stream) *quicConn {
	return &quicConn{
		session: session,
		stream:  stream,
		writer:  bufio.NewWriter(stream),
		decoder: transport.NewLengthBasedFrameDecoder(stream),
	}
}

func toUint24Bytes(n int) (v [3]byte) {
	if n > 0xFFFFFF {
		panic(fmt.Sprintf("%d is overflow", n))
	}
	v[0] = byte(n >> 16)
	v[1] = byte(n >> 8)
	v[2] = byte(n)
	return
}
