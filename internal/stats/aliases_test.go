package stats

import (
	"atad-project/internal/gitlog"
	"os"
	"path/filepath"
	"testing"
)

func TestAliasRules(t *testing.T) {
	file := filepath.Join(t.TempDir(), "aliases.json")
	err := os.WriteFile(file, []byte(`[{"name":"Jo","email":"jo@test","aliases":["old@test","Jo Smith"]}]`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	aliases, err := LoadAliases(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []gitlog.Commit{{Email: "OLD@test"}, {Name: "Jo Smith"}, {Email: "jo@test"}} {
		aliases.Apply(&c)
		if c.Name != "Jo" || c.Email != "jo@test" {
			t.Fatalf("identity = %+v", c)
		}
	}
}
