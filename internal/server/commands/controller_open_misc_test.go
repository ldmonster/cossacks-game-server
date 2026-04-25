package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

func TestUsersListReturnsNotImplementedError(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "users_list", map[string]string{})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Not imlemented") {
		t.Fatalf("expected Not imlemented alert, got %#v", out)
	}
}

func TestTournamentsReturnsInternalServerErrorWithoutRanking(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "tournaments", map[string]string{})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "Internal server error") {
		t.Fatalf("expected internal server error alert, got %#v", out)
	}
}

func TestLcnRegistrationDialogContent(t *testing.T) {
	c := newControllerForJoinTests()
	c.Config.Raw["lcn_host"] = "www.newlcn.com"
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "lcn_registration_dgl", map[string]string{})
	if len(out) != 1 {
		t.Fatalf("expected one response, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "LCN Registration") || !strings.Contains(body, "Open www.newlcn.com?") {
		t.Fatalf("unexpected dialog body: %q", body)
	}
	if !strings.Contains(body, "lang_redir.php") {
		t.Fatalf("expected redirect command in dialog: %q", body)
	}
}

func TestNewRoomDialogRespectsAstate(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	blocked := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "0"})
	if len(blocked) != 1 || !strings.Contains(blocked[0].Args[0], "You can not create or join room!") {
		t.Fatalf("expected astate error, got %#v", blocked)
	}
	allowed := c.dispatchOpen(nil, conn, req, "new_room_dgl", map[string]string{"ASTATE": "1"})
	if len(allowed) != 1 || !strings.Contains(allowed[0].Args[0], "Create new game") {
		t.Fatalf("expected new_room_dgl payload, got %#v", allowed)
	}
}

func TestResizeReturnsLargePayload(t *testing.T) {
	c := newControllerForJoinTests()
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "resize", map[string]string{"height": "420"})
	if len(out) != 1 || out[0].Args[0] != "<RESIZE>\n#large\n<RESIZE>" {
		t.Fatalf("unexpected resize payload: %#v", out)
	}
}

func TestTryEnterLoggedInPostsEnterAccountAction(t *testing.T) {
	var posted url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		posted = r.PostForm
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	c := newControllerForJoinTests()
	c.HTTP = srv.Client()
	c.Config.Raw["lcn_host"] = host
	c.Config.Raw["lcn_key"] = "secret"
	conn := &model.Connection{
		IP:   "127.0.0.1",
		Ctime: time.Now(),
		Data: map[string]any{
			"account": map[string]string{
				"type":  "LCN",
				"login": "User[1]",
				"id":    "42",
			},
		},
	}
	req := &gsc.Stream{Ver: 2}
	out := c.tryEnter(nil, conn, req, map[string]string{"LOGGED_IN": "1"})
	if len(out) != 1 {
		t.Fatalf("expected success enter payload, got %#v", out)
	}
	if posted.Get("action") != "enter" || posted.Get("account_id") != "42" {
		t.Fatalf("unexpected account action form: %#v", posted)
	}
	if posted.Get("key") != "secret" || posted.Get("time") == "" {
		t.Fatalf("expected key and time, got %#v", posted)
	}
	if posted.Get("data") != "" {
		t.Fatalf("enter action should not send data payload, got %q", posted.Get("data"))
	}
}

func TestTournamentsUsesRankingSuccessPath(t *testing.T) {
	c := newControllerForJoinTests()
	dir := t.TempDir()
	path := filepath.Join(dir, "ranking.json")
	if err := os.WriteFile(path, []byte(`{
		"ranking": {
			"total": [
				{"id": 1, "place": 1, "nick": "Alpha", "score": 1200},
				{"id": 2, "place": 2, "nick": "Beta", "score": 1100}
			]
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.Raw["lcn_ranking"] = path
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "tournaments", map[string]string{"option": "total"})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "LCN Rating: total") || !strings.Contains(out[0].Args[0], "Alpha") {
		t.Fatalf("expected tournaments success payload, got %#v", out)
	}
}

func TestGGCupThanksUsesSupportersData(t *testing.T) {
	c := newControllerForJoinTests()
	dir := t.TempDir()
	path := filepath.Join(dir, "gg.json")
	if err := os.WriteFile(path, []byte(`{
		"supporters": [
			{"nick":"DonorOne","amount":500,"url":"http://example.com/u/1"},
			{"nick":"DonorTwo","amount":250,"url":"http://example.com/u/2"}
		]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.Raw["gg_cup_file"] = path
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "gg_cup_thanks_dgl", map[string]string{})
	body := out[0].Args[0]
	if len(out) != 1 || !strings.Contains(body, "<NGDLG>") || !strings.Contains(body, "DonorOne") || !strings.Contains(body, "RUB") {
		t.Fatalf("expected gg cup thanks dialog payload, got %#v", out)
	}
	if !strings.Contains(body, "GW|url&http://example.com/u/1") {
		t.Fatalf("expected profile url command in payload: %q", body)
	}
}

func TestGGCupThanksOverflowLineMatchesPerl(t *testing.T) {
	c := newControllerForJoinTests()
	dir := t.TempDir()
	path := filepath.Join(dir, "gg.json")
	supporters := make([]map[string]any, 18)
	for i := range supporters {
		supporters[i] = map[string]any{
			"nick": fmt.Sprintf("U%d", i), "amount": 1, "url": "http://x",
		}
	}
	raw, err := json.Marshal(map[string]any{"supporters": supporters})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.Raw["gg_cup_file"] = path
	req := &gsc.Stream{Ver: 2}
	conn := &model.Connection{Data: map[string]any{}}
	out := c.dispatchOpen(nil, conn, req, "gg_cup_thanks_dgl", map[string]string{})
	if len(out) != 1 || !strings.Contains(out[0].Args[0], "and more...") {
		t.Fatalf("expected overflow line for >17 supporters, got %#v", out)
	}
}

func TestUserDetailsLCNPlaceFromRanking(t *testing.T) {
	c := newControllerForJoinTests()
	dir := t.TempDir()
	rankPath := filepath.Join(dir, "ranking.json")
	if err := os.WriteFile(rankPath, []byte(`{
		"ranking": {
			"total": [
				{"id": 42, "place": 7, "nick": "X", "score": 100}
			]
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c.Config.Raw["lcn_ranking"] = rankPath
	c.Store.Players[3] = &state.Player{
		ID:   3,
		Nick: "ranked",
		ConnectedAt: time.Now().UTC().Add(-time.Hour),
		Account: map[string]any{
			"type": "LCN",
			"id":   "42",
		},
	}
	conn := &model.Connection{Data: map[string]any{}}
	req := &gsc.Stream{Ver: 2}
	out := c.userDetails(conn, req, map[string]string{"ID": "3"})
	if len(out) != 1 {
		t.Fatalf("expected one LW_show, got %#v", out)
	}
	body := out[0].Args[0]
	if !strings.Contains(body, "Place:") || !strings.Contains(body, "7") {
		t.Fatalf("expected LCN place from ranking, got: %q", body)
	}
}
