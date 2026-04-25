package commands

import (
	"context"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cossacksgameserver/golang/internal/config"
	"cossacksgameserver/golang/internal/integration"
	"cossacksgameserver/golang/internal/protocol/gsc"
	"cossacksgameserver/golang/internal/server/model"
	"cossacksgameserver/golang/internal/server/state"
)

type Controller struct {
	Config *config.Config
	Store  *state.Store
	Redis  *integration.RedisClient
	HTTP   *http.Client

	lcnRankingMTime int64
	lcnRankingData  map[string]any
	// lcnPlaceByID mirrors Perl server data lcn_place_by_id (from lcn_ranking file).
	lcnPlaceByID map[string]int
	ggCupMTime   int64
	ggCupData    map[string]any

	stateMu     sync.Mutex
	mu          sync.Mutex
	aliveTimers map[uint32]*time.Timer
	playerConns map[uint32]*model.Connection
	aliveTTL    time.Duration
}

type HandleResult struct {
	Commands    []gsc.Command
	HasResponse bool
}

func (c *Controller) Handle(
	ctx context.Context,
	conn *model.Connection,
	req *gsc.Stream,
	cmdName string,
	args []string,
	win string,
	key string,
) []gsc.Command {
	_ = key
	_ = win
	r := c.HandleWithMeta(ctx, conn, req, cmdName, args, win, key)
	if !r.HasResponse {
		return nil
	}
	return r.Commands
}

func (c *Controller) HandleWithMeta(
	ctx context.Context,
	conn *model.Connection,
	req *gsc.Stream,
	cmdName string,
	args []string,
	win string,
	key string,
) HandleResult {
	// Phase 6 parity hardening: serialize state mutations to avoid map races
	// across concurrent connection goroutines and alive-timeout callbacks.
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	_ = key
	_ = win
	switch cmdName {
	case "login":
		return HandleResult{Commands: []gsc.Command{{Name: "LW_show", Args: []string{":GW|open&enter.dcml"}}}, HasResponse: true}
	case "echo":
		return HandleResult{Commands: []gsc.Command{{Name: "LW_echo", Args: args}}, HasResponse: true}
	case "open":
		return HandleResult{Commands: c.handleOpen(ctx, conn, req, args), HasResponse: true}
	case "go":
		return HandleResult{Commands: c.handleGo(ctx, conn, req, args), HasResponse: true}
	case "leave":
		c.leaveRoom(conn)
		return HandleResult{HasResponse: false}
	case "alive":
		c.refreshAlive(conn)
		return HandleResult{HasResponse: false}
	case "proxy":
		c.handleProxy(conn, args)
		return HandleResult{HasResponse: false}
	case "stats":
		c.handleStatsLocked(conn, args)
		return HandleResult{HasResponse: false}
	case "endgame":
		c.handleEndgame(conn, args)
		return HandleResult{HasResponse: false}
	case "upfile", "unsync":
		return HandleResult{HasResponse: false}
	case "GETTBL":
		return HandleResult{Commands: c.handleGETTBL(conn, req, args), HasResponse: true}
	case "start":
		return HandleResult{Commands: c.handleStart(conn, req, args), HasResponse: false}
	case "url":
		if len(args) == 0 {
			return HandleResult{HasResponse: false}
		}
		return HandleResult{Commands: []gsc.Command{{Name: "LW_time", Args: []string{"0", "open:" + args[0]}}}, HasResponse: true}
	default:
		log.Printf("unknown command: %s", cmdName)
		// Perl _default pushes an empty response; emulate that.
		return HandleResult{Commands: []gsc.Command{}, HasResponse: true}
	}
}

func (c *Controller) ensureRuntimeMaps() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.aliveTimers == nil {
		c.aliveTimers = map[uint32]*time.Timer{}
	}
	if c.playerConns == nil {
		c.playerConns = map[uint32]*model.Connection{}
	}
	if c.aliveTTL <= 0 {
		c.aliveTTL = 150 * time.Second
	}
}

func (c *Controller) handleOpen(ctx context.Context, conn *model.Connection, req *gsc.Stream, args []string) []gsc.Command {
	if len(args) < 1 {
		return []gsc.Command{}
	}
	rawURL := strings.TrimSpace(strings.ReplaceAll(args[0], "\x00", ""))
	url := strings.TrimSuffix(rawURL, ".dcml")
	params := map[string]string{}
	if len(args) > 1 {
		params = parseOpenParams(strings.ReplaceAll(args[1], "\x00", ""))
	}
	log.Printf("open route debug: conn_id=%d raw_url=%q parsed_method=%q params=%v", conn.ID, rawURL, url, params)
	return c.dispatchOpen(ctx, conn, req, url, params)
}

func (c *Controller) handleGo(ctx context.Context, conn *model.Connection, req *gsc.Stream, args []string) []gsc.Command {
	if len(args) < 1 {
		return []gsc.Command{}
	}
	method := args[0]
	params := map[string]string{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if k, v, ok := strings.Cut(arg, "="); ok {
			params[k] = v
			continue
		}
		if strings.HasSuffix(arg, ":=") && i+1 < len(args) {
			params[strings.TrimSuffix(arg, ":=")] = args[i+1]
			i++
		}
	}
	log.Printf("go route debug: conn_id=%d method=%q params=%v", conn.ID, method, params)
	return c.dispatchOpen(ctx, conn, req, method, params)
}

func (c *Controller) dispatchOpen(ctx context.Context, conn *model.Connection, req *gsc.Stream, method string, p map[string]string) []gsc.Command {
	switch method {
	case "enter":
		// Perl parity: when account data exists, enter page is rendered in logged-in mode.
		if account, ok := conn.Data["account"].(map[string]string); ok && account["type"] != "" {
			return c.renderEnter(req, account["type"], "", "1", account["login"], account["id"])
		}
		if account, ok := conn.Data["account"].(map[string]any); ok {
			typ, _ := account["type"].(string)
			login, _ := account["login"].(string)
			id, _ := account["id"].(string)
			if typ != "" {
				return c.renderEnter(req, typ, "", "1", login, id)
			}
		}
		return c.renderEnter(req, p["TYPE"], "", "", "", "")
	case "try_enter":
		return c.tryEnter(ctx, conn, req, p)
	case "startup", "games", "rooms_table_dgl":
		vars := map[string]string{
			"window_size":   windowSize(conn),
			"chat_server":   c.Config.ChatServer,
			"table_timeout": strconv.Itoa(c.Config.TableTimeout),
			"ver":           strconv.Itoa(int(req.Ver)),
			// cs/startup.tmpl defines this via TT SET; provide explicit value so
			// the generated show body keeps valid y/h coordinates for the bottom bar.
			"bottom_height": "32",
		}
		mergeGGCupIntoStartupVars(c.loadGGCup(), vars)
		body := loadShowBody(req.Ver, "startup.tmpl", vars)
		btnLines := make([]string, 0, 8)
		for _, ln := range strings.Split(body, "\n") {
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "#btn(") || strings.HasPrefix(trim, "#btn[") {
				btnLines = append(btnLines, trim)
			}
		}
		log.Printf(
			"startup payload debug: conn_id=%d ver=%d has_join=%t has_new=%t len=%d btn_lines=%v",
			conn.ID,
			req.Ver,
			strings.Contains(body, "join_game.dcml"),
			strings.Contains(body, "new_room_dgl.dcml"),
			len(body),
			btnLines,
		)
		return []gsc.Command{{Name: "LW_show", Args: []string{body}}}
	case "resize":
		return c.resize(conn, p)
	case "new_room_dgl":
		return c.newRoomDialog(req, p)
	case "reg_new_room":
		return c.regNewRoom(conn, req, p)
	case "join_game":
		return c.joinGame(ctx, conn, req, p)
	case "room_info_dgl":
		return c.roomInfo(conn, req, p)
	case "join_pl_cmd":
		return c.joinPlayer(conn, p)
	case "user_details":
		return c.userDetails(conn, req, p)
	case "direct", "direct_ping", "direct_join", "started_room_message":
		// Perl exposes these routes but currently has no concrete body.
		// Keep open-route parity and return an empty response payload.
		return []gsc.Command{}
	case "users_list":
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "Not imlemented",
		})}}}
	case "tournaments":
		return c.tournaments(req, p)
	case "lcn_registration_dgl":
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "confirm_dgl.tmpl", map[string]string{
			"header":  "LCN Registration",
			"text":    "Open www.newlcn.com?",
			"ok_text": "Ok",
			"height":  "100",
			"command": "GW|url&http://" + c.Config.Raw["lcn_host"] + "/lang_redir.php&from=tournaments",
		})}}}
	case "gg_cup_thanks_dgl":
		return c.ggCupThanks(req)
	default:
		log.Printf("open route not found: conn_id=%d method=%q params=%v", conn.ID, method, p)
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
				"header": "Error",
				"text":   "Page Not Found",
			}),
		}}}
	}
}

