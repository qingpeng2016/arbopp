package leagueopt

import "sync"

// SaveStats 本轮同步见到的去重场次/盘口（与 CloseStaleAfterSync 的 key 一致）。
type SaveStats struct {
	Fixtures int
	Outcomes int
}

func (a SaveStats) Add(b SaveStats) SaveStats {
	return SaveStats{Fixtures: a.Fixtures + b.Fixtures, Outcomes: a.Outcomes + b.Outcomes}
}

func (s SaveStats) Empty() bool {
	return s.Fixtures == 0 && s.Outcomes == 0
}

// Seen 多个联赛协程共用的去重与计数。
type Seen struct {
	mu       sync.Mutex
	deduped  map[string]struct{}
	fixtures map[string]struct{}
	outcomes map[string]struct{}
	saved    int
	err      error
}

func NewSeen() *Seen {
	return &Seen{
		deduped:  make(map[string]struct{}),
		fixtures: make(map[string]struct{}),
		outcomes: make(map[string]struct{}),
	}
}

func (s *Seen) Claim(key string) bool {
	if s == nil || key == "" {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.deduped[key]; ok {
		return false
	}
	s.deduped[key] = struct{}{}
	return true
}

func (s *Seen) Keep(fixtureID, outcomeID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if fixtureID != "" {
		s.fixtures[fixtureID] = struct{}{}
	}
	if outcomeID != "" {
		s.outcomes[outcomeID] = struct{}{}
	}
}

func (s *Seen) AddSaved(n int) int {
	if s == nil {
		return n
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved += n
	return s.saved
}

func (s *Seen) Fail(err error) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

func (s *Seen) Saved() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved
}

func (s *Seen) SaveStats() SaveStats {
	if s == nil {
		return SaveStats{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return SaveStats{
		Fixtures: len(s.fixtures),
		Outcomes: len(s.outcomes),
	}
}

func (s *Seen) Err() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *Seen) Fixtures() map[string]struct{} {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fixtures
}

func (s *Seen) Outcomes() map[string]struct{} {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outcomes
}
