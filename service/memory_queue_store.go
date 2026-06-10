package service

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type MemoryQueueStore struct {
	mu             sync.RWMutex
	releaseLimit   int
	queueStatuses  map[string]QueueStatusSnapshot
	queueUserEvent map[string]string
	purchaseTokens map[string]string
	eventQueues    map[int64][]string
}

func NewMemoryQueueStore(releaseLimit int) *MemoryQueueStore {
	if releaseLimit <= 0 {
		releaseLimit = 1
	}

	return &MemoryQueueStore{
		releaseLimit:   releaseLimit,
		queueStatuses:  map[string]QueueStatusSnapshot{},
		queueUserEvent: map[string]string{},
		purchaseTokens: map[string]string{},
		eventQueues:    map[int64][]string{},
	}
}

func (s *MemoryQueueStore) Join(ctx context.Context, input JoinQueueInput, now time.Time) (*QueueStatusSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := queueUserEventKey(input.EventID, input.UserID)
	if token, ok := s.queueUserEvent[key]; ok {
		if snapshot, exists := s.queueStatuses[token]; exists && isQueueStatusActive(snapshot, now) {
			return nil, ErrUserAlreadyJoinedQueue
		}
	}

	queueToken := generateQueueToken(now)
	s.eventQueues[input.EventID] = append(s.eventQueues[input.EventID], queueToken)
	position := len(s.eventQueues[input.EventID])

	snapshot := QueueStatusSnapshot{
		QueueToken:           queueToken,
		QueueSequence:        int64(position),
		Status:               QueueStatusWaiting,
		EventID:              input.EventID,
		UserID:               input.UserID,
		QueuePosition:        int64(position),
		AheadCount:           int64(position - 1),
		EstimatedWaitSeconds: int64(position-1) * 30,
		JoinedAt:             now,
		ExpiredAt:            now.Add(30 * time.Minute),
		UpdatedAt:            now,
	}

	if s.readyCountLocked(input.EventID, now) < s.releaseLimit {
		purchaseToken := generatePurchaseToken(now)
		expiresAt := now.Add(5 * time.Minute)
		snapshot.Status = QueueStatusReady
		snapshot.PurchaseToken = &purchaseToken
		snapshot.PurchaseTokenExpiresAt = &expiresAt
		s.purchaseTokens[purchaseToken] = queueToken
	}

	s.queueStatuses[queueToken] = snapshot
	s.queueUserEvent[key] = queueToken

	cloned := snapshot
	return &cloned, nil
}

func (s *MemoryQueueStore) Get(ctx context.Context, queueToken string, now time.Time) (*QueueStatusSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, ok := s.queueStatuses[queueToken]
	if !ok {
		return nil, ErrQueueTokenNotFound
	}

	snapshot, ok = s.queueStatuses[queueToken]
	if !ok {
		return nil, ErrQueueTokenNotFound
	}

	position := indexOfToken(s.eventQueues[snapshot.EventID], queueToken)
	if position >= 0 {
		snapshot.QueuePosition = int64(position + 1)
		snapshot.AheadCount = int64(position)
		snapshot.EstimatedWaitSeconds = int64(position) * 30
		s.queueStatuses[queueToken] = snapshot
	}

	cloned := snapshot
	return &cloned, nil
}

func (s *MemoryQueueStore) ConsumePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queueToken, ok := s.purchaseTokens[purchaseToken]
	if !ok {
		return nil, ErrPurchaseTokenNotFound
	}

	snapshot, ok := s.queueStatuses[queueToken]
	if !ok {
		return nil, ErrPurchaseTokenNotFound
	}
	if snapshot.PurchaseToken == nil {
		return nil, ErrPurchaseTokenUsed
	}
	if snapshot.EventID != eventID || snapshot.UserID != userID {
		return nil, ErrPurchaseTokenMismatch
	}
	if snapshot.PurchaseTokenExpiresAt == nil || snapshot.PurchaseTokenExpiresAt.Before(now) {
		return nil, ErrPurchaseTokenExpired
	}

	original := snapshot
	snapshot.PurchaseToken = nil
	snapshot.PurchaseTokenExpiresAt = nil
	snapshot.UpdatedAt = now
	snapshot.PurchaseTokenUsedAt = &now
	s.queueStatuses[queueToken] = snapshot
	delete(s.purchaseTokens, purchaseToken)
	delete(s.queueUserEvent, queueUserEventKey(snapshot.EventID, snapshot.UserID))
	s.eventQueues[snapshot.EventID] = removeToken(s.eventQueues[snapshot.EventID], queueToken)

	return &original, nil
}

