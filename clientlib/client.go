package shadowsocks2

import (
	"context"
	"errors"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type Client struct {
	TCPSocksListener net.Listener
	UDPSocksPC       net.PacketConn
	MaxConnCount     int
	udpTimeout       time.Duration
	udpBufSize       int
	connCount        int
	udpServerAddr    net.Addr
	connResetRLock   sync.RWMutex
	connecter        Connecter
	upgradeConn      shadowUpgradeConn
	pcResetRLock     sync.RWMutex
	pcConnect        PcConnecter
	upgradePc        shadowUpgradePacketConn
	mutex            sync.Mutex
	ConnMap          sync.Map
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewClient(maxConnCount, UDPBufSize int, UDPTimeout time.Duration) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		MaxConnCount: maxConnCount,
		udpTimeout:   UDPTimeout,
		udpBufSize:   UDPBufSize,
		ctx:          ctx,
		cancel:       cancel,
	}
	return c
}

type Connecter interface {
	Connect() (net.Conn, error)
	ServerHost() string
}

type shadowUpgradeConn func(net.Conn) net.Conn
type shadowUpgradePacketConn func(net.PacketConn) net.PacketConn

// Create a SOCKS server listening on addr and proxy to server.
func (c *Client) StartsocksConnLocal(addr string, connecter Connecter, shadow shadowUpgradeConn) error {
	logf("SOCKS proxy %s <-> %s", addr, connecter.ServerHost())
	var err error
	c.TCPSocksListener, err = net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return err
	}
	c.connecter = connecter
	c.upgradeConn = shadow
	go func() {
		connCh := make(chan net.Conn)
		defer close(connCh)
		if c.MaxConnCount > 0 {
			for i := 0; i < c.MaxConnCount; i++ {
				go func() {
					for conn := range connCh {
						lAddr := conn.RemoteAddr().String()
						c.ConnMap.Store(lAddr, conn)
						c.connCount++
						// logf("Conn count++: %d", c.connCount)
						c.handleConn(conn)
						c.connCount--
						// logf("Conn count--: %d", c.connCount)
						c.ConnMap.Delete(lAddr)
					}
				}()
			}
		}
		for {
			lc, err := c.TCPSocksListener.Accept()
			if err != nil {
				logf("failed to accept: %s", err)
				if c.ctx.Err() != nil {
					return
				}
				continue
			}
			lc.(*net.TCPConn).SetKeepAlive(true)
			if c.MaxConnCount == 0 {
				go c.handleConn(lc)
			} else {
				if c.connCount >= c.MaxConnCount {
					go func() {
						c.mutex.Lock()
						defer c.mutex.Unlock()
						var lastSeen *connLastSeen
						var key interface{}
						c.ConnMap.Range(func(k, v interface{}) bool {
							conn := v.(*connLastSeen)
							if lastSeen == nil {
								lastSeen = conn
								key = k
								return true
							}
							if conn.lastSeen.Before(lastSeen.lastSeen) {
								lastSeen = conn
								key = k
							}
							return true
						})
						if lastSeen != nil {
							lastSeen.Close()
							lastSeen.SetDeadline(time.Now())
							c.ConnMap.Delete(key)
						}
					}()
				}
				lastSeenConn := &connLastSeen{
					Conn:     lc,
					lastSeen: time.Now(),
				}
				connCh <- lastSeenConn

			}
		}
	}()

	return nil
}

func (c *Client) handleConn(lc net.Conn) {
	defer func() {
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()
	defer lc.Close()
	if c.connecter == nil || c.upgradeConn == nil {
		return
	}
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
	c.connResetRLock.RLock()
	rc, err := c.connecter.Connect()
	if err != nil {
		logf("Connect to %s failed: %s", c.connecter.ServerHost(), err)
		c.connResetRLock.RUnlock()
		return
	}
	defer rc.Close()

	remoteConn := c.upgradeConn(rc)
	if _, err = remoteConn.Write(tgt); err != nil {
		logf("failed to send target address: %v", err)
		c.connResetRLock.RUnlock()
		return
	}
	logf("proxy %s <-> %s <-> %s", lc.RemoteAddr(), c.connecter.ServerHost(), tgt)
	c.connResetRLock.RUnlock()

	_, _, err = relay(remoteConn, lc)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		logf("relay error: %v", err)
	}
}

type PcConnecter interface {
	DialPacketConn(localAddr net.Addr) (net.PacketConn, error)
}

// Listen on laddr for Socks5 UDP packets, encrypt and send to server to reach target.
func (c *Client) udpSocksLocal(laddr string, server net.Addr, connecter PcConnecter, shadow shadowUpgradePacketConn) error {
	var err error
	c.UDPSocksPC, err = net.ListenPacket("udp", laddr)
	if err != nil {
		logf("UDP local listen error: %v", err)
		return err
	}
	c.pcConnect = connecter
	c.upgradePc = shadow
	c.udpServerAddr = server
	go func() {
		defer c.UDPSocksPC.Close()

		nm := newNATmap(config.UDPTimeout)
		buf := make([]byte, udpBufSize)

		for {
			select {
			case <-c.ctx.Done():
				logf("exit udp\n")
				return
			default:
				n, raddr, err := c.UDPSocksPC.ReadFrom(buf)
				if err != nil {
					logf("UDP local read error: %v", err)
					continue
				}
				pc := nm.Get(raddr.String())
				if pc == nil {
					c.pcResetRLock.RLock()
					pc, err = c.pcConnect.DialPacketConn(&net.UDPAddr{})
					if err != nil {
						logf("UDP local listen error: %v", err)
						c.pcResetRLock.RUnlock()
						continue
					}
					logf("UDP socks tunnel %s <-> %s <-> %s", laddr, c.udpServerAddr, socks.Addr(buf[3:]))
					pc = c.upgradePc(pc)
					nm.Add(raddr, c.UDPSocksPC, pc, socksClient)
				}
				_, err = pc.WriteTo(buf[3:n], c.udpServerAddr)
				c.pcResetRLock.RUnlock()
				if err != nil {
					logf("UDP local write error: %v", err)
					continue
				}
			}
		}
	}()
	return nil
}

func (c *Client) Stop() error {
	logf("stopping tcp ss")
	c.cancel()
	if c.TCPSocksListener != nil {
		err := c.TCPSocksListener.Close()
		if err != nil {
			logf("close tcp listener failed: %s", err)
			return err
		}
	}
	if c.UDPSocksPC != nil {
		err := c.UDPSocksPC.Close()
		if err != nil {
			logf("stop ss err: %s", err)
			return errors.New("stop ss err: " + err.Error())
		}
	}
	return nil
}

func (c *Client) Reset(connecter Connecter, upgradeConn shadowUpgradeConn, UdpServerAddr net.Addr, pcConnect PcConnecter, upgradePc shadowUpgradePacketConn) {
	c.connResetRLock.Lock()
	c.pcResetRLock.Lock()
	c.connecter = connecter
	c.upgradeConn = upgradeConn
	c.udpServerAddr = UdpServerAddr
	c.pcConnect = pcConnect
	c.upgradePc = upgradePc
	c.connResetRLock.Unlock()
	c.pcResetRLock.Unlock()
}
