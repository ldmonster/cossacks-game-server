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

// User-details dialog body builder. Pure rendering — no I/O, no
// controller state —
// so the handler package no longer hosts dialog assembly.

package render

import (
	"fmt"
	"strings"
	"time"
)

// UserDetailsPlayer is the per-player input the user-details dialog
// needs. The handler package adapts state.Player into this shape so
// the render package stays free of state/store dependencies.
type UserDetailsPlayer struct {
	ID          uint32
	Nick        string
	ConnectedAt time.Time
	// Typed account fields; empty AccountType means not authenticated.
	AccountType    string
	AccountLogin   string
	AccountID      string
	AccountProfile string
}

// UserDetailsRoom is the per-room input the user-details dialog needs.
// Pass nil when the player is not currently a host.
type UserDetailsRoom struct {
	ID    uint32
	Title string
}

// UserDetailsBody emits a dialog shaped like share/cs/user_details.
// When `dev` is true the player ID is rendered
// in the dialog header. `lcnPlaceByID` is consulted only when the
// account type is LCN; pass nil to skip the place lookup.
// `connectedAgo` should already be the human-readable interval string
// callers normally compute via the controller's `roomTimeInterval`
// helper — kept as a parameter to avoid pulling time-formatting deps
// into the render package.
func UserDetailsBody(
	dev bool,
	player UserDetailsPlayer,
	room *UserDetailsRoom,
	lcnPlaceByID map[string]int,
	connectedAgo string,
) string {
	var b strings.Builder

	write := func(s string) {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	write("<NGDLG>")
	write("#exec(LW_lockbox&%LBX)")
	write("#exec(LW_enb&0&%RMLST)")
	write("#ebox[%B](x:210,y:40,w:360,h:160)")
	write("#pan[%MPN](%B[x:0,y:0,w:100%,h:100%],8)")
	write("#font(WF,WF,WF)")
	write("#ctxt[%TIT](%B[x:0,y:6,w:100%,h:30],{},\"Player Info\")")

	if dev {
		write(fmt.Sprintf("#rtxt(%%B[x:280,y:6,w:70,h:30],{},\"#%d\")", player.ID))
	}

	write("#font(WF,WF,WF)")
	write("#txt[%L_NAME](%B[x:20,y:48,w:100,h:100],{},\"Nick\")")
	write("#font(YF,YF,YF)")
	write(fmt.Sprintf("#txt(%%B[x:105,y:48,w:200,h:100],{},\"%s\")", CMLSafe(player.Nick)))
	write("#font(WF,YF,WF)")
	write("#txt[%L_CTIME](%B[x:20,y:74,w:100,h:100],{},\"Connected at\")")
	write("#font(YF,WF,WF)")
	write(fmt.Sprintf("#txt(%%B[x:105,y:74,w:240,h:100],{},\"%s (%s ago)\")",
		player.ConnectedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		connectedAgo,
	))

	y := 100

	if player.AccountType != "" {
		accType := player.AccountType
		profile := player.AccountProfile
		accID := player.AccountID

		write("#font(WF,WF,WF)")
		write(fmt.Sprintf("#txt(%%B[x:20,y:%d,w:100,h:100],{},\"Logon with\")", y))

		if profile != "" {
			write(
				fmt.Sprintf(
					"#btn(%%B[x:105,y:%d,w:120,h:24],{GW|url&%s&from=user_details},\"%s\")",
					y,
					CMLSafe(profile),
					CMLSafe(accType),
				),
			)
		} else {
			write(fmt.Sprintf("#txt(%%B[x:105,y:%d,w:120,h:24],{},\"%s\")", y, CMLSafe(accType)))
		}

		if strings.EqualFold(accType, "LCN") && accID != "" && lcnPlaceByID != nil {
			if place, ok := lcnPlaceByID[accID]; ok {
				y += 26

				write("#font(YF,YF,YF)")
				write(fmt.Sprintf("#txt(%%B[x:20,y:%d,w:100,h:100],{},\"Place:\")", y))
				write("#font(WF,WF,WF)")
				write(fmt.Sprintf("#txt(%%B[x:105,y:%d,w:100,h:100],{},\"%d\")", y, place))
			}
		}

		y += 26
	}

	if room != nil {
		write("#font(WF,WF,WF)")
		write(fmt.Sprintf("#txt(%%B[x:20,y:%d,w:100,h:100],{},\"Room\")", y))
		write("#font(YF,WF,WF)")
		write(fmt.Sprintf("#txt(%%B[x:105,y:%d,w:220,h:24],{},\"%s\")", y, CMLSafe(room.Title)))
		write(
			fmt.Sprintf(
				"#btn(%%B[x:105,y:%d,w:44,h:24],{GW|open&join_game.dcml&ASTATE=<%%ASTATE>^VE_RID=%d^BACKTO=user_details},\"join\")",
				y+20,
				room.ID,
			),
		)
		write(
			fmt.Sprintf(
				"#btn(%%B[x:151,y:%d,w:44,h:24],{GW|open&room_info_dgl.dcml&ASTATE=<%%ASTATE>^VE_RID=%d^BACKTO=user_details},\"info\")",
				y+20,
				room.ID,
			),
		)
	}

	write("#font(YF,WF,RF)")
	write(
		"#sbtn[%B_RGST](%B[xc:50%,y:100%+8,w:160,h:24],{LW_file&Internet/Cash/cancel.cml},\"Close\")",
	)
	write("<NGDLG>")

	return b.String()
}
