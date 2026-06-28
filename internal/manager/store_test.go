package manager

import (
	"os"
	"testing"
)

func testApp(t *testing.T) *App {
	t.Helper()
	cfg := DefaultConfig()
	cfg.CoreDir = t.TempDir()
	return NewApp(cfg)
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	a := testApp(t)
	st := newStore()
	st.Nodes = append(st.Nodes, Node{ID: "node-1", Name: "n1", Protocol: "vless", RawURL: "vless://x@h:443?security=tls"})
	st.DefaultNodeID = "node-1"
	st.SceneEnabled[SceneGlobal] = true
	if err := a.saveStore(st); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	got, err := a.loadStore()
	if err != nil {
		t.Fatalf("loadStore: %v", err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "node-1" || !got.SceneEnabled[SceneGlobal] {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestStoreCorruptMainRecoversFromBackup(t *testing.T) {
	a := testApp(t)
	st := newStore()
	st.DefaultNodeID = "node-keep"
	st.Nodes = append(st.Nodes, Node{ID: "node-keep", Name: "keep", Protocol: "ss", RawURL: "ss://m:p@h:1"})
	if err := a.saveStore(st); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	if err := os.WriteFile(a.cfg.StorePath(), []byte("{ not valid json"), 0o600); err != nil {
		t.Fatalf("corrupt main: %v", err)
	}
	got, err := a.loadStore()
	if err != nil {
		t.Fatalf("loadStore should recover from backup, got error: %v", err)
	}
	if got.DefaultNodeID != "node-keep" || len(got.Nodes) != 1 {
		t.Fatalf("expected backup recovery, got %+v", got)
	}
}

func TestStoreCorruptMainAndBackupFails(t *testing.T) {
	a := testApp(t)
	if err := a.saveStore(newStore()); err != nil {
		t.Fatalf("saveStore: %v", err)
	}
	_ = os.WriteFile(a.cfg.StorePath(), []byte("{bad"), 0o600)
	_ = os.WriteFile(a.cfg.StoreBackupPath(), []byte("{bad"), 0o600)
	if _, err := a.loadStore(); err == nil {
		t.Fatalf("expected error when both main and backup are corrupt")
	}
}

func TestLoadStoreMissingReturnsEmpty(t *testing.T) {
	a := testApp(t)
	got, err := a.loadStore()
	if err != nil {
		t.Fatalf("loadStore on empty dir: %v", err)
	}
	if got == nil || len(got.Nodes) != 0 || got.SceneEnabled == nil {
		t.Fatalf("expected empty initialized store, got %+v", got)
	}
}
