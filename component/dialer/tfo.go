package dialer

import (
	"context"
	"github.com/sagernet/tfo-go"
	"io"
	"net"
	"time"
)

type tfoConn struct {
	net.Conn
	closed bool
	dialed chan bool
	cancel context.CancelFunc
	ctx    context.Context
	dialFn func(ctx context.Context, earlyData []byte) (net.Conn, error)
}

func (c *tfoConn) Dial(earlyData []byte) (err error) {
	c.Conn, err = c.dialFn(c.ctx, earlyData)
	if err != nil {
		return
	}
	c.dialed <- true
	return err
}

func (c *tfoConn) Read(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}
	if c.Conn == nil {
		select {
		case <-c.ctx.Done():
			return 0, io.ErrUnexpectedEOF
		case <-c.dialed:
		}
	}
	return c.Conn.Read(b)
}

func (c *tfoConn) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.ErrClosedPipe
	}
	if c.Conn == nil {
		if err := c.Dial(b); err != nil {
			return 0, err
		}
		return len(b), nil
	}

	return c.Conn.Write(b)
}

func (c *tfoConn) Close() error {
	c.closed = true
	c.cancel()
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *tfoConn) LocalAddr() net.Addr {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.LocalAddr()
}

func (c *tfoConn) RemoteAddr() net.Addr {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.RemoteAddr()
}

func (c *tfoConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *tfoConn) SetReadDeadline(t time.Time) error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.SetReadDeadline(t)
}

func (c *tfoConn) SetWriteDeadline(t time.Time) error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.SetWriteDeadline(t)
}

func (c *tfoConn) Upstream() any {
	if c.Conn == nil { // ensure return a nil interface not an interface with nil value
		return nil
	}
	return c.Conn
}

func dialTFO(ctx context.Context, netDialer net.Dialer, network, address string) (net.Conn, error) {
	ctx, cancel := context.WithCancel(ctx)
	dialer := tfo.Dialer{Dialer: netDialer, DisableTFO: false}
	return &tfoConn{
		dialed: make(chan bool, 1),
		cancel: cancel,
		ctx:    ctx,
		dialFn: func(ctx context.Context, earlyData []byte) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address, earlyData)
		},
	}, nil
}
