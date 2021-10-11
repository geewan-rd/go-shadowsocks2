package shadowsocks2

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fregie/mpx"
	"github.com/shadowsocks/go-shadowsocks2/freconn"
	"github.com/shadowsocks/go-shadowsocks2/websocket"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type ssConfig struct {
	Verbose      bool
	UDPTimeout   time.Duration
	UDPBufSize   int
	WSTimeout    time.Duration
	MaxConnCount int
}

var config = ssConfig{
	Verbose:    true,
	UDPTimeout: 10 * time.Second,
	UDPBufSize: 64 * 1024,
	WSTimeout:  10 * time.Second,
}

var (
	logWriter    = os.Stdout
	logger       = log.New(logWriter, "[shadowsocks]", log.LstdFlags)
	stat         = freconn.NewStat()
	client       *Client
	localIP      string
	tcpConnecter = &TCPConnecter{}
)

var ERR_MPXFirstConnectionFail = errors.New("Connect Failed")

func logf(f string, v ...interface{}) {
	if config.Verbose {
		logger.Printf(f, v...)
	}
}

// SetlogOut 设置websocket timeout，单位 ms, 默认 10s
func SetWSTimeout(timeout int) {
	if timeout > 0 {
		config.WSTimeout = time.Duration(timeout) * time.Millisecond
	}
}

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

// SetMaxConnCount 设置最大并发连接数
func SetMaxConnCount(maxConnCount int) {
	if maxConnCount < 0 {
		maxConnCount = 0
	}
	config.MaxConnCount = maxConnCount
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
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Print(err)
		return err
	}
	udpLocal(p[0], udpAddr, p[1], &UDPConnecter{}, ciph.PacketConn)
	return nil
}

func StartUDPWSTunnel(server string, serverPort int, method, URL, username string, password string, tunnel string) error {
	config.Verbose = true
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        URL,
		Username:   username,
		Stat:       stat,
	}
	var key []byte
	ciph, err := core.PickCipher(method, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	socks.UDPEnabled = true
	p := strings.Split(tunnel, "=")
	wsAddr := websocket.WSAddr{
		URL: url.URL{Scheme: "ws", Host: connecter.ServerAddr, Path: connecter.URL},
	}
	udpLocal(p[0], &wsAddr, p[1], connecter, ciph.PacketConn)
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
	if server == "" || password == "" {
		return errors.New("server, password can not be empty")
	}
	if serverPort <= 0 || serverPort > 65535 {
		return errors.New("server port must be between 0 and 65535")
	}
	if localPort <= 0 || localPort > 65535 {
		return errors.New("local port must be between 0 and 65535")
	}

	var err error
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method

	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}

	socks.UDPEnabled = true
	localAddr := fmt.Sprintf("%s:%d", "0.0.0.0", localPort)
	client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
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
	udpAddr, err := net.ResolveUDPAddr("UDP", addr)
	if err != nil {
		return err
	}
	err = client.udpSocksLocal(localAddr, udpAddr, &UDPConnecter{}, upgradePC)
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
		err = client.Stop()
	}
	return
}

// StartWebsocket 启动SSW
func StartWebsocket(server, URL, username string, serverPort int, method string, password string, localPort int, verbose bool) error {
	config.Verbose = verbose
	var key []byte
	if server == "" || URL == "" || username == "" || password == "" {
		return errors.New("server, URL, username, password can not be empty")
	}
	if serverPort <= 0 || serverPort > 65535 {
		return errors.New("server port must be between 0 and 65535")
	}
	if localPort <= 0 || localPort > 65535 {
		return errors.New("local port must be between 0 and 65535")
	}

	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	var err error
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	socks.UDPEnabled = true
	localAddr := fmt.Sprintf("%s:%d", "0.0.0.0", localPort)
	client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
	stat.Reset()
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        URL,
		Username:   username,
		Stat:       stat,
	}
	connecter.SetTimeout(config.WSTimeout)
	logf("Start shadowsocks on websocket, server: %s", connecter.ServerAddr)
	err = client.StartsocksConnLocal(localAddr, connecter, ciph.StreamConn)
	if err != nil {
		return err
	}
	upgradePC := func(pc net.PacketConn) net.PacketConn {
		spc := ciph.PacketConn(pc)
		newPC := freconn.UpgradePacketConn(spc)
		newPC.EnableStat(stat)
		return newPC
	}
	wsAddr := websocket.WSAddr{
		URL: url.URL{Scheme: "ws", Host: connecter.ServerAddr, Path: connecter.URL},
	}
	err = client.udpSocksLocal(localAddr, &wsAddr, connecter, upgradePC)
	if err != nil {
		return err
	}
	return nil
}

// StopWebsocket 停止SSW
func StopWebsocket() error {
	stat.Reset()
	var err error
	if client != nil {
		logf("Stop shadowsocks on websocket")
		err = client.Stop()
		if err != nil {
			logf("Stop shadowsocks on websocket connction failed:%s", err)
		}
	}
	return err
}

var mc *mpxConnecter

func StartWebsocketMpx(server, URL, username string, serverPort int, method string, password string, localPort int, connCount int, verbose bool) (err error) {
	config.Verbose = verbose
	if !verbose {
		mpx.Verbose(false)
	}
	if server == "" || URL == "" || username == "" || password == "" {
		return errors.New("server, URL, username, password can not be empty")
	}
	if serverPort <= 0 || serverPort > 65535 {
		return errors.New("server port must be between 0 and 65535")
	}
	if localPort <= 0 || localPort > 65535 {
		return errors.New("local port must be between 0 and 65535")
	}

	if connCount <= 0 {
		connCount = 2
	}
	var key []byte
	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		logf(err.Error())
		return
	}
	socks.UDPEnabled = true
	localAddr := fmt.Sprintf("%s:%d", "0.0.0.0", localPort)
	client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
	stat.Reset()
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        URL,
		Username:   username,
		Stat:       stat,
	}
	connecter.SetTimeout(config.WSTimeout)
	mc, err = NewMpxConnecter(connecter, connCount)
	if err != nil {
		logf("Mpx first connect failed: %s", err)
		err = ERR_MPXFirstConnectionFail
		// mc.Close()
		// return err
	}
	logf("Start shadowsocks on websocket mpx, server: %s", connecter.ServerAddr)
	err = client.StartsocksConnLocal(localAddr, mc, ciph.StreamConn)
	if err != nil {
		return
	}
	upgradePC := func(pc net.PacketConn) net.PacketConn {
		spc := ciph.PacketConn(pc)
		newPC := freconn.UpgradePacketConn(spc)
		newPC.EnableStat(stat)
		return newPC
	}
	wsAddr := websocket.WSAddr{
		URL: url.URL{Scheme: "ws", Host: connecter.ServerAddr, Path: connecter.URL},
	}
	err = client.udpSocksLocal(localAddr, &wsAddr, connecter, upgradePC)
	if err != nil {
		return
	}
	return
}

func StopWebsocketMpx() error {
	stat.Reset()
	if client != nil {
		logf("Stop shadowsocks on websocket mpx")
		client.Stop()
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