func (s *MemoryQueueStore) RestorePurchaseToken(ctx context.Context, snapshot QueueStatusSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queueStatuses[snapshot.QueueToken] = snapshot
	s.queueUserEvent[queueUserEventKey(snapshot.EventID, snapshot.UserID)] = snapshot.QueueToken
	if snapshot.PurchaseToken != nil {
		s.purchaseTokens[*snapshot.PurchaseToken] = snapshot.QueueToken
	}
	if indexOfToken(s.eventQueues[snapshot.EventID], snapshot.QueueToken) < 0 {
		s.eventQueues[snapshot.EventID] = append([]string{snapshot.QueueToken}, s.eventQueues[snapshot.EventID]...)
	}
	return nil
}

func (s *MemoryQueueStore) SaveSnapshot(ctx context.Context, snapshot QueueStatusSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queueStatuses[snapshot.QueueToken] = snapshot
	s.queueUserEvent[queueUserEventKey(snapshot.EventID, snapshot.UserID)] = snapshot.QueueToken
	if snapshot.PurchaseToken != nil {
		s.purchaseTokens[*snapshot.PurchaseToken] = snapshot.QueueToken
	}
	if indexOfToken(s.eventQueues[snapshot.EventID], snapshot.QueueToken) < 0 && isQueueStatusActive(snapshot, time.Now()) {
		s.eventQueues[snapshot.EventID] = append(s.eventQueues[snapshot.EventID], snapshot.QueueToken)
	}
	return nil
}

func (s *MemoryQueueStore) PromoteReady(ctx context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for eventID := range s.eventQueues {
		s.promoteReadyLocked(eventID, now)
	}

	return nil
}

func (s *MemoryQueueStore) promoteReadyLocked(eventID int64, now time.Time) {
	readyCount := s.readyCountLocked(eventID, now)
	if readyCount >= s.releaseLimit {
		return
	}

	queue := s.eventQueues[eventID]
	for _, token := range queue {
		if readyCount >= s.releaseLimit {
			return
		}

		snapshot, ok := s.queueStatuses[token]
		if !ok || !isQueueStatusActive(snapshot, now) || snapshot.Status == QueueStatusReady {
			continue
		}

		purchaseToken := generatePurchaseToken(now)
		expiresAt := now.Add(5 * time.Minute)
		snapshot.Status = QueueStatusReady
		snapshot.PurchaseToken = &purchaseToken
		snapshot.PurchaseTokenExpiresAt = &expiresAt
		snapshot.UpdatedAt = now
		s.queueStatuses[token] = snapshot
		s.purchaseTokens[purchaseToken] = token
		readyCount++
	}
}

func (s *MemoryQueueStore) readyCountLocked(eventID int64, now time.Time) int {
	count := 0
	for _, token := range s.eventQueues[eventID] {
		snapshot, ok := s.queueStatuses[token]
		if !ok || !isQueueStatusActive(snapshot, now) {
			continue
		}
		if snapshot.Status == QueueStatusReady && snapshot.PurchaseToken != nil {
			count++
		}
	}
	return count
}

func queueUserEventKey(eventID, userID int64) string {
	return strconv.FormatInt(eventID, 10) + ":" + strconv.FormatInt(userID, 10)
}

func isQueueStatusActive(snapshot QueueStatusSnapshot, now time.Time) bool {
	if snapshot.ExpiredAt.Before(now) {
		return false
	}

	return snapshot.Status == QueueStatusWaiting || snapshot.Status == QueueStatusReady
}

func generateQueueToken(now time.Time) string {
	return "qt_" + now.Format("20060102150405.000000000")
}

func generatePurchaseToken(now time.Time) string {
	return "pt_" + now.Format("20060102150405.000000000")
}

func indexOfToken(tokens []string, target string) int {
	for index, token := range tokens {
		if token == target {
			return index
		}
	}
	return -1
}

func removeToken(tokens []string, target string) []string {
	index := indexOfToken(tokens, target)
	if index < 0 {
		return tokens
	}
	return append(tokens[:index], tokens[index+1:]...)
}
