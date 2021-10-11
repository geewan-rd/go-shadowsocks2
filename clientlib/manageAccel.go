package shadowsocks2

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"runtime/debug"
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

type SSClient struct {
	client *Client
	config ssConfig
}

var (
	logWriter = os.Stdout
	stat      = freconn.NewStat()

	localIP string = "0.0.0.0"
)

var ERR_MPXFirstConnectionFail = errors.New("Connect Failed")

func (c *SSClient) init() {
	c.config = ssConfig{
		Verbose:    true,
		UDPTimeout: 10 * time.Second,
		UDPBufSize: 64 * 1024,
		WSTimeout:  10 * time.Second,
	}
}

// SetlogOut 设置websocket timeout，单位 ms, 默认 10s
func (c *SSClient) SetWSTimeout(timeout int) {
	if timeout > 0 {
		c.config.WSTimeout = time.Duration(timeout) * time.Millisecond
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
	websocket.Logger = logger
	return nil
}

// delete log 删除日志
func DeleteLog(path string) error {
	err := os.Remove(path)
	return err
}

// FinishLog 停止记录日志，关闭对应文件
func FinishLog() error {
	if logWriter != nil {
		defer func() {
			logWriter = nil
			logger = nil
		}()
		return logWriter.Close()
	}
	return errors.New("log writter is nil")
}

// SetMaxConnCount 设置最大并发连接数
func (c *SSClient) SetMaxConnCount(maxConnCount int) {
	if maxConnCount < 0 {
		maxConnCount = 0
	}
	c.config.MaxConnCount = maxConnCount
}

func SetLocalIP(ip string) error {

	localIP = ip
	return nil
}
func SetSSWLocalIP(ip string) {
	localIP = ip
}

func (c *SSClient) StartUDPTunnel(server string, serverPort int, method string, password string, tunnel string) error {
	c.config.Verbose = true
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

func (c *SSClient) StartUDPWSTunnel(server string, serverPort int, method, URL, username string, password string, tunnel string) error {
	c.config.Verbose = true
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

func (c *SSClient) StartTCPtunnel(server string, serverPort int, method string, password string, tunnel string) error {
	c.config.Verbose = true
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
func (c *SSClient) StartTCPUDP(server string, serverPort int, method string, password string, localPort int, verbose bool) error {
	Verbose = verbose
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
	localAddr := fmt.Sprintf("%s:%d", localIP, localPort)
	config := c.config
	c.client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
	tcpConnecter := &TCPConnecter{}
	TCPAddr, err := net.ResolveTCPAddr("tcp4", localIP+":0")
	if err != nil {
		logf("local addr failed: %s", err)
		return err
	}
	tcpConnecter.localTCPAddr = TCPAddr
	tcpConnecter.ServerAddr = addr
	stat.Reset()
	tcpConnecter.Stat = stat

	logf("Start shadowsocks on TCP, server: %s local:%s", tcpConnecter.ServerAddr, tcpConnecter.localTCPAddr)
	err = c.client.StartsocksConnLocal(localAddr, tcpConnecter, ciph.StreamConn)
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
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	err = c.client.udpSocksLocal(localAddr, udpAddr, &UDPConnecter{}, upgradePC)
	if err != nil {
		return err
	}
	return nil
}

// StopTCPUDP 停止SS
func (c *SSClient) StopTCPUDP() (err error) {
	stat.Reset()
	if c.client != nil {
		logf("Stop shadowsocks on TCP")
		err = c.client.Stop()
	}
	return
}

// StartWebsocket 启动SSW
func (c *SSClient) StartWebsocket(server, URL, username string, serverPort int, method string, password string, localPort int, verbose bool) error {
	Verbose = verbose
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
	debug.SetGCPercent(10)

	addr := fmt.Sprintf("%s:%d", server, serverPort)
	cipher := method
	var err error
	ciph, err := core.PickCipher(cipher, key, password)
	if err != nil {
		log.Print(err)
		return err
	}
	socks.UDPEnabled = true
	localAddr := fmt.Sprintf("%s:%d", localIP, localPort)
	config := c.config
	c.client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
	stat.Reset()
	connecter := &WSConnecter{
		ServerAddr: addr,
		URL:        URL,
		Username:   username,
		Stat:       stat,
	}
	connecter.SetTimeout(config.WSTimeout)
	logf("Start shadowsocks on websocket, server: %s", connecter.ServerAddr)
	err = c.client.StartsocksConnLocal(localAddr, connecter, ciph.StreamConn)
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
	err = c.client.udpSocksLocal(localAddr, &wsAddr, connecter, upgradePC)
	if err != nil {
		return err
	}

	return nil
}

// StartWebsocket 启动SSW
type sswconf struct {
	Server, Url, Username            string
	ServerPort, LocalPort, PprofPort int
	Method, Password                 string
	Verbose                          bool
}

func (c *SSClient) StartWebsocketWithjson(data []byte) error {

	var conf sswconf
	if err := json.Unmarshal(data, &conf); err != nil {
		return err
	}

	// if conf.PprofPort > 0 {
	// 	go func() {
	// 		for {
	// 			// http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", conf.PprofPort), nil)
	// 		}
	// 	}()

	// }

	return c.StartWebsocket(conf.Server, conf.Url, conf.Username, conf.ServerPort, conf.Method, conf.Password, conf.LocalPort, conf.Verbose)
}

// StopWebsocket 停止SSW
func (c *SSClient) StopWebsocket() error {
	stat.Reset()
	var err error
	if c.client != nil {
		logf("Stop shadowsocks on websocket")
		err = c.client.Stop()
		if err != nil {
			logf("Stop shadowsocks on websocket connction failed:%s", err)
		}
	}
	return err
}

var mc *mpxConnecter

func (c *SSClient) StartWebsocketMpx(server, URL, username string, serverPort int, method string, password string, localPort int, connCount int, verbose bool) (err error) {
	Verbose = verbose
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
	localAddr := fmt.Sprintf("%s:%d", localIP, localPort)
	config := c.config
	c.client = NewClient(config.MaxConnCount, config.UDPBufSize, config.UDPTimeout)
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
	err = c.client.StartsocksConnLocal(localAddr, mc, ciph.StreamConn)
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
	err = c.client.udpSocksLocal(localAddr, &wsAddr, connecter, upgradePC)
	if err != nil {
		return
	}
	return
}

func (c *SSClient) StopWebsocketMpx() error {
	stat.Reset()
	if c.client != nil {
		logf("Stop shadowsocks on websocket mpx")
		c.client.Stop()
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
