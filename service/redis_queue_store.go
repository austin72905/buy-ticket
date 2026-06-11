package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	infraredis "github.com/austin72905/go-infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

type RedisQueueStore struct {
	client       *goredis.Client
	releaseLimit int
}

func NewRedisQueueStore(client *infraredis.Client, releaseLimit int) *RedisQueueStore {
	if releaseLimit <= 0 {
		releaseLimit = 1
	}

	return &RedisQueueStore{
		client:       client.Raw(),
		releaseLimit: releaseLimit,
	}
}

func (s *RedisQueueStore) Join(ctx context.Context, input JoinQueueInput, now time.Time) (*QueueStatusSnapshot, error) {
	key := queueUserEventKey(input.EventID, input.UserID)
	if queueToken, err := s.client.Get(ctx, redisUserEventKey(key)).Result(); err == nil {
		snapshot, getErr := s.loadSnapshot(ctx, queueToken)
		if getErr == nil && isQueueStatusActive(*snapshot, now) {
			return nil, ErrUserAlreadyJoinedQueue
		}
	}

	queueToken := generateQueueToken(now)
	eventQueueKey := redisEventQueueKey(input.EventID)
	sequence, err := s.client.Incr(ctx, redisEventSeqKey(input.EventID)).Result()
	if err != nil {
		return nil, err
	}
	if err := s.client.ZAdd(ctx, eventQueueKey, goredis.Z{
		Score:  float64(sequence),
		Member: queueToken,
	}).Err(); err != nil {
		return nil, err
	}
	if err := s.client.SAdd(ctx, redisActiveEventsKey(), strconv.FormatInt(input.EventID, 10)).Err(); err != nil {
		return nil, err
	}

	rank, err := s.client.ZRank(ctx, eventQueueKey, queueToken).Result()
	if err != nil {
		return nil, err
	}

	snapshot := QueueStatusSnapshot{
		QueueToken:           queueToken,
		QueueSequence:        sequence,
		Status:               QueueStatusWaiting,
		EventID:              input.EventID,
		UserID:               input.UserID,
		QueuePosition:        rank + 1,
		AheadCount:           rank,
		EstimatedWaitSeconds: rank * 30,
		JoinedAt:             now,
		ExpiredAt:            now.Add(30 * time.Minute),
		UpdatedAt:            now,
	}

	readyCount, err := s.readyCount(ctx, input.EventID, now)
	if err != nil {
		return nil, err
	}
	if readyCount < s.releaseLimit {
		purchaseToken := generatePurchaseToken(now)
		expiresAt := now.Add(5 * time.Minute)
		snapshot.Status = QueueStatusReady
		snapshot.PurchaseToken = &purchaseToken
		snapshot.PurchaseTokenExpiresAt = &expiresAt
		if err := s.client.Set(ctx, redisPurchaseTokenKey(purchaseToken), queueToken, time.Until(expiresAt)).Err(); err != nil {
			return nil, err
		}
	}

	if err := s.saveSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}
	if err := s.client.Set(ctx, redisUserEventKey(key), queueToken, time.Until(snapshot.ExpiredAt)).Err(); err != nil {
		return nil, err
	}

	cloned := snapshot
	return &cloned, nil
}

func (s *RedisQueueStore) Get(ctx context.Context, queueToken string, now time.Time) (*QueueStatusSnapshot, error) {
	snapshot, err := s.loadSnapshot(ctx, queueToken)
	if err != nil {
		return nil, err
	}

	rank, err := s.client.ZRank(ctx, redisEventQueueKey(snapshot.EventID), queueToken).Result()
	if err == nil {
		snapshot.QueuePosition = rank + 1
		snapshot.AheadCount = rank
		snapshot.EstimatedWaitSeconds = rank * 30
	}

	return snapshot, nil
}

