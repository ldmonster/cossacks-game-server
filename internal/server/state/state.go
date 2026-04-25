package state

import (
	"strings"
	"sync"
	"time"
)

type Player struct {
	ID          uint32
	Nick        string
	ConnectedAt time.Time
	ExitedAt    time.Time
	Account     map[string]any

	TimeTick    uint32
	Nation      uint32
	Theam       uint32
	Color       uint32
	Zombie      bool
	Stat        *PlayerStat
	StatCycle   PlayerStatCycle
	StatHistory map[string][]StatHistoryPoint
	StatSum     map[string]float64
}

type PlayerStat struct {
	Time        uint32
	PC          uint8
	PlayerID    uint32
	Status      uint8
	Scores      uint32
	RealScores  int64
	Population  uint32
	Wood        uint32
	Gold        uint32
	Stone       uint32
	Food        uint32
	Iron        uint32
	Coal        uint32
	Peasants    uint32
	Units       uint32
	Population2 uint32
	Casuality   int64
	ChangeWood  float64
	ChangeStone float64
	ChangeFood  float64
	ChangeGold  float64
	ChangeIron  float64
	ChangeCoal  float64
	ChangePeas  float64
	ChangeUnits float64
	ChangePop2  float64
}

type PlayerStatCycle struct {
	Peasants uint32
	Units    uint32
	Scores   int64
}

type StatHistoryPoint struct {
	Change   float64
	Time     uint32
	Interval uint32
}

type Room struct {
	ID           uint32
	Title        string
	HostID       uint32
	HostAddr     string
	HostAddrInt  uint32
	Ver          uint8
	Level        int
	Password     string
	MaxPlayers   int
	PlayersCount int
	Players      map[uint32]*Player
	PlayersTime  map[uint32]time.Time
	Started      bool
	StartedAt    time.Time
	StartPlayers int
	CtlSum       uint32
	Row          []string
	Ctime        time.Time
	Map          string
	SaveFrom     int
	TimeTick     uint32
	StartedUsers []*Player
}

type Store struct {
	mu sync.RWMutex

	LastPlayerID uint32
	LastRoomID   uint32
	Players      map[uint32]*Player
	RoomsByID    map[uint32]*Room
	RoomsByPID   map[uint32]*Room
	RoomsBySum   map[uint32]*Room
}

func NewStore() *Store {
	return &Store{
		Players:    map[uint32]*Player{},
		RoomsByID:  map[uint32]*Room{},
		RoomsByPID: map[uint32]*Room{},
		RoomsBySum: map[uint32]*Room{},
	}
}

func (s *Store) NextPlayerID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastPlayerID++
	return s.LastPlayerID
}

func (s *Store) UpsertPlayer(p *Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Players[p.ID] = p
}

// NextRoomID returns a new monotonic room id (Perl: ++last_room).
func (s *Store) NextRoomID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastRoomID++
	return s.LastRoomID
}

func RoomControlSum(row []string) uint32 {
	const mod = 0xFFF1
	const chunk = 5552
	s := strings.Join(row, "")
	v1 := uint32(1)
	v2 := uint32(0)
	for i := 0; i < len(s); i += chunk {
		end := i + chunk
		if end > len(s) {
			end = len(s)
		}
		for j := i; j < end; j++ {
			v1 += uint32(s[j])
			v2 += v1
		}
		v1 %= mod
		v2 %= mod
	}
	return (v2 << 16) | v1
}
