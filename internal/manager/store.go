package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

func (a *App) withStoreLock(fn func() error) error {
	if err := ensureDir(a.cfg.CoreDir, 0o700); err != nil {
		return err
	}
	return withFileLock(a.cfg.StoreLockPath(), fn)
}

func (a *App) loadStore() (*Store, error) {
	if err := ensureDir(a.cfg.CoreDir, 0o700); err != nil {
		return nil, err
	}
	path := a.cfg.StorePath()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newStore(), nil
	}
	if err != nil {
		return nil, err
	}
	st := newStore()
	if len(b) > 0 {
		if err := json.Unmarshal(b, st); err != nil {
			if recovered, ok := a.loadStoreBackup(); ok {
				fmt.Printf("警告：状态文件损坏，已从备份恢复：%s\n", path)
				return recovered, nil
			}
			return nil, fmt.Errorf("状态文件损坏且无法从备份恢复（%s）：%w", path, err)
		}
	}
	normalizeStore(st)
	return st, nil
}

func (a *App) loadStoreBackup() (*Store, bool) {
	b, err := os.ReadFile(a.cfg.StoreBackupPath())
	if err != nil || len(b) == 0 {
		return nil, false
	}
	st := newStore()
	if err := json.Unmarshal(b, st); err != nil {
		return nil, false
	}
	normalizeStore(st)
	return st, true
}

func normalizeStore(st *Store) {
	if st.SceneNodes == nil {
		st.SceneNodes = map[Scene]string{}
	}
	if st.SceneEnabled == nil {
		st.SceneEnabled = map[Scene]bool{}
	}
	if st.SpeedResults == nil {
		st.SpeedResults = map[string]SpeedResult{}
	}
}

func (a *App) saveStore(st *Store) error {
	normalizeStore(st)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data := append(b, '\n')
	if err := writeFileAtomic(a.cfg.StorePath(), data, 0o600); err != nil {
		return err
	}
	// 尽力保存一份备份，供状态文件损坏时恢复；备份失败不影响主流程。
	_ = writeFileAtomic(a.cfg.StoreBackupPath(), data, 0o600)
	return nil
}

func (st *Store) findNode(id string) *Node {
	for i := range st.Nodes {
		if st.Nodes[i].ID == id {
			return &st.Nodes[i]
		}
	}
	return nil
}

func (st *Store) findNodeByURL(raw string) *Node {
	for i := range st.Nodes {
		if st.Nodes[i].RawURL == raw {
			return &st.Nodes[i]
		}
	}
	return nil
}

func (st *Store) firstNodeID() string {
	if len(st.Nodes) == 0 {
		return ""
	}
	return st.Nodes[0].ID
}

func (st *Store) selectedNodeID(scene Scene) string {
	if id := st.SceneNodes[scene]; id != "" && st.findNode(id) != nil {
		return id
	}
	if st.DefaultNodeID != "" && st.findNode(st.DefaultNodeID) != nil {
		return st.DefaultNodeID
	}
	return st.firstNodeID()
}

// nodeIDCounter 保证同一进程内连续生成的节点 ID 唯一，避免在订阅导入等紧循环里
// 因 time.Now() 分辨率不足或时钟回拨而产生重复 ID。
var nodeIDCounter uint64

func newNodeID() string {
	seq := atomic.AddUint64(&nodeIDCounter, 1)
	return fmt.Sprintf("node-%s-%s", strconv.FormatInt(time.Now().UnixNano(), 36), strconv.FormatUint(seq, 36))
}
