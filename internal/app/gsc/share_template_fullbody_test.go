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

package gsc

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ldmonster/cossacks-game-server/internal/render"
)

// Full-body goldens for every .tmpl under templates/ (cs|ac).
// Regenerate with: go test ./internal/server/handler -golden -run TestShareTemplatesFullbodyGolden
//
// These snapshots document the Go TT-subset renderer output for on-disk templates.
// Production routes that bypass loadShowBody (e.g. buildUserDetailsBody)
// still have their .tmpl files covered here for drift detection.
// shareGoldenTemplateRels must match every *.tmpl under templates/{ac,cs}.
var shareGoldenTemplateRels = []string{
	"ac/alert_dgl.tmpl",
	"ac/confirm_dgl.tmpl",
	"ac/confirm_password_dgl.tmpl",
	"ac/enter.tmpl",
	"ac/error_enter.tmpl",
	"ac/join_room.tmpl",
	"ac/new_room_dgl.tmpl",
	"ac/ok_enter.tmpl",
	"ac/reg_new_room.tmpl",
	"ac/startup.tmpl",
	"cs/alert_dgl.tmpl",
	"cs/confirm_dgl.tmpl",
	"cs/confirm_password_dgl.tmpl",
	"cs/enter.tmpl",
	"cs/error_enter.tmpl",
	"cs/gg_cup_thanks_dgl.tmpl",
	"cs/join_room.tmpl",
	"cs/new_room_dgl.tmpl",
	"cs/ok_enter.tmpl",
	"cs/reg_new_room.tmpl",
	"cs/room_info_dgl.tmpl",
	"cs/startup.tmpl",
	"cs/started_room_info.tmpl",
	"cs/started_room_info/statcols.tmpl",
	"cs/user_details.tmpl",
}

// shareTemplateRendererForFullbody resolves the on-disk templates/ directory
// (relative to this test package's cwd) and returns a renderer rooted at it.
// Tests use this in place of the (removed) package-global `templateRoots`.
func shareTemplateRendererForFullbody(t *testing.T) *render.TemplateRenderer {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Clean(filepath.Join(wd, "../../../templates")),
	} {
		if st, e := os.Stat(rel); e == nil && st.IsDir() {
			return render.NewTemplateRenderer(rel)
		}
	}
	t.Fatal("could not resolve templates/ for template fullbody tests")
	return nil
}

func shareDirAbs(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Clean(filepath.Join(wd, "../../../templates")),
	} {
		if st, e := os.Stat(rel); e == nil && st.IsDir() {
			return rel
		}
	}
	t.Fatal("templates dir not found")
	return ""
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// shareTemplateGoldenBaseView returns a `*render.View` populated with
// the canonical golden field set used by every share-template golden.
func shareTemplateGoldenBaseView(ver uint8) *render.View {
	v := strconv.Itoa(int(ver))

	room := &render.RoomView{
		ID:             7,
		Title:          "GoldenRoom",
		HostID:         1,
		HostNick:       "hostnick",
		HostAddrInt:    16909060,
		Level:          2,
		Started:        false,
		StartPlayers:   3,
		PlayersCount:   3,
		MaxPlayers:     8,
		Map:            "golden.m3d",
		CtimeFormatted: "2023-11-14 22:13:20 UTC (10 min ago)",
		Time:           "10 min",
		ActivePlayers:  []string{"p1", "p2"},
		ExitedPlayers:  nil,
		Players: map[uint32]*render.RoomPlayerView{
			1: {Nick: "hostnick"},
		},
	}

	player := &render.PlayerView{
		ID:                   10,
		Nick:                 "PNick",
		BoxHeight:            180,
		LastLabelRef:         "L_ACCOUNT",
		ConnectedAtFormatted: "2020-01-02 03:04:05 UTC (9 min ago)",
		Account: &render.AccountView{
			Type:     "LCN",
			Profile:  "http://profile.example/golden",
			HasPlace: true,
			Place:    3,
		},
		Room: &render.RoomView{
			ID:    7,
			Title: "GoldenRoom",
		},
	}

	ggCup := &render.GGCupView{
		ID:              "99",
		WoInfo:          false,
		Started:         false,
		PlayersCount:    "12",
		PlayersCountLen: 2,
		PrizeFund:       "1000",
		PrizeFundLen:    4,
		BoxHeight:       280,
	}

	return &render.View{
		Ver:          v,
		ID:           "42",
		Nick:         "GoldenNick",
		Nickname:     "GoldenNick",
		ChatServer:   "chat.example.invalid",
		ServerName:   "my-server.example",
		ServerTitle:  "Example Game Server",
		StartAt:      "2099-01-01 00:00:00 UTC",
		TableTimeout: 10000,
		ShowStarted:  false,
		Dev:          true,
		WindowSize:   "800,600",
		LoggedIn:     true,
		Type:         "LCN",
		Header:       "Golden header",
		Text:         "Golden body",
		OkText:       "OK",
		Height:       188,
		Command:      "GW|url&http://example.invalid/&from=golden",
		IP:           "192.0.2.1",
		Port:         34001,
		MaxPl:        8,
		Name:         "GoldenRoom",
		BottomHeight: 32,
		PlayerID:     "10",
		HolePort:     34002,
		HoleHost:     "hole.example",
		HoleInt:      5,
		Error:        "",
		Room:         room,
		Player:       player,
		GGCup:        ggCup,
	}
}

