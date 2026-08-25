package dedup

import "sync"

type Ring struct {
	mu    sync.Mutex
	seen  map[uint64]bool
	order []uint64
	limit int
}

func NewRing(limit int) *Ring {
	if limit < 1 {
		limit = 1
	}
	return &Ring{seen: make(map[uint64]bool), limit: limit}
}

func (r *Ring) CheckAdd(key uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen[key] {
		return false
	}
	r.seen[key] = true
	r.order = append(r.order, key)
	for len(r.order) > r.limit {
		old := r.order[0]
		r.order = r.order[1:]
		delete(r.seen, old)
	}
	return true
}
