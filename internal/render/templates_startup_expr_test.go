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

package render

import "testing"

func TestEvalExprStartupGGCupArithmetic(t *testing.T) {
	vars := map[string]string{
		"gg_cup.prize_fund":    "1000.5",
		"gg_cup.players_count": "12",
		"gg_cup.wo_info":       "0",
		"gg_cup.id":            "5",
		"gg_cup.started":       "0",
		"gg_cup":               "1",
	}
	// len("12")=2, 500+7*2=514 — matches share/cs/startup.cml x offsets.
	if got := evalExpr("500 + 7 * gg_cup.players_count.length", vars); got != "514" {
		t.Fatalf("add/mul+length: got %q, want 514", got)
	}
	if got := evalExpr("POSIX.floor(gg_cup.prize_fund).length", vars); got != "4" {
		t.Fatalf("floor().length: got %q, want 4 (digits in 1000)", got)
	}
	// 7*4=28
	if got := evalExpr("7 * POSIX.floor(gg_cup.prize_fund).length", vars); got != "28" {
		t.Fatalf("7*floor().length: got %q, want 28", got)
	}
}

func TestEvalExprAddMulRespectsParensInFloor(t *testing.T) {
	vars := map[string]string{"gg_cup.prize_fund": "3.2"}
	// * only inside floor(), not a top-level multiply on bare vars.
	if got := evalExpr("POSIX.floor(gg_cup.prize_fund)", vars); got != "3" {
		t.Fatalf("POSIX.floor: got %q", got)
	}
}