func viewForShareTemplateGolden(rel string, ver uint8) *render.View {
	view := shareTemplateGoldenBaseView(ver)

	switch rel {
	case "cs/enter.tmpl":
		view.LoggedIn = false
		view.Type = ""
		view.Error = ""
	case "cs/ok_enter.tmpl":
		view.WindowSize = "small"
	case "cs/started_room_info.tmpl", "cs/started_room_info/statcols.tmpl":
		view.Room.Started = true
		view.Room.Time = "250"

		view.StartedRoom = &render.StartedRoomView{
			ID:           7,
			Title:        "GoldenRoom",
			Map:          "golden.m3d",
			Level:        2,
			RoomTime:     "10 min",
			TimeTickSecs: 10,
			SmallColW:    49,
			LargeColW:    54,
			Page:         "1",
			Res:          "",
			Players: []*render.StartedPlayerView{
				{
					Nick:   "p1",
					Color:  16711935,
					Theam:  1,
					Nation: 1,
					Stat: &render.StartedPlayerStatView{
						RealScores:  100,
						Population:  20,
						ChangeWood:  "1.0",
						ChangeFood:  "0.5",
						ChangeStone: "0.2",
						ChangeGold:  "0.3",
						ChangeIron:  "0.1",
						ChangeCoal:  "0.0",
						ChangePop2:  "0.0",
						Casuality:   0,
					},
				},
			},
		}
	}

	return view
}

func goldenFullbodyPath(rel string) string {
	safe := strings.ReplaceAll(rel, "/", "__")
	return filepath.Join("testdata", "template_fullbody", safe+".golden")
}

func testShareTemplateFullbodyGolden(t *testing.T, r *render.TemplateRenderer, rel string) {
	t.Helper()
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("bad rel %q", rel)
	}
	dir, name := parts[0], parts[1]
	var ver uint8 = 2
	if dir == "ac" {
		ver = 8
	}
	vars := viewForShareTemplateGolden(rel, ver)
	got := strings.TrimSpace(r.Render(ver, name, vars))
	if strings.Contains(got, "server response") && strings.Contains(got, "%BOX[x:10,y:10") {
		t.Fatalf("%s: got renderer fallback (template file missing?)", rel)
	}
	path := goldenFullbodyPath(rel)
	if *golden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
		return
	}
	wantB, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (re-run with: go test ./internal/server/handler -golden -run TestShareTemplatesFullbodyGolden)", path, err)
	}
	want := strings.TrimSuffix(string(wantB), "\n")
	if got != want {
		t.Fatalf("%s: fullbody golden mismatch (len got=%d want=%d)\n--- got ---\n%s\n--- want ---\n%s",
			rel, len(got), len(want), got, want)
	}
}

func TestShareTemplatesFullbodyGolden(t *testing.T) {
	r := shareTemplateRendererForFullbody(t)
	for _, rel := range shareGoldenTemplateRels {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			testShareTemplateFullbodyGolden(t, r, rel)
		})
	}
}

func TestShareTemplateGoldenCoversAllShareTemplates(t *testing.T) {
	root := shareDirAbs(t)
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".tmpl") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(found)

	want := append([]string(nil), shareGoldenTemplateRels...)
	sort.Strings(want)
	if len(found) != len(want) {
		t.Fatalf("share .tmpl count %d != golden list %d\nfound: %q\nwant: %q", len(found), len(want), found, want)
	}
	for i := range found {
		if found[i] != want[i] {
			t.Fatalf("share .tmpl set differs at %d: found %q want %q\nfound: %q\nwant: %q", i, found[i], want[i], found, want)
		}
	}
}
