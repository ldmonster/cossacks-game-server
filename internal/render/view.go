// Copyright 2026 Cossacks Game Server Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Typed view structs for show-template rendering. Templates address
// these structs with PascalCase Go-template syntax (`{{.Room.ID}}`)
// rather than dotted-key string maps.

package render

// View is the canonical top-level data payload passed to show
// templates. Optional sub-views are pointers/slices so callers only
// populate what their target template needs.
type View struct {
	// Generic / per-template scalars.
	ID         string
	Type       string
	LoggedIn   bool
	Password   string
	Nick       string
	Nickname   string
	ChatServer string
	ServerName string
	Header     string
	Text       string
	OkText     string
	Error      string
	ErrorText  string
	Command    string
	Ver        string

	// Layout knobs.
	WindowSize   string
	Height       int
	BottomHeight int

	// Server config used by startup.tmpl.
	ServerTitle  string
	TableTimeout int
	StartAt      string
	Dev          bool
	ShowStarted  bool

	// Connection / network coordinates.
	IP       string
	Port     int
	HoleHost string
	HoleInt  int
	HolePort int
	PlayerID string
	MaxPl    int
	Name     string

	// Sub-views.
	Room        *RoomView
	StartedRoom *StartedRoomView
	GGCup       *GGCupView
	Player      *PlayerView
}

// RoomView is the lobby room shape consumed by `room_info_dgl.tmpl`
// and the `room.*` references inside `started_room_info.tmpl`.
type RoomView struct {
	ID             uint32
	Title          string
	HostID         uint32
	HostNick       string
	HostAddr       string
	HostAddrInt    uint32
	Level          int
	Started        bool
	StartPlayers   int
	PlayersCount   int
	MaxPlayers     int
	Map            string
	AI             bool
	Password       string
	Backto         string
	HasExited      bool
	CtimeFormatted string
	Time           string
	ActivePlayers  []string
	ExitedPlayers  []string
	Players        map[uint32]*RoomPlayerView
}

// RoomPlayerView is the per-player shape used by `room_info_dgl.tmpl`
// to render the host nick from the room's player table.
type RoomPlayerView struct {
	Nick string
}

// StartedRoomView is the in-game (post-launch) room shape consumed by
// `started_room_info.tmpl` and its `statcols.tmpl` partial.
type StartedRoomView struct {
	ID           uint32
	Title        string
	Map          string
	Level        int
	RoomTime     string
	TimeTickSecs uint32
	SmallColW    int
	LargeColW    int
	Page         string
	Res          string
	Players      []*StartedPlayerView
}

// StartedPlayerView is one in-game player slot; matches the field
// names used inside `range $i, $p := .StartedRoom.Players`.
type StartedPlayerView struct {
	Nick   string
	Color  uint32
	Theam  uint32
	Nation uint32
	Zombie bool
	Exited bool
	YOff   int
	Stat   *StartedPlayerStatView
}

// StartedPlayerStatView mirrors the per-tick metrics rendered by
// statcols. Numeric fields stay numeric so template helpers like
// `truthy` and `argFilter` evaluate them naturally.
type StartedPlayerStatView struct {
	RealScores    int64
	Population    uint32
	Population2   uint32
	Casuality     uint32
	Wood          uint32
	Food          uint32
	Stone         uint32
	Gold          uint32
	Iron          uint32
	Coal          uint32
	Peasants      uint32
	Units         uint32
	ChangeWood    string
	ChangeFood    string
	ChangeStone   string
	ChangeGold    string
	ChangeIron    string
	ChangeCoal    string
	ChangePop2    string
	ChangePeas    string
	ChangeUnits   string
	ChangeCoalRes string
}

// GGCupView is the season/event pane consumed by `startup.tmpl` and
// `gg_cup_thanks_dgl.tmpl`.
type GGCupView struct {
	ID              string
	WoInfo          bool
	Started         bool
	PlayersCount    string
	PlayersCountLen int
	PrizeFund       string
	PrizeFundLen    int
	BoxHeight       int
	Supporters      []*GGCupSupporterView
}

// GGCupSupporterView is one row in the GG cup "supporters" table.
type GGCupSupporterView struct {
	Nick   string
	Amount string
	URL    string
}

// PlayerView is the per-user pane consumed by `user_details.tmpl`.
type PlayerView struct {
	ID                   uint32
	Nick                 string
	BoxHeight            int
	LastLabelRef         string
	ConnectedAtFormatted string
	Account              *AccountView
	Room                 *RoomView
}

// AccountView is the auth-account block used by user_details.
type AccountView struct {
	Type     string
	Profile  string
	HasPlace bool
	Place    int
}
