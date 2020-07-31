package shadowsocks2

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/shadowsocks/go-shadowsocks2/freconn"
)

type QuicConnecter struct {
	ServerAddr string
	Stat       *freconn.Stat
	sess       quic.Session
	tlsConf    *tls.Config
}

type QuicConn struct {
	quic.Stream
	quic.Session
}

func NewQuicConnecter(Addr string) (*QuicConnecter, error) {
	q := &QuicConnecter{
		ServerAddr: Addr,
	}
	q.tlsConf = &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic"},
	}
	return q, nil
}

func (q *QuicConnecter) Connect() (net.Conn, error) {
	var err error
	if q.sess == nil {
		q.sess, err = quic.DialAddr(q.ServerAddr, q.tlsConf, nil)
		if err != nil {
			log.Printf("Dail failed:%s", err)
			return nil, err
		}
	}
	stream, err := q.sess.OpenStream()
	if err != nil {
		q.sess, err = quic.DialAddr(q.ServerAddr, q.tlsConf, nil)
		if err != nil {
			log.Printf("Dail failed:%s", err)
			return nil, err
		}
		stream, err = q.sess.OpenStream()
		if err != nil {
			log.Printf("OpenStream failed:%s", err)
			return nil, err
		}
	}
	return QuicConn{Stream: stream, Session: q.sess}, nil
}

func (q *QuicConnecter) ServerHost() string {
	return q.ServerAddr
}
