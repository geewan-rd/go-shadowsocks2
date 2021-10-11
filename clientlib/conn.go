package shadowsocks2

import (
	"errors"
	"net"
	"runtime/debug"
	"time"
)

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
func relay(left, right net.Conn) (int64, int64, error) {

	// interval := 2 * time.Second
	// heartTimer := time.NewTimer(interval)
	var lastErr error
	copyFunc := func(left net.Conn, right net.Conn) {
		buf := make([]byte, 512*2)
		var currentErr error
		close := func() {
			left.Close()
			right.Close()
			debug.FreeOSMemory()
			if lastErr == nil {
				lastErr = currentErr
			}
		}
		defer close()
		for {
			n, err := left.Read(buf)
			if err != nil {
				currentErr = err
				return
			}
			n, err = right.Write(buf[:n])
			if err != nil {
				currentErr = err
				return
			}
		}
	}
	go copyFunc(left, right)
	copyFunc(right, left)

	if lastErr == nil {
		lastErr = errors.New("closed")
	}

	return 0, 0, lastErr
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
