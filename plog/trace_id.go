package plog

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	math_rand "math/rand"
	"net"
	"sync/atomic"
	"time"
)

func NewTraceId() string {
	now := time.Now()
	idx := sTraceIdInternal.NextTraceIndex()
	var result [16 + 8]byte

	day := byte(now.Day())
	result[0] = (day / 10 << 4) + (day % 10)
	copy(result[1:7], sTraceIdInternal.traceIdSeed)
	// 3 bytes time
	binary.LittleEndian.PutUint32(result[7:11], uint32(now.Unix()))
	// 3 bytes index
	binary.LittleEndian.PutUint32(result[10:14], idx)
	// 3 bytes random number
	binary.LittleEndian.PutUint32(result[13:], math_rand.Uint32())
	return hex.EncodeToString(result[0:16])
}

func GetLocalIp() string {
	return sTraceIdInternal.sIp
}

type traceIdInternal struct {
	traceIdSeed []byte
	traceIndex  uint32
	reserveConn net.Conn
	sIp         string
}

func newTraceIdInternal() *traceIdInternal {
	tii := &traceIdInternal{}
	tii.init()
	return tii
}

func (tii *traceIdInternal) NextTraceIndex() uint32 {
	return atomic.AddUint32(&tii.traceIndex, 1)
}

func (tii *traceIdInternal) init() {
	tii.initTraceIndex()
	var ipBytes []byte
	var portBytes []byte
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err == nil {
		conn.Write([]byte("hello"))
		localAddr, _ := conn.LocalAddr().(*net.UDPAddr)
		if localAddr != nil && len(localAddr.IP) >= 4 {
			// reserve the port until process exit.
			tii.reserveConn = conn
			ipBytes = localAddr.IP[0:4]
			portBytes = make([]byte, 2)
			tii.sIp = localAddr.IP.String()
			binary.LittleEndian.PutUint16(portBytes, uint16(localAddr.Port))
		}
	}
	if ipBytes == nil || portBytes == nil {
		ipBytes = make([]byte, 4)
		portBytes = make([]byte, 2)
		rand.Read(ipBytes)
		rand.Read(portBytes)
	}
	tii.traceIdSeed = nil
	tii.traceIdSeed = append(tii.traceIdSeed, ipBytes...)
	tii.traceIdSeed = append(tii.traceIdSeed, portBytes...)
}

func (tii *traceIdInternal) initTraceIndex() {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
	if err != nil {
		tii.traceIndex = math_rand.Uint32()
		return
	}
	tii.traceIndex = uint32(n.Int64())
}

var sTraceIdInternal *traceIdInternal
