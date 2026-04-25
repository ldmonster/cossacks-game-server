package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Full-body goldens for every .tmpl under golang/templates (cs|ac).
// Regenerate with: go test ./internal/server/commands -golden -run TestShareTemplatesFullbodyGolden
//
// These snapshots document the Go TT-subset renderer output for on-disk templates
// (not Perl TT). Production routes that bypass loadShowBody (e.g. buildUserDetailsBody)
// still have their .tmpl files covered here for drift detection.
// shareGoldenTemplateRels must match every *.tmpl under templates/{ac,cs} (see parity test).
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

func ensureShareTemplateRootsForFullbody(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Clean(filepath.Join(wd, "../../../templates")),
		filepath.Clean(filepath.Join(wd, "../../../../golang/templates")),
	} {
		if st, e := os.Stat(rel); e == nil && st.IsDir() {
			templateRoots = []string{rel}
			return
		}
	}
	t.Fatal("could not resolve golang/templates for template fullbody tests")
}

func shareDirAbs(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Clean(filepath.Join(wd, "../../../templates")),
		filepath.Clean(filepath.Join(wd, "../../../../golang/templates")),
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

func shareTemplateGoldenBase(ver uint8) map[string]string {
	v := strconv.Itoa(int(ver))
	m := map[string]string{
		"ver":                             v,
		"id":                              "42",
		"nick":                            "GoldenNick",
		"P.NICK":                          "GoldenNick",
		"error_text":                      "golden error",
		"chat_server":                     "chat.example.invalid",
		"server.config.chat_server":       "chat.example.invalid",
		"server.config.table_timeout":     "10000",
		"server.config.show_started_rooms": "1",
		"h.connection.data.dev":           "1",
		"window_size":                     "800,600",
		"logged_in":                       "1",
		"type":                            "LCN",
		"table_timeout":                   "10000",
		"header":                          "Golden header",
		"text":                            "Golden body",
		"ok_text":                         "OK",
		"height":                          "188",
		"command":                         "GW|url&http://example.invalid/&from=golden",
		"ip":                              "192.0.2.1",
		"port":                            "34001",
		"max_pl":                          "8",
		"name":                            "GoldenRoom",
		"bottom_height":                   "32",
		"gg_cup":                          "1",
		"gg_cup.id":                       "99",
		"gg_cup.wo_info":                  "0",
		"gg_cup.started":                  "0",
		"gg_cup.players_count":            "12",
		"gg_cup.prize_fund":               "1000.5",
		"room_id":                         "7",
		"room_name":                       "GoldenRoom",
		"room_players":                    "3/8",
		"room_host":                       "1.2.3.4",
		"room_ctime":                      "1700000000",
		"room_started":                    "true",
		"room_max_pl":                     "8",
		"room_pl_count":                   "3",
		"room_time":                       "10 min",
		"backto":                          "",
		"room_players_start":              "3",
		"active_players":                  "p1, p2",
		"exited_players":                  "",
		"has_exited_players":              "0",
		"page":                            "1",
		"res":                             "0",
		"room.id":                         "7",
		"room.title":                      "GoldenRoom",
		"room.time":                       "250",
		"room.level":                      "2",
		"room.map":                        "golden.m3d",
		"room.host_id":                    "1",
		"room.started":                    "0",
		"room.start_players_count":        "3",
		"room.players_count":              "3",
		"room.max_players":                "8",
		"room.players.1.nick":             "hostnick",
		"server.data.start_at":            "2099-01-01 00:00:00 UTC",
		"player.account":                  "1",
		"player.nick":                     "PNick",
		"player.id":                       "10",
		"player.connected_at":             "2020-01-02 03:04:05 UTC",
		"connection_time":                 "9 min",
		"player.account.type":             "LCN",
		"player.account.profile":          "http://profile.example/golden",
		"player.account.id":               "555",
		"h.server.data.lcn_place_by_id.555": "3",
		"room":                            "1",
		"error":                           "",
	}
	return m
}

func varsForShareTemplateGolden(rel string, ver uint8) map[string]string {
	base := cloneStringMap(shareTemplateGoldenBase(ver)) // ver selects h.req.ver / ROOMS_V* via base["ver"]
	switch rel {
	case "cs/enter.tmpl":
		// Logged-out, no type: widest anonymous enter shell (stable height branch).
		base["logged_in"] = ""
		base["type"] = ""
		base["error"] = ""
	case "cs/ok_enter.tmpl":
		// Exercise window_size branch (large vs non-large) with non-large default.
		base["window_size"] = "small"
	case "ac/startup.tmpl", "cs/startup.tmpl":
		// GG cup marketing branch (not started, with info).
		base["gg_cup.started"] = "0"
		base["gg_cup.wo_info"] = "0"
	case "cs/started_room_info.tmpl", "cs/started_room_info/statcols.tmpl":
		base["room_started"] = "true"
		base["room.started"] = "1"
		base["room.time"] = "250"
	}
	return base
}

func goldenFullbodyPath(rel string) string {
	safe := strings.ReplaceAll(rel, "/", "__")
	return filepath.Join("testdata", "template_fullbody", safe+".golden")
}

func testShareTemplateFullbodyGolden(t *testing.T, rel string) {
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
	vars := varsForShareTemplateGolden(rel, ver)
	got := strings.TrimSpace(loadShowBody(ver, name, vars))
	if strings.Contains(got, "server response") && strings.Contains(got, "%BOX[x:10,y:10") {
		t.Fatalf("%s: got loadShowBody fallback (template file missing?)", rel)
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
		t.Fatalf("read %s: %v (re-run with: go test ./internal/server/commands -golden -run TestShareTemplatesFullbodyGolden)", path, err)
	}
	want := strings.TrimSuffix(string(wantB), "\n")
	if got != want {
		t.Fatalf("%s: fullbody golden mismatch (len got=%d want=%d)\n--- got ---\n%s\n--- want ---\n%s",
			rel, len(got), len(want), got, want)
	}
}

func TestShareTemplatesFullbodyGolden(t *testing.T) {
	ensureShareTemplateRootsForFullbody(t)
	for _, rel := range shareGoldenTemplateRels {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			testShareTemplateFullbodyGolden(t, rel)
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
