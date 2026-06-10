package middleware

import "sync"

// AccessLogStore は直近のアクセスログを保持するスレッドセーフなリングバッファ。
// 永続化はしない（プロセス再起動で消える）。管理画面の「最近のリクエスト」表示用で、
// DB スキーマを増やさずに「行が多い一覧」を提供するためのもの。
type AccessLogStore struct {
	mu       sync.RWMutex
	entries  []AccessLogEntry
	capacity int
}

// NewAccessLogStore は指定容量のリングバッファを作る。capacity<=0 のときは 1000。
func NewAccessLogStore(capacity int) *AccessLogStore {
	if capacity <= 0 {
		capacity = 1000
	}
	return &AccessLogStore{capacity: capacity, entries: make([]AccessLogEntry, 0, capacity)}
}

// Add は1エントリを追加する。容量を超えたら古いものから捨てる。
func (s *AccessLogStore) Add(e AccessLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	if len(s.entries) > s.capacity {
		s.entries = s.entries[len(s.entries)-s.capacity:]
	}
}

// Recent は新しい順に最大 limit 件返す。limit<=0 のときは全件。
func (s *AccessLogStore) Recent(limit int) []AccessLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.entries)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]AccessLogEntry, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, s.entries[i])
	}
	return out
}
