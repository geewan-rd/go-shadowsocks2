package shadowaead

// payloadSizeMask is the maximum size of payload in bytes.
var payloadSizeMask = 0x3FFF // 16*1024 - 1
var packetConnBufSize uint = 4 * 1024

func SetPayloadSize(size uint) {
	payloadSizeMask = int(size)
}

func SetPacketConnBufSize(size uint) {
	packetConnBufSize = size
}
