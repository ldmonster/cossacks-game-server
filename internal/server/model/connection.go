package model

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"
)

var connSeq uint64

type Connection struct {
	ID      uint64
	NetConn net.Conn
	IP      string
	Port    int
	IntIP   uint32
	Ctime   time.Time
	Data    map[string]any
	Closed  bool
}

func NewConnection(conn net.Conn) *Connection {
	host, portStr, _ := net.SplitHostPort(conn.RemoteAddr().String())
	ip := net.ParseIP(host).To4()
	intIP := uint32(0)
	if ip != nil {
		intIP = binary.LittleEndian.Uint32(ip)
	}
	return &Connection{
		ID:      atomic.AddUint64(&connSeq, 1),
		NetConn: conn,
		IP:      host,
		Port:    atoi(portStr),
		IntIP:   intIP,
		Ctime:   time.Now(),
		Data:    map[string]any{},
	}
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return n
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}
