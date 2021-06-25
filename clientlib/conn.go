package shadowsocks2

import (
	"errors"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Jeffail/tunny"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type Client struct {
	TCPListener  net.Listener
	TCPRuning    bool
	MaxConnCount int
	connCount    int
	mutex        sync.Mutex
	closeTCP     bool
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
		var connArray = make([]net.Conn, c.MaxConnCount)
		addConn := func(conn net.Conn, index int) {
			if c.MaxConnCount > 0 {
				c.mutex.Lock()
				con := connArray[index]
				if con != nil {
					con.Close()
				}
				connArray[index] = conn
				c.mutex.Unlock()
			}
		}
		var pool *tunny.Pool
		if c.MaxConnCount > 0 {
			pool = tunny.NewFunc(c.MaxConnCount+2, func(p interface{}) interface{} {
				f := p.(func())
				f()
				return true
			})
		}

		defer func() {
			c.TCPRuning = false
			c.mutex.Lock()
			for _, con := range connArray {
				con.Close()
			}
			c.mutex.Unlock()
		}()
		var connCount = 0
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
			var currnetIndex = 0
			if c.MaxConnCount > 0 {
				currnetIndex = connCount % c.MaxConnCount
			}
			addConn(lc, currnetIndex)
			connCount += 1
			lc.(*net.TCPConn).SetKeepAlive(false)
			if pool == nil {
				go c.handleConn(lc, connecter, shadow)
			} else {
				go pool.Process(func() {
					c.handleConn(lc, connecter, shadow)
				})
			}

		}
	}()

	return nil
}

func (c *Client) handleConn(lc net.Conn, connecter Connecter, shadow shadowUpgrade) {
	// defer func() {
	// 	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
	// 		runtime.GC()
	// 		debug.FreeOSMemory()
	// 	}
	// }()
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
	// log.Printf("web addr:%s", rc.LocalAddr())
	// log.Printf("accept addr:%s", lc.RemoteAddr())
	if err != nil {
		logf("Connect to %s failed: %s", connecter.ServerHost(), err)
		return
	}
	defer rc.Close()

	var remoteConn net.Conn
	remoteConn = shadow(rc)
	if _, err = remoteConn.Write(tgt); err != nil {
		logf("failed to send target address: %v", err)
		return
	}

	logf("proxy %s <-> %s <-> %s", lc.RemoteAddr(), connecter.ServerHost(), tgt)
	_, _, err = relay(remoteConn, lc)
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

	// interval := 2 * time.Second
	// heartTimer := time.NewTimer(interval)

	copyFunc := func(left net.Conn, right net.Conn) {
		buf := make([]byte, 512*2)
		close := func() {
			left.Close()
			right.Close()
			debug.FreeOSMemory()
		}
		for {
			n, err := left.Read(buf)
			if err != nil {
				close()
				return
			}
			n, err = right.Write(buf[:n])
			if err != nil {
				close()
				return
			}
		}
	}
	go copyFunc(left, right)
	copyFunc(right, left)

	return 0, 0, errors.New("closed")
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

type connLastSeen struct {
	net.Conn
	lastSeen time.Time
}

var readCount = 1
var writeCount = 1

func (c *connLastSeen) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	readCount += 1
	logf("readCount:%d,length:%d", readCount, len(b))
	if err != nil {
		c.lastSeen = time.Now()
	}
	return
}

func (c *connLastSeen) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	writeCount += 1
	logf("readCount:%d,length:%d", writeCount, len(p))
	if err != nil {
		c.lastSeen = time.Now()
	}
	return
}
