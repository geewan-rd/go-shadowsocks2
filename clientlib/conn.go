package shadowsocks2

import (
	"errors"
	"io"
	"net"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type Client struct {
	TCPListener net.Listener
	TCPRuning   bool
	closeTCP    bool
}

type Connecter interface {
	Connect() (net.Conn, error)
	ServerHost() string
}

type shadowUpgrade func(net.Conn) net.Conn

// Create a SOCKS server listening on addr and proxy to server.
func (c *Client) StartsocksConnLocal(addr string, connecter Connecter, shadow shadowUpgrade) error {
	logf("SOCKS proxy %s <-> %s", addr, connecter.ServerHost())
	var err error
	c.TCPListener, err = net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return err
	}
	c.TCPRuning = true
	go func() {
		defer func() { c.TCPRuning = false }()
		for {
			lc, err := c.TCPListener.Accept()
			if err != nil {
				logf("failed to accept: %s", err)
				if c.closeTCP {
					logf("tcp ss stoped")
					break
				}
				continue
			}
			lc.(*net.TCPConn).SetKeepAlive(true)
			go c.handleConn(lc, connecter, shadow)
		}
	}()

	return nil
}

func (c *Client) StopsocksConnLocal() error {
	logf("stopping tcp ss")
	if !c.TCPRuning {
		logf("TCP is not running")
		return errors.New("Not running")
	}
	c.closeTCP = true
	err := c.TCPListener.Close()
	if err != nil {
		logf("close tcp listener failed: %s", err)
		return err
	}
	return nil
}

func (c *Client) handleConn(lc net.Conn, connecter Connecter, shadow shadowUpgrade) {
	defer func() {
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()
	defer lc.Close()
	tgt, err := socks.Handshake(lc)
	if err != nil {
		// UDP: keep the connection until disconnect then free the UDP socket
		if err == socks.InfoUDPAssociate {
			buf := []byte{}
			// block here
			for {
				_, err := lc.Read(buf)
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				logf("UDP Associate End.")
				return
			}
		}
		logf("failed to get target address: %v", err)
		return
	}
	rc, err := connecter.Connect()
	if err != nil {
		logf("Connect to %s failed: %s", connecter.ServerHost(), err)
		return
	}
	defer rc.Close()
	rc = shadow(rc)
	if _, err = rc.Write(tgt); err != nil {
		logf("failed to send target address: %v", err)
		return
	}

	logf("proxy %s <-> %s <-> %s", lc.RemoteAddr(), connecter.ServerHost(), tgt)
	_, _, err = relay(rc, lc)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		logf("relay error: %v", err)
	}
}

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
func relay(left, right net.Conn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
		ch <- res{n, err}
	}()

	n, err := io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}
