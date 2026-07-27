package netbird

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProfileConfigPreservesUnknownAndHidesSecrets(t *testing.T) {
	root := t.TempDir()
	s := NewProfileConfigStore(root)
	if e := os.MkdirAll(s.root, 0700); e != nil {
		t.Fatal(e)
	}
	raw := map[string]any{"name": "default", "futureField": "kept", "setupKeyConfigured": true, "presharedKeyConfigured": true}
	b, _ := json.Marshal(raw)
	if e := os.WriteFile(filepath.Join(s.root, "default.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	v, e := s.Get("default")
	if e != nil || !v.SetupKeyConfigured || !v.PresharedKeyConfigured {
		t.Fatal("sensitive state not read")
	}
	v.Name = "work"
	if e = s.Put("default", v); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(filepath.Join(s.root, "default.json"))
	var result map[string]any
	_ = json.Unmarshal(after, &result)
	if result["futureField"] != "kept" {
		t.Fatal("unknown field lost")
	}
}