func (c *Controller) tournaments(req *gsc.Stream, p map[string]string) []gsc.Command {
	option := strings.TrimSpace(p["option"])
	if option == "" {
		option = "total"
	}
	rating := c.loadLCNRanking()
	if rating == nil {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "Internal server error",
		})}}}
	}
	rankingByOption, _ := rating["ranking"].(map[string]any)
	rowsAny, _ := rankingByOption[option].([]any)
	if len(rowsAny) == 0 && option != "total" {
		rowsAny, _ = rankingByOption["total"].([]any)
	}
	lines := make([]string, 0, 12)
	lines = append(lines, "LCN Rating: "+option)
	for i, rowAny := range rowsAny {
		if i >= 10 {
			break
		}
		row, _ := rowAny.(map[string]any)
		place := fmt.Sprintf("%v", row["place"])
		nick := fmt.Sprintf("%v", row["nick"])
		score := fmt.Sprintf("%v", row["score"])
		lines = append(lines, fmt.Sprintf("%s. %s (%s)", place, nick, score))
	}
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
		"header": "Tournaments",
		"text":   strings.Join(lines, "\n"),
	})}}}
}

func (c *Controller) ggCupThanks(req *gsc.Stream) []gsc.Command {
	ggCup := c.loadGGCup()
	if ggCup == nil || toBool(ggCup["wo_info"]) {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Thanks for",
			"text":   "No info yet",
		})}}}
	}
	supportersAny, _ := ggCup["supporters"].([]any)
	supporters := make([]map[string]any, 0, len(supportersAny))
	for _, sAny := range supportersAny {
		s, ok := sAny.(map[string]any)
		if !ok {
			continue
		}
		supporters = append(supporters, s)
	}
	if len(supporters) == 0 {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Thanks for",
			"text":   "No info yet",
		})}}}
	}
	return []gsc.Command{{Name: "LW_show", Args: []string{buildGGCupThanksBody(supporters)}}}
}

// ggCupThanksBoxHeight mirrors Perl started_room_info / gg_cup_thanks_dgl height logic.
func ggCupThanksBoxHeight(supporterCount int) int {
	const rows = 17
	if supporterCount <= 9 {
		return 280
	}
	if supporterCount > rows {
		return 55 + (rows+1)*25
	}
	return 55 + supporterCount*25
}

// buildGGCupThanksBody emits a dialog shaped like share/cs/gg_cup_thanks_dgl (Perl show payload).
func buildGGCupThanksBody(supporters []map[string]any) string {
	var b strings.Builder
	n := len(supporters)
	h := ggCupThanksBoxHeight(n)
	fmt.Fprintf(&b, "<NGDLG>\n")
	b.WriteString("#exec(LW_lockbox&%LBX)\n")
	b.WriteString("#exec(LW_enb&0&%RMLST)\n")
	fmt.Fprintf(&b, "#ebox[%%B](x:215,y:10,w:320,h:%d)\n", h)
	b.WriteString("#pan[%MPN](%B[x:0,y:0,w:100%,h:100%],8)\n")
	b.WriteString("#font(WF,WF,WF)\n")
	b.WriteString("#ctxt[%TIT](%B[x:0,y:6,w:100%,h:30],{},\"Thanks for\")\n\n")
	// Perl gg_cup_thanks_dgl: rows=17; when loop.index==rows on an extra supporter, emit "and more..." and LAST.
	// With 0-based indexing: render supporter i for i < 17; if n > 17 and i == 17, emit overflow line (Perl TT).
	const rows = 17
	yoff := 43
	for i := 0; i < n; i++ {
		if n > rows && i == rows {
			b.WriteString("#font(YF,YF,YF)\n")
			fmt.Fprintf(&b, "#txt(%%B[x:20,y:%d,w:100%%,h:25],{},\"and more...\")\n", yoff+3)
			break
		}
		s := supporters[i]
		nick := cmlSafe(fmt.Sprintf("%v", s["nick"]))
		amt := supporterAmountString(s["amount"])
		url := cmlSafe(fmt.Sprintf("%v", s["url"]))
		b.WriteString("#font(YF,YF,YF)\n")
		fmt.Fprintf(&b, "#txt(%%B[x:20,y:%d,w:100%%,h:25],{},\"%s\")\n", yoff+3, nick)
		b.WriteString("#font(WF,WF,WF)\n")
		fmt.Fprintf(&b, "#rtxt(%%B[x:100%%-204,y:%d,w:100,h:25],{},\"%s RUB \")\n", yoff+3, amt)
		fmt.Fprintf(&b, "#btn(%%B[x:230,y:%d,w:72,h:25],{GW|url&%s},\"profile\")\n", yoff, url)
		yoff += 25
	}
	b.WriteString("\n#font(YF,WF,RF)\n")
	b.WriteString("#sbtn[%B_RGST](%B[xc:50%,y:100%+8,w:160,h:24],{LW_file&Internet/Cash/cancel.cml},\"Ok\")\n")
	b.WriteString("<NGDLG>\n")
	return b.String()
}

func supporterAmountString(v any) string {
	switch t := v.(type) {
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return strconv.FormatInt(n, 10)
		}
		s := strings.TrimSpace(t.String())
		if s == "" {
			return "0"
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
		return s
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" {
			return "0"
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
		return s
	}
}

func (c *Controller) loadLCNRanking() map[string]any {
	path := strings.TrimSpace(c.Config.Raw["lcn_ranking"])
	if path == "" {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	mtime := st.ModTime().Unix()
	if c.lcnRankingData != nil && c.lcnRankingMTime == mtime {
		return c.lcnRankingData
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if rankingAny, ok := payload["ranking"].(map[string]any); ok {
		if total, ok := rankingAny["total"].([]any); ok {
			placeByID := map[string]int{}
			for _, rowAny := range total {
				row, _ := rowAny.(map[string]any)
				id := fmt.Sprintf("%v", row["id"])
				place := 0
				switch v := row["place"].(type) {
				case float64:
					place = int(v)
				case int:
					place = v
				}
				if id != "" {
					placeByID[id] = place
				}
			}
			c.lcnPlaceByID = placeByID
			c.Config.Raw["lcn_place_count"] = strconv.Itoa(len(placeByID))
		} else {
			c.lcnPlaceByID = nil
		}
	} else {
		c.lcnPlaceByID = nil
	}
	c.lcnRankingMTime = mtime
	c.lcnRankingData = payload
	return payload
}

func (c *Controller) loadGGCup() map[string]any {
	path := strings.TrimSpace(c.Config.Raw["gg_cup_file"])
	if path == "" {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		c.ggCupData = map[string]any{"wo_info": true}
		return c.ggCupData
	}
	mtime := st.ModTime().Unix()
	if c.ggCupData != nil && c.ggCupMTime == mtime {
		return c.ggCupData
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		c.ggCupMTime = mtime
		c.ggCupData = map[string]any{"wo_info": true}
		return c.ggCupData
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		c.ggCupMTime = mtime
		c.ggCupData = map[string]any{"wo_info": true}
		return c.ggCupData
	}
	c.ggCupMTime = mtime
	c.ggCupData = payload
	return payload
}

// mergeGGCupIntoStartupVars flattens Perl's gg_cup hash (Open.pm startup) into
// the string table expected by the TT fragment renderer for startup.tmpl.
func mergeGGCupIntoStartupVars(gg map[string]any, vars map[string]string) {
	if gg == nil {
		return
	}
	// Map presence: Perl TT treats a missing/undef gg_cup as falsy; a loaded hash
	// (including the wo_info error stub) is truthy.
	vars["gg_cup"] = "1"
	for _, k := range []string{"id", "wo_info", "started", "players_count", "prize_fund"} {
		v, ok := gg[k]
		if !ok {
			continue
		}
		vars["gg_cup."+k] = anyToStringVar(v)
	}
}

func anyToStringVar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "1"
		}
		return "0"
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (c *Controller) resize(conn *model.Connection, p map[string]string) []gsc.Command {
	height, err := strconv.Atoi(strings.TrimSpace(p["height"]))
	if err == nil {
		conn.Data["height"] = height
	}
	if windowSize(conn) == "large" {
		return []gsc.Command{{Name: "LW_show", Args: []string{"<RESIZE>\n#large\n<RESIZE>"}}}
	}
	return []gsc.Command{{Name: "LW_show", Args: []string{"<RESIZE>\n<RESIZE>"}}}
}

func (c *Controller) newRoomDialog(req *gsc.Stream, p map[string]string) []gsc.Command {
	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		})}}}
	}
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "new_room_dgl.tmpl", map[string]string{})}}}
}

