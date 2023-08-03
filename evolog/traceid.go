package evolog

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	math_rand "math/rand"
	"net"
	"sync/atomic"
	"time"
)

func NewTraceId() string {
	now := time.Now()
	idx := atomic.AddInt32(&traceIndex, 1)
	var result [16 + 8]byte

	day := byte(now.Day())
	result[0] = (day / 10 << 4) + (day % 10)
	copy(result[1:7], traceIdSeed)
	// 3 bytes time
	binary.LittleEndian.PutUint32(result[7:11], uint32(now.Unix()))
	// 3 bytes index
	binary.LittleEndian.PutUint32(result[10:14], uint32(idx))
	// 3 bytes random number
	binary.LittleEndian.PutUint32(result[13:], math_rand.Uint32())
	return hex.EncodeToString(result[0:16])
}

var traceIdSeed []byte
var traceIndex int32
var reserveConn net.Conn

func initTraceIdSeed() {
	var ipBytes []byte
	var portBytes []byte
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		conn.Write([]byte("hello"))
		localAddr, _ := conn.LocalAddr().(*net.UDPAddr)
		if localAddr != nil && len(localAddr.IP) >= 4 {
			// reserve the port until process exit.
			reserveConn = conn
			ipBytes = localAddr.IP[0:4]
			portBytes = make([]byte, 2)
			binary.LittleEndian.PutUint16(portBytes, uint16(localAddr.Port))
		}
	}
	if ipBytes == nil || portBytes == nil {
		ipBytes = make([]byte, 4)
		portBytes = make([]byte, 2)
		rand.Read(ipBytes)
		rand.Read(portBytes)
	}
	traceIdSeed = nil
	traceIdSeed = append(traceIdSeed, ipBytes...)
	traceIdSeed = append(traceIdSeed, portBytes...)
}