func (s *RedisQueueStore) ConsumePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error) {
	queueToken, err := s.client.Get(ctx, redisPurchaseTokenKey(purchaseToken)).Result()
	if err == goredis.Nil {
		return nil, ErrPurchaseTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	snapshot, err := s.loadSnapshot(ctx, queueToken)
	if err != nil {
		return nil, err
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

	original := *snapshot
	snapshot.PurchaseToken = nil
	snapshot.PurchaseTokenExpiresAt = nil
	snapshot.PurchaseTokenUsedAt = &now
	snapshot.UpdatedAt = now

	if err := s.client.Del(ctx, redisPurchaseTokenKey(purchaseToken)).Err(); err != nil {
		return nil, err
	}
	if err := s.client.Del(ctx, redisUserEventKey(queueUserEventKey(snapshot.EventID, snapshot.UserID))).Err(); err != nil {
		return nil, err
	}
	if err := s.client.ZRem(ctx, redisEventQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
		return nil, err
	}
	if err := s.saveSnapshot(ctx, *snapshot); err != nil {
		return nil, err
	}

	return &original, nil
}

func (s *RedisQueueStore) RestorePurchaseToken(ctx context.Context, snapshot QueueStatusSnapshot) error {
	if err := s.client.ZAdd(ctx, redisEventQueueKey(snapshot.EventID), goredis.Z{
		Score:  float64(snapshot.QueueSequence),
		Member: snapshot.QueueToken,
	}).Err(); err != nil {
		return err
	}
	if snapshot.PurchaseToken != nil && snapshot.PurchaseTokenExpiresAt != nil {
		if err := s.client.Set(ctx, redisPurchaseTokenKey(*snapshot.PurchaseToken), snapshot.QueueToken, time.Until(*snapshot.PurchaseTokenExpiresAt)).Err(); err != nil {
			return err
		}
	}
	if err := s.client.Set(ctx, redisUserEventKey(queueUserEventKey(snapshot.EventID, snapshot.UserID)), snapshot.QueueToken, time.Until(snapshot.ExpiredAt)).Err(); err != nil {
		return err
	}
	return s.saveSnapshot(ctx, snapshot)
}

func (s *RedisQueueStore) SaveSnapshot(ctx context.Context, snapshot QueueStatusSnapshot) error {
	return s.saveSnapshot(ctx, snapshot)
}

func (s *RedisQueueStore) PromoteReady(ctx context.Context, now time.Time) error {
	eventIDs, err := s.client.SMembers(ctx, redisActiveEventsKey()).Result()
	if err != nil {
		return err
	}

	for _, eventIDValue := range eventIDs {
		eventID, parseErr := strconv.ParseInt(eventIDValue, 10, 64)
		if parseErr != nil {
			continue
		}
		if promoteErr := s.promoteReady(ctx, eventID, now); promoteErr != nil {
			return promoteErr
		}
	}

	return nil
}

func (s *RedisQueueStore) CleanupExpiredPurchaseTokens(ctx context.Context, now time.Time) (int, error) {
	eventIDs, err := s.client.SMembers(ctx, redisActiveEventsKey()).Result()
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for _, eventIDValue := range eventIDs {
		eventID, parseErr := strconv.ParseInt(eventIDValue, 10, 64)
		if parseErr != nil {
			continue
		}

		count, cleanupErr := s.cleanupExpiredPurchaseTokensByEvent(ctx, eventID, now)
		if cleanupErr != nil {
			return cleaned, cleanupErr
		}
		cleaned += count
	}

	return cleaned, nil
}

func (s *RedisQueueStore) promoteReady(ctx context.Context, eventID int64, now time.Time) error {
	readyCount, err := s.readyCount(ctx, eventID, now)
	if err != nil {
		return err
	}
	if readyCount >= s.releaseLimit {
		return nil
	}

	tokens, err := s.client.ZRange(ctx, redisEventQueueKey(eventID), 0, -1).Result()
	if err != nil || len(tokens) == 0 {
		return err
	}

	for _, token := range tokens {
		if readyCount >= s.releaseLimit {
			return nil
		}

		snapshot, loadErr := s.loadSnapshot(ctx, token)
		if loadErr != nil || !isQueueStatusActive(*snapshot, now) || snapshot.Status == QueueStatusReady {
			continue
		}

		purchaseToken := generatePurchaseToken(now)
		expiresAt := now.Add(5 * time.Minute)
		snapshot.Status = QueueStatusReady
		snapshot.PurchaseToken = &purchaseToken
		snapshot.PurchaseTokenExpiresAt = &expiresAt
		snapshot.UpdatedAt = now

		if err := s.client.Set(ctx, redisPurchaseTokenKey(purchaseToken), snapshot.QueueToken, time.Until(expiresAt)).Err(); err != nil {
			return err
		}
		if err := s.saveSnapshot(ctx, *snapshot); err != nil {
			return err
		}
		readyCount++
	}

	return nil
}

func (s *RedisQueueStore) cleanupExpiredPurchaseTokensByEvent(ctx context.Context, eventID int64, now time.Time) (int, error) {
	tokens, err := s.client.ZRange(ctx, redisEventQueueKey(eventID), 0, -1).Result()
	if err != nil || len(tokens) == 0 {
		return 0, err
	}

	cleaned := 0
	for _, token := range tokens {
		snapshot, loadErr := s.loadSnapshot(ctx, token)
		if loadErr != nil {
			continue
		}
		if snapshot.Status != QueueStatusReady || snapshot.PurchaseToken == nil || snapshot.PurchaseTokenExpiresAt == nil {
			continue
		}
		if snapshot.PurchaseTokenExpiresAt.After(now) {
			continue
		}

		if err := s.client.Del(ctx, redisPurchaseTokenKey(*snapshot.PurchaseToken)).Err(); err != nil {
			return cleaned, err
		}
		if err := s.client.Del(ctx, redisUserEventKey(queueUserEventKey(snapshot.EventID, snapshot.UserID))).Err(); err != nil {
			return cleaned, err
		}
		if err := s.client.ZRem(ctx, redisEventQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
			return cleaned, err
		}

		snapshot.Status = QueueStatusExpired
		snapshot.PurchaseToken = nil
		snapshot.PurchaseTokenExpiresAt = nil
		snapshot.UpdatedAt = now
		if err := s.saveSnapshot(ctx, *snapshot); err != nil {
			return cleaned, err
		}
		cleaned++
	}

	return cleaned, nil
}

func (s *RedisQueueStore) loadSnapshot(ctx context.Context, queueToken string) (*QueueStatusSnapshot, error) {
	value, err := s.client.Get(ctx, redisQueueTokenKey(queueToken)).Result()
	if err == goredis.Nil {
		return nil, ErrQueueTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	var snapshot QueueStatusSnapshot
	if err := json.Unmarshal([]byte(value), &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *RedisQueueStore) saveSnapshot(ctx context.Context, snapshot QueueStatusSnapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, redisQueueTokenKey(snapshot.QueueToken), payload, time.Until(snapshot.ExpiredAt)).Err()
}

func redisEventQueueKey(eventID int64) string {
	return "queue:event:" + strconv.FormatInt(eventID, 10)
}

func redisEventSeqKey(eventID int64) string {
	return "queue:event:" + strconv.FormatInt(eventID, 10) + ":seq"
}

func redisQueueTokenKey(queueToken string) string {
	return "queue:token:" + queueToken
}

func redisPurchaseTokenKey(purchaseToken string) string {
	return "queue:purchase:" + purchaseToken
}

func redisUserEventKey(key string) string {
	return "queue:user:event:" + key
}

func redisActiveEventsKey() string {
	return "queue:events:active"
}

func (s *RedisQueueStore) readyCount(ctx context.Context, eventID int64, now time.Time) (int, error) {
	tokens, err := s.client.ZRange(ctx, redisEventQueueKey(eventID), 0, -1).Result()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, token := range tokens {
		snapshot, loadErr := s.loadSnapshot(ctx, token)
		if loadErr != nil || !isQueueStatusActive(*snapshot, now) {
			continue
		}
		if snapshot.Status == QueueStatusReady && snapshot.PurchaseToken != nil {
			count++
		}
	}

	return count, nil
}
