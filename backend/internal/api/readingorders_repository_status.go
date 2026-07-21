package api

import "context"

func (s *cblRepositorySyncer) setCurrentFile(file string) {
	s.update(func(status *CBLRepositorySyncStatus) { status.CurrentFile = file })
}

func (s *cblRepositorySyncer) increment(kind string) {
	s.update(func(status *CBLRepositorySyncStatus) {
		switch kind {
		case "imported":
			status.Imported++
		case "updated":
			status.Updated++
		default:
			status.Unchanged++
		}
	})
}

func (s *cblRepositorySyncer) recordFailure(err error) {
	if err == nil {
		return
	}
	s.update(func(status *CBLRepositorySyncStatus) {
		status.Failed++
		status.LastError = err.Error()
	})
}

func (s *cblRepositorySyncer) update(change func(*CBLRepositorySyncStatus)) {
	s.mu.Lock()
	change(&s.status)
	s.mu.Unlock()
	s.broadcast()
}

func (s *cblRepositorySyncer) snapshot(ctx context.Context) CBLRepositorySyncStatus {
	settings, _ := loadCBLRepositorySyncSettings(ctx, s.db)
	s.mu.Lock()
	status := s.status
	s.mu.Unlock()
	status.Settings = settings
	return status
}

func (s *cblRepositorySyncer) subscribe(ctx context.Context) (<-chan CBLRepositorySyncStatus, func()) {
	s.mu.Lock()
	s.nextSubscriberID++
	id := s.nextSubscriberID
	updates := make(chan CBLRepositorySyncStatus, 16)
	s.subscribers[id] = updates
	s.mu.Unlock()
	updates <- s.snapshot(ctx)
	return updates, func() {
		s.mu.Lock()
		if current, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(current)
		}
		s.mu.Unlock()
	}
}

func (s *cblRepositorySyncer) broadcast() {
	status := s.snapshot(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, updates := range s.subscribers {
		select {
		case updates <- status:
		default:
			select {
			case <-updates:
			default:
			}
			select {
			case updates <- status:
			default:
			}
		}
	}
}