func (c *Controller) userDetails(conn *model.Connection, req *gsc.Stream, p map[string]string) []gsc.Command {
	_ = c.loadLCNRanking()
	id, err := parseUint32Arg(p["ID"])
	if err != nil {
		return []gsc.Command{}
	}
	player := c.Store.Players[id]
	if player == nil {
		return []gsc.Command{}
	}
	room := c.Store.RoomsByPID[player.ID]
	return []gsc.Command{{Name: "LW_show", Args: []string{
		c.buildUserDetailsBody(conn, player, room),
	}}}
}

func (c *Controller) buildUserDetailsBody(conn *model.Connection, player *state.Player, room *state.Room) string {
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
	if toBool(conn.Data["dev"]) {
		write(fmt.Sprintf("#rtxt(%%B[x:280,y:6,w:70,h:30],{},\"#%d\")", player.ID))
	}
	write("#font(WF,WF,WF)")
	write("#txt[%L_NAME](%B[x:20,y:48,w:100,h:100],{},\"Nick\")")
	write("#font(YF,YF,YF)")
	write(fmt.Sprintf("#txt(%%B[x:105,y:48,w:200,h:100],{},\"%s\")", cmlSafe(player.Nick)))
	write("#font(WF,YF,WF)")
	write("#txt[%L_CTIME](%B[x:20,y:74,w:100,h:100],{},\"Connected at\")")
	write("#font(YF,WF,WF)")
	write(fmt.Sprintf("#txt(%%B[x:105,y:74,w:240,h:100],{},\"%s (%s ago)\")",
		player.ConnectedAt.UTC().Format("2006-01-02 15:04:05 UTC"),
		roomTimeInterval(&state.Room{Ctime: player.ConnectedAt}),
	))
	y := 100
	if player.Account != nil {
		accType, _ := player.Account["type"].(string)
		profile, _ := player.Account["profile"].(string)
		accID := fmt.Sprintf("%v", player.Account["id"])
		write("#font(WF,WF,WF)")
		write(fmt.Sprintf("#txt(%%B[x:20,y:%d,w:100,h:100],{},\"Logon with\")", y))
		if profile != "" {
			write(fmt.Sprintf("#btn(%%B[x:105,y:%d,w:120,h:24],{GW|url&%s&from=user_details},\"%s\")", y, cmlSafe(profile), cmlSafe(accType)))
		} else {
			write(fmt.Sprintf("#txt(%%B[x:105,y:%d,w:120,h:24],{},\"%s\")", y, cmlSafe(accType)))
		}
		if strings.EqualFold(accType, "LCN") && accID != "" {
			if place, ok := c.lcnPlaceByID[accID]; ok {
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
		write(fmt.Sprintf("#txt(%%B[x:105,y:%d,w:220,h:24],{},\"%s\")", y, cmlSafe(room.Title)))
		write(fmt.Sprintf("#btn(%%B[x:105,y:%d,w:44,h:24],{GW|open&join_game.dcml&ASTATE=<%%ASTATE>^VE_RID=%d^BACKTO=user_details},\"join\")", y+20, room.ID))
		write(fmt.Sprintf("#btn(%%B[x:151,y:%d,w:44,h:24],{GW|open&room_info_dgl.dcml&ASTATE=<%%ASTATE>^VE_RID=%d^BACKTO=user_details},\"info\")", y+20, room.ID))
	}
	write("#font(YF,WF,RF)")
	write("#sbtn[%B_RGST](%B[xc:50%,y:100%+8,w:160,h:24],{LW_file&Internet/Cash/cancel.cml},\"Close\")")
	write("<NGDLG>")
	return b.String()
}

func cmlSafe(v string) string {
	v = strings.ReplaceAll(v, "\"", "'")
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	return v
}

func (c *Controller) tryEnter(ctx context.Context, conn *model.Connection, req *gsc.Stream, p map[string]string) []gsc.Command {
	_ = ctx
	nick := strings.TrimSpace(p["NICK"])
	loginType := strings.TrimSpace(p["TYPE"])
	if strings.HasSuffix(nick, "#dev4231") {
		nick = strings.TrimSuffix(nick, "#dev4231")
		conn.Data["dev"] = true
	} else {
		conn.Data["dev"] = false
	}

	// Perl: explicit logout flow returns enter screen.
	if p["RESET"] != "" {
		delete(conn.Data, "account")
		return c.renderEnter(req, "", "", "", "", "")
	}

	// Perl: already logged-in account can proceed without password branch.
	if p["LOGGED_IN"] != "" {
		if account, ok := conn.Data["account"].(map[string]string); ok && account["login"] != "" {
			nick = sanitizeAccountNick(account["login"])
			c.postAccountAction(conn, "enter", nil)
			return c.successEnter(conn, req, nick)
		}
		return c.renderEnter(req, "", "", "", "", "")
	}

	// Perl: LCN/WCL branch asks for login/password and performs remote auth.
	// Go implementation currently keeps strict UX semantics even without remote auth.
	if loginType == "LCN" || loginType == "WCL" {
		if nick == "" {
			return c.renderEnter(req, loginType, "enter nick", "", "", "")
		}
		if strings.TrimSpace(p["PASSWORD"]) == "" {
			return c.renderEnter(req, loginType, "enter password", "", "", "")
		}
		account, errText := c.authenticateAccount(conn, loginType, nick, strings.TrimSpace(p["PASSWORD"]))
		if errText != "" {
			return c.renderEnter(req, loginType, errText, "", "", "")
		}
		conn.Data["account"] = account
		nick = sanitizeAccountNick(account["login"])
		return c.successEnter(conn, req, nick)
	}

	if h, err := strconv.Atoi(strings.TrimSpace(p["HEIGHT"])); err == nil {
		conn.Data["height"] = h
	}
	if nick == "" {
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "error_enter.tmpl", map[string]string{"error_text": "Enter nick"}),
		}}}
	}
	if ok, _ := regexp.MatchString(`^[\[\]_\w-]+$`, nick); !ok {
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "error_enter.tmpl", map[string]string{
				"error_text": "Bad character in nick. Nick can contain only a-z,A-Z,0-9,[]_-",
			}),
		}}}
	}
	if strings.HasPrefix(nick, "-") || (nick[0] >= '0' && nick[0] <= '9') {
		msg := "Bad character in nick. Nick can't start with numerical digit"
		if strings.HasPrefix(nick, "-") {
			msg = "Bad character in nick. Nick can't start with -"
		}
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "error_enter.tmpl", map[string]string{"error_text": msg}),
		}}}
	}
	if len(nick) > 25 {
		nick = nick[:25]
	}
	return c.successEnter(conn, req, nick)
}

