package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicIfChanged(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	changed, err := writeFileAtomicIfChanged(p, []byte("a"), 0o644)
	if err != nil || !changed {
		t.Fatalf("首次写应 changed=true：changed=%v err=%v", changed, err)
	}
	changed, err = writeFileAtomicIfChanged(p, []byte("a"), 0o644)
	if err != nil || changed {
		t.Fatalf("同内容应 changed=false：changed=%v err=%v", changed, err)
	}
	changed, err = writeFileAtomicIfChanged(p, []byte("b"), 0o644)
	if err != nil || !changed {
		t.Fatalf("不同内容应 changed=true：changed=%v err=%v", changed, err)
	}
	if b, _ := os.ReadFile(p); string(b) != "b" {
		t.Fatalf("内容=%q，期望 b", b)
	}
}

func TestRemoveFileReport(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	if removeFileReport(p) {
		t.Fatalf("不存在的文件应报 false")
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !removeFileReport(p) {
		t.Fatalf("存在的文件应报 true")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("文件应已删除")
	}
}
