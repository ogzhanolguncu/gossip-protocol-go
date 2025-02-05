package node

import "encoding/binary"

const (
	MessageTypeVersionCheck  = 0x01
	MessageTypeCounterUpdate = 0x02
)

type Message struct {
	Type    byte   // 1 byte for message type (0x01 for version check and 0x02 for counter update)
	Version uint32 // 4 bytes for version number
	Counter uint64 // 8 bytes for the actual counter value
}

// binary.BigEndian ensures the bytes are in network order
func Encode(version uint32, counter uint64) []byte {
	buffer := make([]byte, 13)

	buffer[0] = MessageTypeCounterUpdate
	binary.BigEndian.PutUint32(buffer[1:5], version)
	binary.BigEndian.PutUint64(buffer[5:], counter)

	return buffer
}

func Decode(msg []byte) (version uint32, counter uint64) {
	return binary.BigEndian.Uint32(msg[1:5]),
		binary.BigEndian.Uint64(msg[5:])
}

func EncodeVersionCheck(version uint32) []byte {
	buffer := make([]byte, 5)
	buffer[0] = MessageTypeVersionCheck
	binary.BigEndian.PutUint32(buffer[1:5], version)
	return buffer
}

func DecodeVersionCheck(msg []byte) (version uint32) {
	return binary.BigEndian.Uint32(msg[1:5])
}
