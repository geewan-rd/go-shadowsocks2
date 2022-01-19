package freconn

import (
	"time"
)

type Stat struct {
	Rx     uint64
	Tx     uint64
	in1    statStatus
	in10   statStatus
	isStop bool
}

type statStatus struct {
	t           time.Time
	rx          uint64
	tx          uint64
	bandwidthRx uint64
	bandwidthTx uint64
}

func (s *statStatus) Reset() {
	s.t = time.Now()
	s.rx = 0
	s.tx = 0
	s.bandwidthRx = 0
	s.bandwidthTx = 0
}

func NewStat() *Stat {
	s := &Stat{
		Rx: 0,
		Tx: 0,
		in1: statStatus{
			t:           time.Now(),
			rx:          0,
			tx:          0,
			bandwidthRx: 0,
			bandwidthTx: 0,
		},
		in10: statStatus{
			t:           time.Now(),
			rx:          0,
			tx:          0,
			bandwidthRx: 0,
			bandwidthTx: 0,
		},
		isStop: false,
	}
	go s.RunBandwidthIn1()
	go s.RunBandwidthIn10()

	return s
}

func (s *Stat) Stop() {
	s.isStop = true
}

func (s *Stat) RunBandwidthIn1() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		if s.isStop {
			break
		}
		s.in1.bandwidthRx = s.Rx - s.in1.rx
		s.in1.bandwidthTx = s.Tx - s.in1.tx
		s.in1.rx = s.Rx
		s.in1.tx = s.Tx
		s.in1.t = time.Now()

		// log.Printf("[1s]RX: %dbps", s.in1.bandwidthRx)
		// log.Printf("[1s]TX: %dbps", s.in1.bandwidthTx)
	}
}

func (s *Stat) RunBandwidthIn10() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		if s.isStop {
			break
		}
		s.in10.bandwidthRx = (s.Rx - s.in10.rx) / 10
		s.in10.bandwidthTx = (s.Tx - s.in10.tx) / 10
		s.in10.rx = s.Rx
		s.in10.tx = s.Tx
		s.in10.t = time.Now()
		// log.Printf("[10s]RX: %dbps", s.in10.bandwidthRx)
		// log.Printf("[10s]TX: %dbps", s.in10.bandwidthTx)
	}
}

func (s *Stat) AddRx(len uint64) {
	s.Rx += len
}

func (s *Stat) AddTx(len uint64) {
	s.Tx += len
}

func (s *Stat) Bandwidth1() (r, t uint64, lastTime time.Time) {
	r = s.in1.bandwidthRx
	t = s.in1.bandwidthTx
	lastTime = s.in1.t
	return
}

func (s *Stat) Bandwidth10() (r, t uint64, lastTime time.Time) {
	r = s.in10.bandwidthRx
	t = s.in10.bandwidthTx
	lastTime = s.in10.t
	return
}

func (s *Stat) Reset() {
	s.Rx = 0
	s.Tx = 0
	s.in1.Reset()
	s.in10.Reset()
	s.Stop()
}