func (c *Controller) authenticateAccount(conn *model.Connection, loginType, nick, password string) (map[string]string, string) {
	hostKey := strings.ToLower(loginType) + "_host"
	keyKey := strings.ToLower(loginType) + "_key"
	serverNameKey := strings.ToLower(loginType) + "_server_name"
	host := c.Config.Raw[hostKey]
	secret := c.Config.Raw[keyKey]
	serverName := c.Config.Raw[serverNameKey]
	if serverName == "" {
		serverName = host
	}
	if host == "" || secret == "" {
		return nil, "problem with " + serverName + " server"
	}

	form := url.Values{}
	form.Set("action", "logon")
	form.Set("key", secret)
	form.Set("login", nick)
	form.Set("password", password)
	endpoint := "http://" + host + "/api/server.php"
	req, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Client-IP", conn.IP)
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("auth request failed: %v", err)
		return nil, "problem with " + serverName + " server"
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("auth bad response from %s: %s", endpoint, resp.Status)
		return nil, "problem with " + serverName + " server"
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "problem with " + serverName + " server"
	}
	var payload struct {
		Success bool   `json:"success"`
		ID      any    `json:"id"`
		Profile string `json:"profile"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		log.Printf("auth bad json from %s: %v", endpoint, err)
		return nil, "problem with " + serverName + " server"
	}
	if !payload.Success {
		return nil, "incorrect login or password"
	}
	id := fmt.Sprintf("%v", payload.ID)
	acc := map[string]string{
		"type":  loginType,
		"login": nick,
		"id":    id,
	}
	if payload.Profile != "" {
		acc["profile"] = payload.Profile
	}
	if loginType == "LCN" && c.Config.Raw["lcn_host"] != "" {
		acc["profile"] = "http://" + c.Config.Raw["lcn_host"] + "/lang_redir.php?path=player.php?plid=" + id
	}
	return acc, ""
}

// postAccountAction mirrors Perl SimpleCossacksServer::post_account_action: POST to
// http://{lcn|wcl}_host/api/server.php with action, time, key, account_id, and
// optional JSON in data.
func (c *Controller) postAccountAction(conn *model.Connection, action string, payload map[string]any) {
	account, ok := conn.Data["account"].(map[string]string)
	if !ok || account["type"] == "" || account["id"] == "" {
		return
	}
	accType := strings.ToLower(account["type"])
	host := c.Config.Raw[accType+"_host"]
	key := c.Config.Raw[accType+"_key"]
	if host == "" || key == "" {
		return
	}
	form := url.Values{}
	form.Set("action", action)
	form.Set("time", fmt.Sprintf("%d", time.Now().UTC().Unix()))
	form.Set("key", key)
	form.Set("account_id", account["id"])
	if payload != nil {
		if raw, err := json.Marshal(payload); err == nil {
			form.Set("data", string(raw))
		}
	}
	endpoint := "http://" + host + "/api/server.php"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "cossacks-server.net bot")
	req.Header.Set("X-Client-IP", conn.IP)
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("account action %s request failed: %v", action, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("account action %s bad response: %s", action, resp.Status)
	}
}

func (c *Controller) successEnter(conn *model.Connection, req *gsc.Stream, nick string) []gsc.Command {
	c.ensureRuntimeMaps()
	id, hasID := conn.Data["id"].(uint32)
	if hasID && id > 0 {
		// Perl parity: re-enter keeps id and leaves current room.
		c.leaveRoom(conn)
	} else {
		id = c.Store.NextPlayerID()
	}
	player := &state.Player{
		ID:          id,
		Nick:        nick,
		ConnectedAt: conn.Ctime,
	}
	if acc, ok := conn.Data["account"].(map[string]string); ok {
		player.Account = map[string]any{
			"type":    acc["type"],
			"login":   acc["login"],
			"id":      acc["id"],
			"profile": acc["profile"],
		}
	}
	c.Store.UpsertPlayer(player)
	conn.Data["id"] = id
	conn.Data["nick"] = nick
	c.mu.Lock()
	c.playerConns[id] = conn
	c.mu.Unlock()
	return []gsc.Command{{Name: "LW_show", Args: []string{
		loadShowBody(req.Ver, "ok_enter.tmpl", map[string]string{
			"nick":        nick,
			"id":          fmt.Sprintf("%d", id),
			"chat_server": c.Config.ChatServer,
			"window_size": windowSize(conn),
			"ver":         strconv.Itoa(int(req.Ver)),
		}),
	}}}
}

func (c *Controller) renderEnter(req *gsc.Stream, loginType, errText, loggedIn, nick, id string) []gsc.Command {
	vars := map[string]string{
		"type":          loginType,
		"error":         errText,
		"logged_in":     loggedIn,
		"nick":          nick,
		"id":            id,
		"chat_server":   c.Config.ChatServer,
		"table_timeout": strconv.Itoa(c.Config.TableTimeout),
		"ver":           strconv.Itoa(int(req.Ver)),
	}
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "enter.tmpl", vars)}}}
}

func parseOpenParams(params string) map[string]string {
	out := map[string]string{}
	for len(params) > 0 {
		eq := strings.IndexByte(params, '=')
		if eq <= 0 {
			break
		}
		key := params[:eq]
		rest := params[eq+1:]
		next := -1
		for i := 0; i < len(rest)-1; i++ {
			if rest[i] == '^' {
				j := i + 1
				for j < len(rest) && ((rest[j] >= 'A' && rest[j] <= 'Z') || (rest[j] >= 'a' && rest[j] <= 'z') || (rest[j] >= '0' && rest[j] <= '9') || rest[j] == '_') {
					j++
				}
				if j < len(rest) && rest[j] == '=' {
					next = i
					break
				}
			}
		}
		if next == -1 {
			out[key] = rest
			break
		}
		out[key] = rest[:next]
		params = rest[next+1:]
	}
	return out
}

func windowSize(conn *model.Connection) string {
	h, ok := conn.Data["height"].(int)
	if !ok {
		return "small"
	}
	if h > 366 {
		return "large"
	}
	return "small"
}

func sanitizeAccountNick(nick string) string {
	var b strings.Builder
	for _, r := range nick {
		if r == '[' || r == ']' || r == '_' || r == '-' ||
			(r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "player"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "_" + out
	}
	return out
}

func (c *Controller) refreshAlive(conn *model.Connection) {
	conn.Data["alive_at"] = time.Now().UTC()
	id, ok := conn.Data["id"].(uint32)
	if !ok || id == 0 {
		return
	}
	c.armAliveTimer(id)
}

func (c *Controller) handleStats(conn *model.Connection, args []string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.handleStatsLocked(conn, args)
}

func (c *Controller) handleStatsLocked(conn *model.Connection, args []string) {
	c.refreshAlive(conn)
	if len(args) < 2 {
		return
	}
	roomID, err := parseUint32Arg(args[1])
	if err != nil {
		return
	}
	room := c.Store.RoomsByID[roomID]
	if room == nil {
		return
	}
	userID, ok := conn.Data["id"].(uint32)
	if !ok || userID == 0 {
		return
	}
	player := room.Players[userID]
	if player == nil {
		return
	}
	stat, ok := decodeRawStat([]byte(args[0]))
	if !ok || stat.PlayerID != userID {
		return
	}
	if room.TimeTick < stat.Time {
		room.TimeTick = stat.Time
	}
	if player.TimeTick > stat.Time {
		log.Printf("player.time > stat.time")
		return
	}
	if player.StatCycle.Peasants == 0 && player.StatCycle.Units == 0 && player.StatCycle.Scores == 0 {
		player.StatCycle = state.PlayerStatCycle{}
	}
	old := oldPlayerStatForUpdate(player, stat)
	if player.Stat == nil && stat.Scores == 0 && stat.Population == 0 {
		for _, started := range room.StartedUsers {
			if started != nil && started.ID != player.ID && started.Theam == player.Theam {
				player.Zombie = true
				player.Color = started.Color
				break
			}
		}
	}
	if stat.Peasants < old.Peasants-player.StatCycle.Peasants*0x10000 {
		player.StatCycle.Peasants++
	}
	stat.Peasants += player.StatCycle.Peasants * 0x10000
	if stat.Units < old.Units-player.StatCycle.Units*0x10000 {
		player.StatCycle.Units++
	}
	stat.Units += player.StatCycle.Units * 0x10000

	scoresChange := int64(stat.Scores) - int64(old.Scores)
	if absI64(scoresChange) > 0x7FFF {
		if scoresChange > 0 {
			player.StatCycle.Scores--
		} else {
			player.StatCycle.Scores++
		}
	}
	stat.RealScores = player.StatCycle.Scores*0x10000 + int64(stat.Scores)
	stat.Population2 = stat.Units + stat.Peasants

	interval := stat.Time - player.TimeTick
	if interval == 0 {
		interval = 1
	}
	intervalF := float64(interval)
	stat.ChangeGold = (float64(diffU32(stat.Gold, old.Gold)) / intervalF) * 25 / 2
	stat.ChangeIron = (float64(diffU32(stat.Iron, old.Iron)) / intervalF) * 25 / 2
	stat.ChangeCoal = (float64(diffU32(stat.Coal, old.Coal)) / intervalF) * 25 / 2

	intervals := map[string]uint32{
		"wood":        60 * 25,
		"stone":       60 * 25,
		"food":        120 * 25,
		"peasants":    600,
		"units":       1000,
		"population2": 1000,
	}
	coefs := map[string]float64{
		"wood":        25.0 / 2.0,
		"stone":       25.0 / 2.0,
		"food":        25.0 / 2.0,
		"peasants":    200,
		"units":       50,
		"population2": 50,
	}
	if player.StatHistory == nil {
		player.StatHistory = map[string][]state.StatHistoryPoint{}
	}
	if player.StatSum == nil {
		player.StatSum = map[string]float64{}
	}
	updateRollingChange := func(name string, cur, prev uint32) float64 {
		key := "change_" + name
		change := float64(diffU32(cur, prev))
		player.StatHistory[key] = append(player.StatHistory[key], state.StatHistoryPoint{
			Change:   change,
			Time:     stat.Time,
			Interval: interval,
		})
		player.StatSum["sum_"+name] += change
		cutoff := stat.Time - intervals[name]
		for len(player.StatHistory[key]) > 0 && player.StatHistory[key][0].Time < cutoff {
			player.StatSum["sum_"+name] -= player.StatHistory[key][0].Change
			player.StatHistory[key] = player.StatHistory[key][1:]
		}
		if len(player.StatHistory[key]) == 0 {
			return 0
		}
		first := player.StatHistory[key][0]
		denom := float64(stat.Time - (first.Time - first.Interval))
		if denom <= 0 {
			denom = 1
		}
		return player.StatSum["sum_"+name] / denom * coefs[name]
	}
	stat.ChangeWood = updateRollingChange("wood", stat.Wood, old.Wood)
	stat.ChangeFood = updateRollingChange("food", stat.Food, old.Food)
	stat.ChangeStone = updateRollingChange("stone", stat.Stone, old.Stone)
	stat.ChangePeas = updateRollingChange("peasants", stat.Peasants, old.Peasants)
	stat.ChangeUnits = updateRollingChange("units", stat.Units, old.Units)
	stat.ChangePop2 = updateRollingChange("population2", stat.Population2, old.Population2)

	casualityChange := int64(stat.Population2-old.Population2) - int64(stat.Population-old.Population)
	stat.Casuality = old.Casuality + casualityChange

	player.TimeTick = stat.Time
	player.Stat = stat
}

func (c *Controller) handleEndgame(conn *model.Connection, args []string) {
	ev, ok := c.parseEndgame(conn, args)
	if !ok {
		return
	}
	log.Printf("send game result: %s:%d %s in %sgame %d%s", ev.Nick, ev.PlayerID, ev.Result, ev.Own, ev.GameID, ev.Title)
}

type endgameEvent struct {
	GameID   int
	PlayerID uint32
	Result   string
	Nick     string
	Own      string
	Title    string
}

func (c *Controller) parseEndgame(conn *model.Connection, args []string) (endgameEvent, bool) {
	if len(args) < 3 {
		return endgameEvent{}, false
	}
	gameID, _ := parseIntArg(args[0])
	playerID, _ := parseIntArg(args[1])
	result, _ := parseIntArg(args[2])
	playerU32 := uint32(int32(playerID))
	id, _ := conn.Data["id"].(uint32)
	nick := "."
	if pl := c.Store.Players[playerU32]; pl != nil {
		nick = pl.Nick
	}
	resultStr := fmt.Sprintf("?%d?", result)
	switch result {
	case 1:
		resultStr = "loose"
	case 2:
		resultStr = "win"
	case 5:
		resultStr = "disconnect"
	}
	room := c.Store.RoomsByID[uint32(gameID)]
	own := ""
	if room != nil && id != 0 && room.HostID == id {
		own = "his "
	}
	title := ""
	if room != nil {
		title = " " + room.Title
	}
	return endgameEvent{
		GameID:   gameID,
		PlayerID: playerU32,
		Result:   resultStr,
		Nick:     nick,
		Own:      own,
		Title:    title,
	}, true
}

func (c *Controller) handleProxy(conn *model.Connection, args []string) {
	// Perl: proxy(ip,port,key) validates proxy_key and rewrites endpoint.
	if len(args) < 3 {
		conn.Closed = true
		return
	}
	ipArg, portArg, keyArg := args[0], args[1], args[2]
	validKey := c.Config.Raw["proxy_key"]
	if validKey == "" || keyArg != validKey {
		conn.Closed = true
		return
	}
	ip := net.ParseIP(strings.TrimSpace(ipArg)).To4()
	if ip == nil {
		conn.Closed = true
		return
	}
	port, err := strconv.Atoi(strings.TrimSpace(portArg))
	if err != nil || port <= 0 || port >= 0xFFFF {
		conn.Closed = true
		return
	}
	conn.IP = ip.String()
	conn.IntIP = binary.LittleEndian.Uint32(ip)
	conn.Port = port
}

func (c *Controller) OnDisconnect(conn *model.Connection) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	// Perl ConnectionController::_close leaves room and removes player.
	id, ok := conn.Data["id"].(uint32)
	if !ok {
		return
	}
	c.clearAliveTimer(id)
	c.mu.Lock()
	delete(c.playerConns, id)
	c.mu.Unlock()
	c.leaveRoomByID(id)
	delete(c.Store.Players, id)
}

func (c *Controller) regNewRoom(conn *model.Connection, req *gsc.Stream, p map[string]string) []gsc.Command {
	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		})}}}
	}
	idAny := conn.Data["id"]
	nickAny := conn.Data["nick"]
	if idAny == nil || nickAny == nil {
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "alert_dgl.tmpl", map[string]string{
				"header": "Error",
				"text":   "Your was disconnected from the server. Enter again.",
			}),
		}}}
	}
	playerID := idAny.(uint32)
	rawTitle := p["VE_TITLE"]
	if strings.TrimSpace(rawTitle) == "" || strings.ContainsAny(rawTitle, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F\x7F") {
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(req.Ver, "confirm_dgl.tmpl", map[string]string{
				"header":  "Error",
				"text":    "Illegal title!\nPress Edit button to check title",
				"ok_text": "Edit",
				"height":  "180",
				"command": "GW|open&new_room_dgl.dcml&ASTATE=<%ASTATE>",
			}),
		}}}
	}
	title := rawTitle
	if len(title) > 60 {
		title = title[:60]
	}
	title = strings.TrimSpace(title)
	c.leaveRoomByID(playerID)
	maxPlayers := 8
	if v, err := strconv.Atoi(p["VE_MAX_PL"]); err == nil {
		maxPlayers = v + 2
	}
	level := 0
	if v, err := strconv.Atoi(p["VE_LEVEL"]); err == nil {
		level = v
	}
	levelLabel := "For all"
	switch level {
	case 1:
		levelLabel = "Easy"
	case 2:
		levelLabel = "Normal"
	case 3:
		levelLabel = "Hard"
	}
	lockMark := ""
	if p["VE_PASSWD"] != "" {
		lockMark = "#"
	}
	roomID := c.Store.NextRoomID()
	row := []string{fmt.Sprintf("%d", roomID), lockMark, title, nickAny.(string)}
	if isAC(req.Ver) {
		row = append(row, p["VE_TYPE"])
	}
	row = append(row,
		levelLabel,
		fmt.Sprintf("1/%d", maxPlayers),
		fmt.Sprintf("%d", req.Ver),
		fmt.Sprintf("%d", conn.IntIP),
		fmt.Sprintf("0%X", uint32(0xFFFFFFFF-roomID)),
	)
	hostName := os.Getenv("HOST_NAME")
	if hostName == "" {
		hostName = conn.IP
	}
	room := &state.Room{
		ID:           roomID,
		Title:        title,
		HostID:       playerID,
		HostAddr:     conn.IP,
		HostAddrInt:  conn.IntIP,
		Ver:          req.Ver,
		Level:        level,
		Password:     p["VE_PASSWD"],
		MaxPlayers:   maxPlayers,
		PlayersCount: 1,
		Players:      map[uint32]*state.Player{playerID: c.Store.Players[playerID]},
		PlayersTime:  map[uint32]time.Time{playerID: time.Now()},
		Row:          row,
		CtlSum:       state.RoomControlSum(row),
		Ctime:        time.Now(),
	}
	c.Store.RoomsByID[roomID] = room
	c.Store.RoomsByPID[playerID] = room
	c.Store.RoomsBySum[room.CtlSum] = room
	c.armAliveTimer(playerID)
	gameID := fmt.Sprintf("%d", roomID)
	if p["VE_TYPE"] != "" {
		gameID = "HB" + gameID
	}
	log.Printf(
		"createGame stun endpoint debug: player_id=%d hole_host=%q hole_port=%d hole_int=%d",
		playerID, hostName, c.Config.HolePort, c.Config.HoleInterval,
	)
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(req.Ver, "reg_new_room.tmpl", map[string]string{
		"player_id": fmt.Sprintf("%d", playerID),
		"hole_port": strconv.Itoa(c.Config.HolePort),
		"hole_host": hostName,
		"hole_int":  strconv.Itoa(c.Config.HoleInterval),
		"id":        gameID,
		"name":      title,
		"max_pl":    fmt.Sprintf("%d", maxPlayers),
	})}}}
}

func (c *Controller) joinGame(ctx context.Context, conn *model.Connection, req *gsc.Stream, p map[string]string) []gsc.Command {
	reqVer := uint8(2)
	if req != nil {
		reqVer = req.Ver
	}
	if p["ASTATE"] == "" || p["ASTATE"] == "0" {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(reqVer, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not create or join room!\nYou are already participate in some room\nPlease disconnect from that room first to create a new one",
		})}}}
	}
	playerID, ok := conn.Data["id"].(uint32)
	if !ok || strings.TrimSpace(fmt.Sprintf("%v", conn.Data["nick"])) == "" {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(reqVer, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "Your was disconnected from the server. Enter again.",
		})}}}
	}
	if okRID, _ := regexp.MatchString(`^\d+$`, strings.TrimSpace(p["VE_RID"])); !okRID {
		// Perl returns empty dialog pair on invalid VE_RID format.
		return []gsc.Command{{Name: "LW_show", Args: []string{"<NGDLG>\n<NGDLG>"}}}
	}
	rid, _ := strconv.ParseUint(p["VE_RID"], 10, 32)
	room := c.Store.RoomsByID[uint32(rid)]
	if room == nil {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(reqVer, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not join this room!\nThe room is closed",
		})}}}
	}
	if room.Started {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not join this room!\nThe game has already started",
		})}}}
	}
	if room.PlayersCount >= room.MaxPlayers {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "You can not join this room!\nThe room is full",
		})}}}
	}
	if room.Password != "" && p["VE_PASSWD"] != room.Password {
		return []gsc.Command{{Name: "LW_show", Args: []string{
			loadShowBody(room.Ver, "confirm_password_dgl.tmpl", map[string]string{
				"id": fmt.Sprintf("%d", room.ID),
			}),
		}}}
	}
	c.leaveRoomByID(playerID)
	room.Players[playerID] = c.Store.Players[playerID]
	room.PlayersTime[playerID] = time.Now()
	room.PlayersCount++
	delete(c.Store.RoomsBySum, room.CtlSum)
	room.Row = setRoomPlayersColumn(room.Row, room.PlayersCount, room.MaxPlayers)
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsByPID[playerID] = room
	c.Store.RoomsBySum[room.CtlSum] = room

	ip := room.HostAddr
	port := 0
	redisHit := false
	if c.Redis != nil {
		if raw, err := c.Redis.Get(ctx, strconv.Itoa(int(room.HostID))); err == nil {
			var remote struct {
				Host string `json:"host"`
				Port int    `json:"port"`
			}
			if json.Unmarshal([]byte(raw), &remote) == nil {
				ip = remote.Host
				port = remote.Port
				redisHit = true
			}
		}
	}
	log.Printf(
		"join endpoint debug: conn_id=%d room_id=%d host_id=%d redis_hit=%t ip=%q port=%d",
		conn.ID, room.ID, room.HostID, redisHit, ip, port,
	)
	_ = ip
	_ = port
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, "join_room.tmpl", map[string]string{
		"id":     fmt.Sprintf("%d", room.ID),
		"max_pl": fmt.Sprintf("%d", room.MaxPlayers),
		"name":   room.Title,
		"ip":     ip,
		"port":   fmt.Sprintf("%d", port),
	})}}}
}

func (c *Controller) roomInfo(conn *model.Connection, req *gsc.Stream, p map[string]string) []gsc.Command {
	reqVer := uint8(2)
	if req != nil {
		reqVer = req.Ver
	}
	veRID := p["VE_RID"]
	if okRID, _ := regexp.MatchString(`^\d+$`, veRID); !okRID {
		// Perl behavior for invalid VE_RID.
		return []gsc.Command{{Name: "LW_show", Args: []string{"<NGDLG>\n<NGDLG>"}}}
	}
	rid, _ := strconv.ParseUint(veRID, 10, 32)
	room := c.Store.RoomsByID[uint32(rid)]
	if room == nil {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(reqVer, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "The room is closed",
		})}}}
	}
	backto := ""
	if p["BACKTO"] == "user_details" {
		if id, ok := conn.Data["id"].(uint32); ok {
			backto = fmt.Sprintf("open&user_details.dcml&ID=%d", id)
		}
	}
	if room.Started && (toBool(conn.Data["dev"]) || toBool(c.Config.Raw["show_started_room_info"])) {
		tpl := "started_room_info.tmpl"
		if p["part"] == "statcols" {
			tpl = "started_room_info/statcols.tmpl"
		}
		page := normalizePage(p["page"])
		res := normalizeRes(p["res"])
		activePlayers, exitedPlayers := startedPlayerNames(room)
		exited := "0"
		if len(exitedPlayers) > 0 {
			exited = "1"
		}
		roomTicks := room.TimeTick
		if roomTicks == 0 && !room.StartedAt.IsZero() {
			roomTicks = uint32(time.Since(room.StartedAt).Seconds() * 25)
		}
		vars := map[string]string{
			"room_id":              fmt.Sprintf("%d", room.ID),
			"room_name":            room.Title,
			"room_players":         fmt.Sprintf("%d/%d", room.PlayersCount, room.MaxPlayers),
			"room_players_start":   fmt.Sprintf("%d", room.StartPlayers),
			"room_host":            room.HostAddr,
			"room_ctime":           strconv.FormatInt(room.Ctime.Unix(), 10),
			"room_started":         strconv.FormatBool(room.Started),
			"room.id":              fmt.Sprintf("%d", room.ID),
			"room.title":           room.Title,
			"room.time":            fmt.Sprintf("%d", roomTicks),
			"room.level":           fmt.Sprintf("%d", room.Level),
			"room.map":             room.Map,
			"room_max_pl":          fmt.Sprintf("%d", room.MaxPlayers),
			"room_pl_count":        fmt.Sprintf("%d", room.PlayersCount),
			"room_time":            roomTimeInterval(room),
			"active_players":       strings.Join(activePlayers, ", "),
			"exited_players":       strings.Join(exitedPlayers, ", "),
			"has_exited_players":   exited,
			"backto":               backto,
			"page":                 page,
			"res":                  res,
		}
		mergeRoomDottedVars(conn, room, vars)
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, tpl, vars)}}}
	}
	roomVars := map[string]string{
		"room_id":       fmt.Sprintf("%d", room.ID),
		"room_name":     room.Title,
		"room_players":  fmt.Sprintf("%d/%d", room.PlayersCount, room.MaxPlayers),
		"room_host":     room.HostAddr,
		"room_ctime":    strconv.FormatInt(room.Ctime.Unix(), 10),
		"room_started":  strconv.FormatBool(room.Started),
		"room_max_pl":   fmt.Sprintf("%d", room.MaxPlayers),
		"room_pl_count": fmt.Sprintf("%d", room.PlayersCount),
		"room_time":     roomTimeInterval(room),
		"backto":        backto,
	}
	mergeRoomDottedVars(conn, room, roomVars)
	return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, "room_info_dgl.tmpl", roomVars)}}}
}

func normalizePage(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "1"
	}
	if ok, _ := regexp.MatchString(`^\d+$`, s); !ok {
		return "1"
	}
	if s != "1" && s != "2" && s != "3" {
		return "1"
	}
	return s
}

func normalizeRes(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "0"
	}
	if ok, _ := regexp.MatchString(`^\d+$`, s); !ok {
		return "0"
	}
	return s
}


// joinPlayer implements Open join_pl_cmd (Open.pm). VE_PLAYER is a player id;
// the room is rooms_by_player{VE_PLAYER} (Store.RoomsByPID), then the same as
// room_info_dgl with VE_RID => that room, or an error if started, or no response
// if there is no room, or an empty response if the caller is already in a room.
func (c *Controller) joinPlayer(conn *model.Connection, p map[string]string) []gsc.Command {
	// Perl: push_empty, return if connection->data->{id} && rooms_by_player{that id}
	if id, ok := conn.Data["id"].(uint32); ok && c.Store.RoomsByPID[id] != nil {
		return []gsc.Command{}
	}
	vePlayer := strings.TrimSpace(p["VE_PLAYER"])
	if okID, _ := regexp.MatchString(`^\d+$`, vePlayer); !okID {
		return nil
	}
	joinedID, _ := strconv.ParseUint(vePlayer, 10, 32)
	room := c.Store.RoomsByPID[uint32(joinedID)]
	if room == nil {
		return nil
	}
	if room.Started {
		return []gsc.Command{{Name: "LW_show", Args: []string{loadShowBody(room.Ver, "alert_dgl.tmpl", map[string]string{
			"header": "Error",
			"text":   "Game alredy started",
		})}}}
	}
	return c.roomInfo(conn, &gsc.Stream{Ver: room.Ver}, map[string]string{
		"VE_RID": fmt.Sprintf("%d", room.ID),
	})
}

func (c *Controller) leaveRoom(conn *model.Connection) {
	playerID, ok := conn.Data["id"].(uint32)
	if !ok {
		return
	}
	c.leaveRoomByID(playerID)
}

func (c *Controller) leaveRoomByID(playerID uint32) {
	c.clearAliveTimer(playerID)
	room := c.Store.RoomsByPID[playerID]
	if room == nil {
		return
	}
	oldCtl := room.CtlSum
	delete(c.Store.RoomsByPID, playerID)
	room.PlayersCount--
	if room.Started {
		// Perl parity: for started rooms player remains in room list and is marked exited.
		if pl := room.Players[playerID]; pl != nil {
			pl.ExitedAt = time.Now().UTC()
		}
	} else {
		delete(room.Players, playerID)
		delete(room.PlayersTime, playerID)
		room.Row = setRoomPlayersColumn(room.Row, room.PlayersCount, room.MaxPlayers)
	}
	delete(c.Store.RoomsBySum, oldCtl)
	room.CtlSum = state.RoomControlSum(room.Row)
	shouldDelete := false
	if room.Started {
		// Perl parity: started room is removed only when all players are gone.
		shouldDelete = room.PlayersCount <= 0
	} else {
		// Perl parity: pre-start room is removed when host leaves.
		shouldDelete = room.HostID == playerID || room.PlayersCount <= 0
	}
	if shouldDelete {
		delete(c.Store.RoomsByID, room.ID)
		delete(c.Store.RoomsBySum, room.CtlSum)
		return
	}
	c.Store.RoomsBySum[room.CtlSum] = room
}

func (c *Controller) handleGETTBL(conn *model.Connection, req *gsc.Stream, args []string) []gsc.Command {
	if len(args) < 3 {
		return []gsc.Command{}
	}
	name := strings.Trim(args[0], "\x00")
	requestedSums := unpackU32LE([]byte(args[2]))
	requested := map[uint32]bool{}
	for _, sum := range requestedSums {
		requested[sum] = true
	}
	hideStarted := !toBool(conn.Data["dev"]) && !c.Config.ShowStartedRooms
	dtbl := make([]uint32, 0)
	tblRows := make([][]string, 0)

	for _, sum := range requestedSums {
		room, ok := c.Store.RoomsBySum[sum]
		if !ok || (hideStarted && room.Started) {
			dtbl = append(dtbl, sum)
		}
	}
	for _, room := range c.Store.RoomsByID {
		if hideStarted && room.Started {
			continue
		}
		if strings.EqualFold(name, "ROOMS_V"+strconv.Itoa(int(req.Ver))) && !requested[room.CtlSum] {
			tblRows = append(tblRows, room.Row)
		}
	}
	dtblBin := packU32LE(dtbl)
	tblArgs := []string{name + "\x00", fmt.Sprintf("%d", len(tblRows))}
	for _, row := range tblRows {
		tblArgs = append(tblArgs, row...)
	}
	return []gsc.Command{
		{Name: "LW_dtbl", Args: []string{name + "\x00", string(dtblBin)}},
		{Name: "LW_tbl", Args: tblArgs},
	}
}

func (c *Controller) handleStart(conn *model.Connection, req *gsc.Stream, args []string) []gsc.Command {
	playerID, ok := conn.Data["id"].(uint32)
	if !ok {
		return nil
	}
	room := c.Store.RoomsByPID[playerID]
	if room == nil {
		return nil
	}
	_ = req
	sav := ""
	mapName := ""
	if len(args) > 0 {
		sav = strings.TrimRight(args[0], "\x00")
	}
	if len(args) > 1 {
		mapName = strings.TrimRight(args[1], "\x00")
	}
	delete(c.Store.RoomsBySum, room.CtlSum)
	room.Started = true
	room.StartedAt = time.Now().UTC()
	room.StartPlayers = room.PlayersCount
	room.Map = mapName
	room.SaveFrom = 0
	if m := regexp.MustCompile(`^sav:\[(\d+)\]$`).FindStringSubmatch(sav); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			room.SaveFrom = v
		}
	}
	// Perl flips started marker in row payload and recalculates checksum.
	if len(room.Row) > 1 {
		room.Row[1] = "\x7f0018"
	}
	if len(room.Row) > 0 && len(room.Row[len(room.Row)-1]) > 0 {
		last := []byte(room.Row[len(room.Row)-1])
		last[0] = '1'
		room.Row[len(room.Row)-1] = string(last)
	}
	room.CtlSum = state.RoomControlSum(room.Row)
	c.Store.RoomsBySum[room.CtlSum] = room
	for pid := range room.Players {
		c.armAliveTimer(pid)
	}
	if playerID == room.HostID && len(args) >= 3 {
		count, err := parseIntArg(args[2])
		if err == nil && count > 0 {
			room.StartedUsers = nil
			list := args[3:]
			for i := 0; i+3 < len(list) && len(room.StartedUsers) < count; i += 4 {
				pid, err1 := parseIntArg(list[i])
				nation, err2 := parseIntArg(list[i+1])
				theam, err3 := parseIntArg(list[i+2])
				color, err4 := parseIntArg(list[i+3])
				if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
					continue
				}
				pl := room.Players[uint32(int32(pid))]
				if pl == nil {
					continue
				}
				pl.Nation = uint32(nation)
				pl.Theam = uint32(theam)
				pl.Color = uint32(color)
				room.StartedUsers = append(room.StartedUsers, pl)
			}
		}
	}
	if payload := c.buildStartAccountPayload(room, args, sav, mapName); payload != nil {
		c.postAccountAction(conn, "start", payload)
	}
	return nil
}

func (c *Controller) buildStartAccountPayload(room *state.Room, args []string, sav, mapName string) map[string]any {
	if room == nil || len(args) < 3 {
		return nil
	}
	playersCount, err := parseIntArg(args[2])
	if err != nil || playersCount <= 0 {
		return nil
	}
	payload := map[string]any{
		"id":            int(room.ID),
		"title":         room.Title,
		"max_players":   room.MaxPlayers,
		"players_count": room.PlayersCount,
		"level":         room.Level,
		"ctime":         room.Ctime.Unix(),
		"map":           mapName,
	}
	if m := regexp.MustCompile(`^sav:\[(\d+)\]$`).FindStringSubmatch(sav); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			payload["save_from"] = v
		}
	}
	players := make([]map[string]any, 0, playersCount)
	raw := args[3:]
	for i := 0; i+3 < len(raw) && len(players) < playersCount; i += 4 {
		playerIDInt, err1 := parseIntArg(raw[i])
		nation, err2 := parseIntArg(raw[i+1])
		theam, err3 := parseIntArg(raw[i+2])
		color, err4 := parseIntArg(raw[i+3])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		playerID := uint32(int32(playerIDInt))
		postPlayer := map[string]any{
			"id":     int(playerID),
			"nation": nation,
			"theam":  theam,
			"color":  color,
		}
		if pl := c.Store.Players[playerID]; pl != nil {
			postPlayer["nick"] = pl.Nick
			postPlayer["connected_at"] = pl.ConnectedAt.Unix()
			if pl.Account != nil {
				account := map[string]any{}
				if t, ok := pl.Account["type"].(string); ok {
					account["type"] = t
				}
				if p, ok := pl.Account["profile"].(string); ok && p != "" {
					account["profile"] = p
				}
				switch v := pl.Account["id"].(type) {
				case string:
					account["id"] = v
				case fmt.Stringer:
					account["id"] = v.String()
				case nil:
				default:
					account["id"] = fmt.Sprintf("%v", v)
				}
				if len(account) > 0 {
					postPlayer["account"] = account
				}
			}
		} else {
			postPlayer["lost"] = true
		}
		players = append(players, postPlayer)
	}
	payload["players"] = players
	return payload
}

func unpackU32LE(b []byte) []uint32 {
	if len(b) < 4 {
		return nil
	}
	n := len(b) / 4
	out := make([]uint32, 0, n)
	for i := 0; i+4 <= len(b); i += 4 {
		out = append(out, binary.LittleEndian.Uint32(b[i:i+4]))
	}
	return out
}

func packU32LE(v []uint32) []byte {
	b := make([]byte, len(v)*4)
	for i := range v {
		binary.LittleEndian.PutUint32(b[i*4:(i+1)*4], v[i])
	}
	return b
}

func toBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t != "" && t != "0" && strings.ToLower(t) != "false"
	default:
		return false
	}
}

func setRoomPlayersColumn(row []string, playersCount, maxPlayers int) []string {
	if len(row) == 0 {
		return row
	}
	out := append([]string(nil), row...)
	idx := len(out) - 4
	if idx < 0 {
		idx = len(out) - 1
	}
	out[idx] = fmt.Sprintf("%d/%d", playersCount, maxPlayers)
	return out
}

func startedPlayerNames(room *state.Room) (active []string, exited []string) {
	// Keep order by join time for parity with Perl players_time.nsort.
	type pair struct {
		id uint32
		t  time.Time
	}
	ordered := make([]pair, 0, len(room.PlayersTime))
	for id, t := range room.PlayersTime {
		ordered = append(ordered, pair{id: id, t: t})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].t.Before(ordered[j].t) })
	for _, it := range ordered {
		pl := room.Players[it.id]
		if pl == nil {
			continue
		}
		if !pl.ExitedAt.IsZero() {
			exited = append(exited, pl.Nick)
		} else {
			active = append(active, pl.Nick)
		}
	}
	return active, exited
}

// timeIntervalFromElapsedSec matches legacy Perl Open::_time_interval(age in seconds).
func timeIntervalFromElapsedSec(secs int) string {
	if secs < 0 {
		secs = 0
	}
	t := secs
	d := t / 86400
	t %= 86400
	h := t / 3600
	t %= 3600
	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if d > 0 {
		return strings.Join(parts, " ")
	}
	m := t / 60
	t %= 60
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if h > 0 || m >= 10 {
		return strings.Join(parts, " ")
	}
	if t > 0 {
		parts = append(parts, fmt.Sprintf("%ds", t))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return "0s"
}

func roomTimeInterval(room *state.Room) string {
	base := room.Ctime
	if room.Started && !room.StartedAt.IsZero() {
		base = room.StartedAt
	}
	secs := int(time.Since(base).Seconds())
	return timeIntervalFromElapsedSec(secs)
}

func mergeRoomDottedVars(conn *model.Connection, room *state.Room, vars map[string]string) {
	if vars == nil {
		return
	}
	// room_info_dgl.tmpl uses room.*; TT "room" object is approximated with dotted keys.
	if toBool(conn.Data["dev"]) {
		vars["h.connection.data.dev"] = "1"
	} else {
		vars["h.connection.data.dev"] = ""
	}
	vars["room.id"] = fmt.Sprintf("%d", room.ID)
	vars["room.title"] = room.Title
	vars["room.host_id"] = fmt.Sprintf("%d", room.HostID)
	vars["room.host_addr_int"] = fmt.Sprintf("%d", room.HostAddrInt)
	vars["room.level"] = fmt.Sprintf("%d", room.Level)
	if room.Started {
		vars["room.started"] = "1"
	} else {
		vars["room.started"] = "0"
	}
	vars["room.start_players_count"] = strconv.Itoa(room.StartPlayers)
	vars["room.players_count"] = strconv.Itoa(room.PlayersCount)
	vars["room.max_players"] = strconv.Itoa(room.MaxPlayers)
	vars["room.map"] = room.Map
	// No AI field in state.Room yet; keep a stable negative for the template.
	vars["room.ai"] = "0"
	vars["room.passwd"] = room.Password
	if p := room.Players[room.HostID]; p != nil {
		vars[fmt.Sprintf("room.players.%d.nick", room.HostID)] = p.Nick
	}
	vars["room.ctime"] = room.Ctime.UTC().Format("2006-01-02 15:04:05 UTC")
}

func decodeRawStat(raw []byte) (*state.PlayerStat, bool) {
	if len(raw) < 42 {
		return nil, false
	}
	s := &state.PlayerStat{}
	s.Time = binary.LittleEndian.Uint32(raw[0:4])
	s.PC = raw[4]
	s.PlayerID = binary.LittleEndian.Uint32(raw[5:9])
	s.Status = raw[9]
	s.Scores = uint32(binary.LittleEndian.Uint16(raw[10:12]))
	s.Population = uint32(binary.LittleEndian.Uint16(raw[12:14]))
	s.Wood = binary.LittleEndian.Uint32(raw[14:18])
	s.Gold = binary.LittleEndian.Uint32(raw[18:22])
	s.Stone = binary.LittleEndian.Uint32(raw[22:26])
	s.Food = binary.LittleEndian.Uint32(raw[26:30])
	s.Iron = binary.LittleEndian.Uint32(raw[30:34])
	s.Coal = binary.LittleEndian.Uint32(raw[34:38])
	s.Peasants = uint32(binary.LittleEndian.Uint16(raw[38:40]))
	s.Units = uint32(binary.LittleEndian.Uint16(raw[40:42]))
	return s, true
}

func oldPlayerStatForUpdate(player *state.Player, cur *state.PlayerStat) *state.PlayerStat {
	if player.Stat != nil {
		prev := *player.Stat
		return &prev
	}
	return &state.PlayerStat{
		Time:        0,
		PC:          cur.PC,
		PlayerID:    cur.PlayerID,
		Status:      cur.Status,
		Scores:      cur.Scores,
		Population:  cur.Population,
		Wood:        cur.Wood,
		Gold:        cur.Gold,
		Stone:       cur.Stone,
		Food:        cur.Food,
		Iron:        cur.Iron,
		Coal:        cur.Coal,
		Peasants:    cur.Peasants,
		Units:       cur.Units,
		Population2: cur.Units + cur.Peasants,
		Casuality:   0,
	}
}

func parseUint32Arg(v string) (uint32, error) {
	i, err := parseIntArg(v)
	if err != nil {
		return 0, err
	}
	return uint32(i), nil
}

func parseIntArg(v string) (int, error) {
	s := strings.TrimRight(strings.TrimSpace(v), "\x00")
	re := regexp.MustCompile(`-?\d+`)
	m := re.FindString(s)
	if m == "" {
		return 0, fmt.Errorf("no int")
	}
	return strconv.Atoi(m)
}

func absI64(v int64) int64 {
	return int64(math.Abs(float64(v)))
}

func diffU32(cur, prev uint32) int64 {
	return int64(cur) - int64(prev)
}

func (c *Controller) armAliveTimer(playerID uint32) {
	c.ensureRuntimeMaps()
	c.mu.Lock()
	if t := c.aliveTimers[playerID]; t != nil {
		t.Stop()
	}
	c.aliveTimers[playerID] = time.AfterFunc(c.aliveTTL, func() {
		c.notAlive(playerID)
	})
	c.mu.Unlock()
}

func (c *Controller) clearAliveTimer(playerID uint32) {
	c.ensureRuntimeMaps()
	c.mu.Lock()
	if t := c.aliveTimers[playerID]; t != nil {
		t.Stop()
	}
	delete(c.aliveTimers, playerID)
	c.mu.Unlock()
}

func (c *Controller) notAlive(playerID uint32) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	c.leaveRoomByID(playerID)
	c.mu.Lock()
	delete(c.playerConns, playerID)
	c.mu.Unlock()
}
