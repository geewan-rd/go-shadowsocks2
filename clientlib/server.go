package shadowsocks2

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

func RunServerQuic(addr, method, password string) error {
	ciph, err := core.PickCipher(method, []byte{}, password)
	if err != nil {
		log.Printf("Create SS on port [%s] failed: %s", addr, err)
		return err
	}
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		return err
	}
REACCEPT:
	sess, err := listener.Accept(context.Background())
	if err != nil {
		log.Printf("accept failed:%s", err)
		return err
	}
	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			log.Printf("accept Stream failed:%s", err)
			goto REACCEPT
		}
		conn := QuicConn{Stream: stream, Session: sess}
		go func() {
			defer conn.Close()
			c := ciph.StreamConn(conn)
			handleConn(c)
		}()

	}

}

func handleConn(c net.Conn) {
	tgt, err := socks.ReadAddr(c)
	if err != nil {
		log.Printf("failed to get target address: %v", err)
		return
	}

	var rc net.Conn
	rc, err = net.Dial("tcp4", tgt.String())
	if err != nil || rc == nil {
		log.Printf("failed to connect to target[%s]: %v", tgt.String(), err)
		return
	}
	defer rc.Close()

	log.Printf("proxy %s <-> %s", c.RemoteAddr(), tgt)

	_, _, err = relay(c, rc)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		log.Printf("relay error: %v", err)
	}
}

func generateTLSConfig() *tls.Config {
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
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic"},
	}
}
