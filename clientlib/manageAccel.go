package shadowsocks2

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type ssConfig struct {
	Verbose    bool
	UDPTimeout time.Duration
}

var config = ssConfig{
	Verbose: true,
}

var logWriter = os.Stderr
var logger = log.New(logWriter, "[shadowsocks]", log.LstdFlags)

func logf(f string, v ...interface{}) {
	if config.Verbose {
		logger.Printf(f, v...)
	}
}

var client *Client
var localIP string
var tcpConnecter = &TCPConnecter{}

func SetlogOut(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	logWriter = f
	logger.SetOutput(logWriter)
	return nil
}

func FinishLog() error {
	if logWriter != nil {
		return logWriter.Close()
	}
	return errors.New("log writter is nil")
}

func SetLocalIP(ip string) error {
	TCPAddr, err := net.ResolveTCPAddr("tcp4", ip+":0")
	if err != nil {
		logf("local addr failed: %s", err)
		return err
	}
	tcpConnecter.localTCPAddr = TCPAddr
	localIP = ip
	return nil
}

func StartTCPUDP(server string, serverPort int, method string, password string, localPort int, verbose bool) error {
	config.Verbose = verbose
	var key []byte

	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	var err error

	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}

	socks.UDPEnabled = true
	localAddr := fmt.Sprintf("%s:%d", "127.0.0.1", localPort)
	client = &Client{}
	tcpConnecter.ServerAddr = addr

	logf("Start shadowsocks on TCP, server: %s", tcpConnecter.ServerAddr)
	err = client.StartsocksConnLocal(localAddr, tcpConnecter, ciph.StreamConn)
	if err != nil {
		return err
	}
	logf("Start shadowsocks on UDP, server: %s", tcpConnecter.ServerAddr)
	err = udpSocksLocal(localAddr, addr, ciph.PacketConn)
	if err != nil {
		return err
	}
	return nil
}

func StopTCPUDP() (err error) {
	if client != nil {
		logf("Stop shadowsocks on TCP")
		err = client.StopsocksConnLocal()
	}
	logf("Stop shadowsocks on UDP")
	err = stopUdpSocksLocal()
	return
}

func StartWebsocket(server, url, username string, serverPort int, method string, password string, localPort int, verbose bool) error {
	config.Verbose = verbose
	var key []byte
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	var err error
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	socks.UDPEnabled = false
	localAddr := fmt.Sprintf("%s:%d", "127.0.0.1", localPort)
	client = &Client{}
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        url,
		Username:   username,
	}
	logf("Start shadowsocks on websocket, server: %s", connecter.ServerAddr)
	return client.StartsocksConnLocal(localAddr, connecter, ciph.StreamConn)
}

func StopWebsocket() error {
	if client != nil {
		logf("Stop shadowsocks on websocket")
		return client.StopsocksConnLocal()
	}
	return errors.New("SS client is nil")
}
