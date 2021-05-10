package shadowstream

var connBufSize uint = 8 * 1024
var packetConnBufSize uint = 4 * 1024

func SetConnBufferSize(size uint) {
	connBufSize = size
}

func SetPacketConnBufSize(size uint) {
	packetConnBufSize = size
}
