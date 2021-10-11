package websocket

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSPacketConn struct {
	Username       string
	localAddr      net.Addr
	wsConnMap      sync.Map
	reader         chan *packet
	ctx            context.Context
	cancel         context.CancelFunc
	readCtx        context.Context
	readCtxCancel  context.CancelFunc
	writeCtx       context.Context
	writeCtxCancel context.CancelFunc
	closed         bool
	dailer         *websocket.Dialer
}

type WSAddr struct {
	url.URL
}

func (a *WSAddr) Network() string {
	return a.Scheme
}

type packet struct {
	remoteAddr net.Addr
	buff       []byte
	len        int
}

func NewWSPacketConn(localAddr net.Addr, username string) *WSPacketConn {
	ctx, cancel := context.WithCancel(context.Background())
	readctx, readcancel := context.WithCancel(ctx)
	writectx, writecancel := context.WithCancel(ctx)
	return &WSPacketConn{
		Username:       username,
		localAddr:      localAddr,
		reader:         make(chan *packet),
		ctx:            ctx,
		cancel:         cancel,
		readCtx:        readctx,
		readCtxCancel:  readcancel,
		writeCtx:       writectx,
		writeCtxCancel: writecancel,
		dailer:         &websocket.Dialer{},
	}
}

func (ws *WSPacketConn) SetWSTimeout(timeout time.Duration) {
	if ws.dailer == nil {
		ws.dailer = &websocket.Dialer{
			HandshakeTimeout: timeout,
		}
	} else {
		ws.dailer.HandshakeTimeout = timeout
	}
}

var packetCount = 0

func (ws *WSPacketConn) HandleWSConn(conn *websocket.Conn, remoteAddr net.Addr) error {
	_, exist := ws.wsConnMap.LoadOrStore(remoteAddr.String(), conn)
	if exist {
		conn.Close()
		log.Printf("Conn remote add exist")
		return errors.New("Conn remote add exist")
	}
	ws.localAddr = conn.LocalAddr()
	connCtx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-ws.ctx.Done():
			conn.Close()
		case <-connCtx.Done(): // 防止泄漏
		}
	}()
	go func() {
		defer cancel()
		defer conn.Close()
		defer ws.wsConnMap.Delete(remoteAddr.String())
		for {
			t, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("ReadMessage: %s", err)
				break
			}
			if t != websocket.BinaryMessage {
				log.Printf("Type not Binary")
				continue
			}
			pack := &packet{
				remoteAddr: remoteAddr,
				buff:       p,
				len:        len(p),
			}
			packetCount += 1
			runtime.SetFinalizer(pack, func(p *packet) {

			})
			runtime.GC()
			debug.FreeOSMemory()
			select {
			case ws.reader <- pack:
				// logf("当前长度为：%d", len(ws.reader))
				pack = nil
			case <-ws.ctx.Done():
				break
			}

		}
	}()
	return nil
}

var (
	Logger *log.Logger
)

func logf(f string, v ...interface{}) {
	if Logger == nil {
		return
	}
	Logger.Printf(f, v...)
}

// ReadFrom reads a packet from the connection,
// copying the payload into p. It returns the number of
// bytes copied into p and the return address that
// was on the packet.
// It returns the number of bytes read (0 <= n <= len(p))
// and any error encountered. Callers should always process
// the n > 0 bytes returned before considering the error err.
// ReadFrom can be made to time out and return
// an Error with Timeout() == true after a fixed time limit;
// see SetDeadline and SetReadDeadline.

func (ws *WSPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if ws.closed {
		return 0, nil, io.EOF
	}

	select {
	case <-ws.readCtx.Done():
		return 0, nil, io.EOF
	case packet := <-ws.reader:
		if len(p) < packet.len {
			return 0, packet.remoteAddr, errors.New("Buffer too short")
		}
		// packet.buff = p
		// packet.len = len(p)
		// ws.reader = nil
		copy(p, packet.buff)

		return packet.len, packet.remoteAddr, nil
	}
}

// WriteTo writes a packet with payload p to addr.
// WriteTo can be made to time out and return
// an Error with Timeout() == true after a fixed time limit;
// see SetDeadline and SetWriteDeadline.
// On packet-oriented connections, write timeouts are rare.
func (ws *WSPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if ws.closed {
		return 0, errors.New("conn closed")
	}
	c, ok := ws.wsConnMap.Load(addr.String())
	if !ok {
		if wsAddr, ok := addr.(*WSAddr); ok && ws.Username != "" {
			if ws.dailer == nil {
				ws.dailer = websocket.DefaultDialer
			}
			header := http.Header{
				"Shadowsocks-Username": []string{ws.Username},
				"Shadowsocks-Type":     []string{"packet"},
			}
			wc, _, err := ws.dailer.Dial(addr.String(), header)
			if err != nil {
				return 0, err
			}
			ws.HandleWSConn(wc, wsAddr)
		} else {
			return 0, errors.New("ws conn not found")
		}
	}
	c, ok = ws.wsConnMap.Load(addr.String())
	if !ok {
		return 0, errors.New("ws conn not found")
	}
	conn := c.(*websocket.Conn)
	log.Printf("Write to %s", addr.String())
	err = conn.WriteMessage(websocket.BinaryMessage, p)

	return len(p), err
}

// Close closes the connection.
// Any blocked ReadFrom or WriteTo operations will be unblocked and return errors.
func (ws *WSPacketConn) Close() error {
	if ws.closed {
		return errors.New("already closed")
	}
	ws.cancel()
	ws.wsConnMap.Range(func(k, v interface{}) bool {
		conn := v.(*websocket.Conn)
		conn.Close()
		return true
	})
	return nil
}

// LocalAddr returns the local network address.
func (ws *WSPacketConn) LocalAddr() net.Addr {
	return ws.localAddr
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to ReadFrom or
// WriteTo. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful ReadFrom or WriteTo calls.
//
// A zero value for t means I/O operations will not time out.
func (p *WSPacketConn) SetDeadline(t time.Time) error {
	p.SetReadDeadline(t)
	p.SetWriteDeadline(t)
	return nil
}

// SetReadDeadline sets the deadline for future ReadFrom calls
// and any currently-blocked ReadFrom call.
// A zero value for t means ReadFrom will not time out.
func (p *WSPacketConn) SetReadDeadline(t time.Time) error {
	p.wsConnMap.Range(func(k, v interface{}) bool {
		conn := v.(*websocket.Conn)
		conn.SetReadDeadline(t)
		return true
	})
	now := time.Now()
	if !t.After(now) {
		p.readCtxCancel()
	} else {
		time.AfterFunc(t.Sub(now), func() {
			p.readCtxCancel()
		})
	}
	return nil
}

// SetWriteDeadline sets the deadline for future WriteTo calls
// and any currently-blocked WriteTo call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means WriteTo will not time out.
func (p *WSPacketConn) SetWriteDeadline(t time.Time) error {
	p.wsConnMap.Range(func(k, v interface{}) bool {
		conn := v.(*websocket.Conn)
		conn.SetWriteDeadline(t)
		return true
	})
	now := time.Now()
	if !t.After(now) {
		p.writeCtxCancel()
	} else {
		time.AfterFunc(t.Sub(now), func() {
			p.writeCtxCancel()
		})
	}
	return nil
}
