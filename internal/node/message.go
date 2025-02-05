package node

type Message struct {
	Type    byte   // 1 byte for message type (0x01 for version check and 0x02 for counter update)
	Version uint32 // 4 bytes for version number
	Counter uint64 // 8 bytes for the actual counter value
}
