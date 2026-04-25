package commands

import (
	"flag"
	"os"
	"testing"
)

// golden selects regeneration of checked-in fixtures. Run:
//
//	go test ./internal/server/commands -golden
//
// Output is the same bytes the tests already compute for assertions (LW command
// metadata JSON and template fullbody .golden files).
var golden = flag.Bool("golden", false, "regenerate golden fixtures (testdata/golden + testdata/template_fullbody)")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
