package shadowsocks2

import (
	"errors"
	"fmt"
	"net"
	"time"

	"sync"

	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type mode int

const (
	remoteServer mode = iota
	relayClient
	socksClient
)

const udpBufSize = 10 * 1024

func newBuff() []byte {
	buf := make([]byte, udpBufSize)
	return buf
}

// Listen on laddr for UDP packets, encrypt and send to server to reach target.
func udpLocal(laddr string, server net.Addr, target string, connecter PcConnecter, shadow func(net.PacketConn) net.PacketConn) {
	var err error
	tgt := socks.ParseAddr(target)
	if tgt == nil {
		err = fmt.Errorf("invalid target address: %q", target)
		logf("UDP target address error: %v", err)
		return
	}

	c, err := net.ListenPacket("udp", laddr)
	if err != nil {
		logf("UDP local listen error: %v", err)
		return
	}
	defer c.Close()

	nm := newNATmap(10 * time.Second)

	buf := newBuff()
	copy(buf, tgt)

	logf("222UDP tunnel %s <-> %s <-> %s", laddr, server, target)
	for {

		n, raddr, err := c.ReadFrom(buf[len(tgt):])
		logf("udpLocal-ReadFrom:%d", len(buf))
		if err != nil {
			logf("UDP local read error: %v", err)
			continue
		}

		pc := nm.Get(raddr.String())
		if pc == nil {
			pc, err = connecter.DialPacketConn(&net.UDPAddr{})
			if err != nil {
				logf("UDP local listen error: %v", err)
				continue
			}

			pc = shadow(pc)
			nm.Add(raddr, c, pc, relayClient)
		}

		logf("%s <-> %s <-> %s", pc.LocalAddr().String(), server.String(), tgt.String())
		_, err = pc.WriteTo(buf[:len(tgt)+n], server)
		if err != nil {
			logf("UDP local write error: %v", err)
			continue
		}
	}
}

type PcConnecter interface {
	DialPacketConn(localAddr net.Addr) (net.PacketConn, error)
}

var (
	chExit     = make(chan string)
	socksPc    net.PacketConn
	runningUPD = false
)

// Listen on laddr for Socks5 UDP packets, encrypt and send to server to reach target.
func udpSocksLocal(laddr string, server net.Addr, connecter PcConnecter, shadow func(net.PacketConn) net.PacketConn) error {
	if runningUPD {
		stopUdpSocksLocal()
		time.Sleep(500 * time.Microsecond)
	}
	var err error
	socksPc, err = net.ListenPacket("udp", laddr)
	if err != nil {
		loge("UDP local listen error: %v", err)
		return err
	}
	runningUPD = true
	go func() {
		defer socksPc.Close()

		nm := newNATmap(config.UDPTimeout)

		defer func() { runningUPD = false }()
		for {
			select {
			case <-chExit:
				logf("exit ss\n")
				return
			default:
				buf := newBuff()
				n, raddr, err := socksPc.ReadFrom(buf)

				if err != nil {
					loge("UDP local read error: %v", err)
					continue
				}
				pc := nm.Get(raddr.String())
				if pc == nil {
					pc, err = connecter.DialPacketConn(&net.UDPAddr{})
					if err != nil {
						loge("UDP local listen error: %v", err)
						continue
					}

					pc = shadow(pc)
					nm.Add(raddr, socksPc, pc, socksClient)
				}

				_, err = pc.WriteTo(buf[3:n], server)
				// _, err = pc.WriteTo(payload, tgtUDPAddr)
				if err != nil {
					loge("UDP local write error: %v", err)
					continue
				}
			}
		}
	}()
	return nil
}

func stopUdpSocksLocal() error {
	if !runningUPD {
		return errors.New("ss not running")
	}
	logf("stoping ss")
	err := socksPc.Close()
	if err != nil {
		logf("stop ss err: %s", err)
		return errors.New("stop ss err: " + err.Error())
	}
	chExit <- "exit"
	return nil
}

// Listen on addr for encrypted packets and basically do UDP NAT.
func udpRemote(addr string, shadow func(net.PacketConn) net.PacketConn) {
	c, err := net.ListenPacket("udp", addr)
	if err != nil {
		logf("UDP remote listen error: %v", err)
		return
	}
	defer c.Close()
	c = shadow(c)

	nm := newNATmap(config.UDPTimeout)

	logf("listening UDP on %s", addr)
	for {
		buf := newBuff()
		n, raddr, err := c.ReadFrom(buf)
		logf("udpRemote-ReadFrom:%d", len(buf))
		if err != nil {
			logf("UDP remote read error: %v", err)
			continue
		}

		tgtAddr := socks.SplitAddr(buf[:n])
		if tgtAddr == nil {
			logf("failed to split target address from packet: %q", buf[:n])
			continue
		}

		tgtUDPAddr, err := net.ResolveUDPAddr("udp", tgtAddr.String())
		if err != nil {
			logf("failed to resolve target UDP address: %v", err)
			continue
		}

		payload := buf[len(tgtAddr):n]

		pc := nm.Get(raddr.String())
		if pc == nil {
			pc, err = net.ListenPacket("udp", "")
			if err != nil {
				logf("UDP remote listen error: %v", err)
				continue
			}

			nm.Add(raddr, c, pc, remoteServer)
		}

		_, err = pc.WriteTo(payload, tgtUDPAddr) // accept only UDPAddr despite the signature
		if err != nil {
			logf("UDP remote write error: %v", err)
			continue
		}
	}
}

// Packet NAT table
type natmap struct {
	sync.RWMutex
	m       map[string]net.PacketConn
	timeout time.Duration
}

func newNATmap(timeout time.Duration) *natmap {
	m := &natmap{}
	m.m = make(map[string]net.PacketConn)
	m.timeout = timeout
	return m
}

func (m *natmap) Get(key string) net.PacketConn {
	m.RLock()
	defer m.RUnlock()
	return m.m[key]
}

func (m *natmap) Set(key string, pc net.PacketConn) {
	m.Lock()
	defer m.Unlock()

	m.m[key] = pc
}

func (m *natmap) Del(key string) net.PacketConn {
	m.Lock()
	defer m.Unlock()

	pc, ok := m.m[key]
	if ok {
		delete(m.m, key)
		return pc
	}
	return nil
}

func (m *natmap) Add(peer net.Addr, dst, src net.PacketConn, role mode) {
	m.Set(peer.String(), src)

	go func() {
		timedCopy(dst, peer, src, m.timeout, role)
		if pc := m.Del(peer.String()); pc != nil {
			pc.Close()
		}
	}()
}

// copy from src to dst at target with read timeout

func timedCopy(dst net.PacketConn, target net.Addr, src net.PacketConn, timeout time.Duration, role mode) error {

	for {
		buf := newBuff()
		src.SetReadDeadline(time.Now().Add(timeout))
		n, raddr, err := src.ReadFrom(buf)

		if err != nil {
			return err
		}

		switch role {
		case remoteServer: // server -> client: add original packet source
			srcAddr := socks.ParseAddr(raddr.String())
			copy(buf[len(srcAddr):], buf[:n])
			copy(buf, srcAddr)

			_, err = dst.WriteTo(buf[:len(srcAddr)+n], target)
		case relayClient: // client -> user: strip original packet source
			srcAddr := socks.SplitAddr(buf[:n])
			logf("write to %s", target.String())
			_, err = dst.WriteTo(buf[len(srcAddr):n], target)

		case socksClient: // client -> socks5 program: just set RSV and FRAG = 0
			// srcAddr := socks.ParseAddr(raddr.String())
			// copy(buf[len(srcAddr):], buf[:n])
			// copy(buf, srcAddr)
			// _, err = dst.WriteTo(append([]byte{0, 0, 0}, buf[:len(srcAddr)+n]...), target)
			_, err = dst.WriteTo(append([]byte{0, 0, 0}, buf[:n]...), target)

		}

		if err != nil {
			return err
		}
	}
}
