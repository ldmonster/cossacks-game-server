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

// GG Cup "thanks" dialog body builder. Pure rendering — no I/O, no
// controller state.

package render

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GGCupThanksBoxHeight gg_cup_thanks_dgl height logic.
func GGCupThanksBoxHeight(supporterCount int) int {
	const rows = 17

	if supporterCount <= 9 {
		return 280
	}

	if supporterCount > rows {
		return 55 + (rows+1)*25
	}

	return 55 + supporterCount*25
}

// GGCupThanksBody emits a dialog shaped like share/cs/gg_cup_thanks_dgl
// for the supplied supporter list. Each supporter
// map is expected to provide `nick` (string), `amount` (numeric), and
// `url` (string) keys; missing values render as empty strings.
func GGCupThanksBody(supporters []map[string]any) string {
	var b strings.Builder

	n := len(supporters)
	h := GGCupThanksBoxHeight(n)

	fmt.Fprintf(&b, "<NGDLG>\n")
	b.WriteString("#exec(LW_lockbox&%LBX)\n")
	b.WriteString("#exec(LW_enb&0&%RMLST)\n")
	fmt.Fprintf(&b, "#ebox[%%B](x:215,y:10,w:320,h:%d)\n", h)
	b.WriteString("#pan[%MPN](%B[x:0,y:0,w:100%,h:100%],8)\n")
	b.WriteString("#font(WF,WF,WF)\n")
	b.WriteString("#ctxt[%TIT](%B[x:0,y:6,w:100%,h:30],{},\"Thanks for\")\n\n")
	// when loop.index==rows on an extra
	// supporter, emit "and more..." and LAST. With 0-based indexing:
	// render supporter i for i < 17; if n > 17 and i == 17, emit overflow
	// line.
	const rows = 17

	yoff := 43

	for i := 0; i < n; i++ {
		if n > rows && i == rows {
			b.WriteString("#font(YF,YF,YF)\n")
			fmt.Fprintf(&b, "#txt(%%B[x:20,y:%d,w:100%%,h:25],{},\"and more...\")\n", yoff+3)

			break
		}

		s := supporters[i]
		nick := CMLSafe(fmt.Sprintf("%v", s["nick"]))
		amt := SupporterAmountString(s["amount"])
		url := CMLSafe(fmt.Sprintf("%v", s["url"]))

		b.WriteString("#font(YF,YF,YF)\n")
		fmt.Fprintf(&b, "#txt(%%B[x:20,y:%d,w:100%%,h:25],{},\"%s\")\n", yoff+3, nick)
		b.WriteString("#font(WF,WF,WF)\n")
		fmt.Fprintf(&b, "#rtxt(%%B[x:100%%-204,y:%d,w:100,h:25],{},\"%s RUB \")\n", yoff+3, amt)
		fmt.Fprintf(&b, "#btn(%%B[x:230,y:%d,w:72,h:25],{GW|url&%s},\"profile\")\n", yoff, url)
		yoff += 25
	}

	b.WriteString("\n#font(YF,WF,RF)\n")
	b.WriteString(
		"#sbtn[%B_RGST](%B[xc:50%,y:100%+8,w:160,h:24],{LW_file&Internet/Cash/cancel.cml},\"Ok\")\n",
	)
	b.WriteString("<NGDLG>\n")

	return b.String()
}

// SupporterAmountString normalises a heterogeneous JSON-decoded
// supporter `amount` value (float64, int, int64, json.Number, or
// arbitrary stringy fallback) into the integer-shaped representation
// expected by the GG Cup template (RUB column).
func SupporterAmountString(v any) string {
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
