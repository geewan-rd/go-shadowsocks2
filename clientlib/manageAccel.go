package shadowsocks2

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/freconn"

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

var logWriter = os.Stdout
var logger = log.New(logWriter, "[shadowsocks]", log.LstdFlags)
var stat = freconn.NewStat()

func logf(f string, v ...interface{}) {
	if config.Verbose {
		logger.Printf(f, v...)
	}
}

var client *Client
var localIP string
var tcpConnecter = &TCPConnecter{}

// SetlogOut 设定日志输出到哪个文件
func SetlogOut(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	logWriter = f
	logger.SetOutput(logWriter)
	return nil
}

// FinishLog 停止记录日志，关闭对应文件
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

func StartUDPTunnel(server string, serverPort int, method string, password string, tunnel string) error {
	config.Verbose = true
	var key []byte
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	socks.UDPEnabled = true
	p := strings.Split(tunnel, "=")
	udpLocal(p[0], addr, p[1], ciph.PacketConn)
	return nil
}

func StartTCPtunnel(server string, serverPort int, method string, password string, tunnel string) error {
	config.Verbose = true
	var key []byte
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	p := strings.Split(tunnel, "=")
	tcpTun(p[0], addr, p[1], ciph.StreamConn)
	return nil
}

// StartTCPUDP 启动SS(TCP和UDP)
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
	stat.Reset()
	tcpConnecter.Stat = stat

	logf("Start shadowsocks on TCP, server: %s", tcpConnecter.ServerAddr)
	err = client.StartsocksConnLocal(localAddr, tcpConnecter, ciph.StreamConn)
	if err != nil {
		return err
	}
	logf("Start shadowsocks on UDP, server: %s", tcpConnecter.ServerAddr)
	upgradePC := func(pc net.PacketConn) net.PacketConn {
		spc := ciph.PacketConn(pc)
		newPC := freconn.UpgradePacketConn(spc)
		newPC.EnableStat(stat)
		return newPC
	}
	err = udpSocksLocal(localAddr, addr, upgradePC)
	if err != nil {
		return err
	}
	return nil
}

// StopTCPUDP 停止SS
func StopTCPUDP() (err error) {
	stat.Reset()
	if client != nil {
		logf("Stop shadowsocks on TCP")
		err = client.StopsocksConnLocal()
	}
	logf("Stop shadowsocks on UDP")
	err = stopUdpSocksLocal()
	return
}

// StartWebsocket 启动SSW
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
	stat.Reset()
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        url,
		Username:   username,
		Stat:       stat,
	}
	logf("Start shadowsocks on websocket, server: %s", connecter.ServerAddr)
	return client.StartsocksConnLocal(localAddr, connecter, ciph.StreamConn)
}

// StopWebsocket 停止SSW
func StopWebsocket() error {
	stat.Reset()
	if client != nil {
		logf("Stop shadowsocks on websocket")
		return client.StopsocksConnLocal()
	}
	return errors.New("SS client is nil")
}

var mc *mpxConnecter

func StartWebsocketMpx(server, url, username string, serverPort int, method string, password string, localPort int, verbose bool) error {
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
	stat.Reset()
	dialer := &WSConnecter{
		ServerAddr: addr,
		URL:        url,
		Username:   username,
		Stat:       stat,
	}
	mc = NewMpxConnecter(dialer, 5)
	logf("Start shadowsocks on websocket mpx, server: %s", dialer.ServerAddr)
	return client.StartsocksConnLocal(localAddr, mc, ciph.StreamConn)
}

func StopWebsocketMpx() error {
	stat.Reset()
	if client != nil {
		logf("Stop shadowsocks on websocket mpx")
		return client.StopsocksConnLocal()
	}
	if mc != nil {
		mc.Close()
	}
	return errors.New("SS client is nil")
}

// StatReset 重置（清零）统计数据
// 一般情况不需要手动重置，在启动和停止的时候会自动清零
func StatReset() {
	stat.Reset()
}

// BandwidthInfo 表示一组带宽数据
type BandwidthInfo struct {
	// RX 代表下行（接收）带宽 (bit/s)
	RX int64
	// TX 代表上行（发送）带宽 (bit/s)
	TX int64
	// Timestamp 代表上次计算带宽的时间戳
	Timestamp int64
}

// GetRx 获取下行（接收）带宽 (bit/s)
func (b *BandwidthInfo) GetRx() int64 { return b.RX }

// GetTx 获取上行（发送）带宽 (bit/s)
func (b *BandwidthInfo) GetTx() int64 { return b.TX }

// GetTimestamp 获取上次计算带宽的时间戳
func (b *BandwidthInfo) GetTimestamp() int64 { return b.Timestamp }

// Bandwidth1 返回最近1s的带宽(bit/s)
func Bandwidth1() *BandwidthInfo {
	if stat == nil {
		return &BandwidthInfo{}
	}
	r, t, time := stat.Bandwidth1()
	return &BandwidthInfo{RX: int64(r), TX: int64(t), Timestamp: time.Unix()}
}

// Bandwidth10 返回最近10s的带宽(bit/s)
func Bandwidth10() *BandwidthInfo {
	if stat == nil {
		return &BandwidthInfo{}
	}
	r, t, time := stat.Bandwidth10()
	return &BandwidthInfo{RX: int64(r), TX: int64(t), Timestamp: time.Unix()}
}

// GetRx 返回已经接收的流量总数(bit)
func GetRx() int64 {
	if stat == nil {
		return 0
	}
	return int64(stat.Rx)
}

// GetTx 返回已经发出的流量总数(bit)
func GetTx() int64 {
	if stat == nil {
		return 0
	}
	return int64(stat.Tx)
}
