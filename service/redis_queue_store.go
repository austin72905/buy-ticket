package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	infraredis "github.com/austin72905/go-infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

var promoteReadyScript = goredis.NewScript(`
local readyCount = redis.call("ZCARD", KEYS[2])
local releaseLimit = tonumber(ARGV[1])
local slots = releaseLimit - readyCount
if slots <= 0 then
	return {}
end

local entries = redis.call("ZPOPMIN", KEYS[1], slots)
local promoted = {}
local tokenIndex = 7

for index = 1, #entries, 2 do
	local queueToken = entries[index]
	local score = entries[index + 1]
	local snapshotKey = ARGV[5] .. queueToken
	local snapshotPayload = redis.call("GET", snapshotKey)

	if snapshotPayload then
		local snapshot = cjson.decode(snapshotPayload)
		if snapshot["Status"] == 1 and (snapshot["ExpiredAt"] == nil or snapshot["ExpiredAt"] >= ARGV[2]) then
			local purchaseToken = ARGV[tokenIndex]
			tokenIndex = tokenIndex + 1

			snapshot["Status"] = 2
			snapshot["PurchaseToken"] = purchaseToken
			snapshot["PurchaseTokenExpiresAt"] = ARGV[3]
			snapshot["UpdatedAt"] = ARGV[2]

			local snapshotTTL = redis.call("PTTL", snapshotKey)
			if snapshotTTL > 0 then
				redis.call("PSETEX", snapshotKey, snapshotTTL, cjson.encode(snapshot))
			else
				redis.call("SET", snapshotKey, cjson.encode(snapshot))
			end
			redis.call("SET", ARGV[6] .. purchaseToken, queueToken, "PX", ARGV[4])
			redis.call("ZADD", KEYS[2], score, queueToken)
			table.insert(promoted, queueToken)
		end
	end
end

return promoted
`)

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
	sequence, err := s.client.Incr(ctx, redisEventSeqKey(input.EventID)).Result()
	if err != nil {
		return nil, err
	}
	if err := s.client.ZAdd(ctx, redisWaitingQueueKey(input.EventID), goredis.Z{
		Score:  float64(sequence),
		Member: queueToken,
	}).Err(); err != nil {
		return nil, err
	}
	if err := s.client.SAdd(ctx, redisActiveEventsKey(), strconv.FormatInt(input.EventID, 10)).Err(); err != nil {
		return nil, err
	}

	rank, err := s.client.ZRank(ctx, redisWaitingQueueKey(input.EventID), queueToken).Result()
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

	rank, err := s.client.ZRank(ctx, redisWaitingQueueKey(snapshot.EventID), queueToken).Result()
	if err == nil {
		snapshot.QueuePosition = rank + 1
		snapshot.AheadCount = rank
		snapshot.EstimatedWaitSeconds = rank * 30
		return snapshot, nil
	}

	if _, err := s.client.ZRank(ctx, redisReadyQueueKey(snapshot.EventID), queueToken).Result(); err == nil {
		snapshot.QueuePosition = 0
		snapshot.AheadCount = 0
		snapshot.EstimatedWaitSeconds = 0
	}

	return snapshot, nil
}

func (s *RedisQueueStore) ConsumePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error) {
	snapshot, err := s.validatePurchaseToken(ctx, purchaseToken, eventID, userID, now)
	if err != nil {
		return nil, err
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
	if err := s.client.ZRem(ctx, redisReadyQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
		return nil, err
	}
	if err := s.saveSnapshot(ctx, *snapshot); err != nil {
		return nil, err
	}
	if err := s.removeActiveEventIfEmpty(ctx, snapshot.EventID); err != nil {
		return nil, err
	}

	return &original, nil
}

func (s *RedisQueueStore) ValidatePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error) {
	snapshot, err := s.validatePurchaseToken(ctx, purchaseToken, eventID, userID, now)
	if err != nil {
		return nil, err
	}

	cloned := *snapshot
	return &cloned, nil
}

func (s *RedisQueueStore) validatePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error) {
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

	return snapshot, nil
}

func (s *RedisQueueStore) RestorePurchaseToken(ctx context.Context, snapshot QueueStatusSnapshot) error {
	if snapshot.PurchaseToken != nil && snapshot.PurchaseTokenExpiresAt != nil {
		if err := s.client.Set(ctx, redisPurchaseTokenKey(*snapshot.PurchaseToken), snapshot.QueueToken, time.Until(*snapshot.PurchaseTokenExpiresAt)).Err(); err != nil {
			return err
		}
		if err := s.client.ZAdd(ctx, redisReadyQueueKey(snapshot.EventID), goredis.Z{
			Score:  float64(snapshot.QueueSequence),
			Member: snapshot.QueueToken,
		}).Err(); err != nil {
			return err
		}
	} else if snapshot.Status == QueueStatusWaiting && isQueueStatusActive(snapshot, time.Now()) {
		if err := s.client.ZAdd(ctx, redisWaitingQueueKey(snapshot.EventID), goredis.Z{
			Score:  float64(snapshot.QueueSequence),
			Member: snapshot.QueueToken,
		}).Err(); err != nil {
			return err
		}
	}
	if err := s.client.Set(ctx, redisUserEventKey(queueUserEventKey(snapshot.EventID, snapshot.UserID)), snapshot.QueueToken, time.Until(snapshot.ExpiredAt)).Err(); err != nil {
		return err
	}
	return s.saveSnapshot(ctx, snapshot)
}

func (s *RedisQueueStore) SaveSnapshot(ctx context.Context, snapshot QueueStatusSnapshot) error {
	if snapshot.PurchaseToken != nil && snapshot.PurchaseTokenExpiresAt != nil {
		if err := s.client.ZAdd(ctx, redisReadyQueueKey(snapshot.EventID), goredis.Z{
			Score:  float64(snapshot.QueueSequence),
			Member: snapshot.QueueToken,
		}).Err(); err != nil {
			return err
		}
	} else if snapshot.Status == QueueStatusWaiting && isQueueStatusActive(snapshot, time.Now()) {
		if err := s.client.ZAdd(ctx, redisWaitingQueueKey(snapshot.EventID), goredis.Z{
			Score:  float64(snapshot.QueueSequence),
			Member: snapshot.QueueToken,
		}).Err(); err != nil {
			return err
		}
	}
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

func (s *RedisQueueStore) CleanupExpiredQueues(ctx context.Context, now time.Time) (int, error) {
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

		count, cleanupErr := s.cleanupExpiredQueuesByEvent(ctx, eventID, now)
		if cleanupErr != nil {
			return cleaned, cleanupErr
		}
		cleaned += count
	}

	return cleaned, nil
}

func (s *RedisQueueStore) promoteReady(ctx context.Context, eventID int64, now time.Time) error {
	readyCount, err := s.client.ZCard(ctx, redisReadyQueueKey(eventID)).Result()
	if err != nil {
		return err
	}
	slots := s.releaseLimit - int(readyCount)
	if slots <= 0 {
		return nil
	}

	expiresAt := now.Add(5 * time.Minute)
	purchaseTokenTTL := time.Until(expiresAt)
	if purchaseTokenTTL <= 0 {
		return nil
	}

	args := []interface{}{
		s.releaseLimit,
		now.Format(time.RFC3339Nano),
		expiresAt.Format(time.RFC3339Nano),
		purchaseTokenTTL.Milliseconds(),
		redisQueueTokenKeyPrefix(),
		redisPurchaseTokenKeyPrefix(),
	}
	for index := 0; index < s.releaseLimit; index++ {
		args = append(args, generatePurchaseToken(now))
	}

	if _, err := promoteReadyScript.Run(ctx, s.client, []string{
		redisWaitingQueueKey(eventID),
		redisReadyQueueKey(eventID),
	}, args...).Result(); err != nil {
		return err
	}

	return nil
}

func (s *RedisQueueStore) cleanupExpiredPurchaseTokensByEvent(ctx context.Context, eventID int64, now time.Time) (int, error) {
	tokens, err := s.client.ZRange(ctx, redisReadyQueueKey(eventID), 0, -1).Result()
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
		if err := s.client.ZRem(ctx, redisReadyQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
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

	if err := s.removeActiveEventIfEmpty(ctx, eventID); err != nil {
		return cleaned, err
	}

	return cleaned, nil
}

func (s *RedisQueueStore) cleanupExpiredQueuesByEvent(ctx context.Context, eventID int64, now time.Time) (int, error) {
	waitingTokens, err := s.client.ZRange(ctx, redisWaitingQueueKey(eventID), 0, -1).Result()
	if err != nil {
		return 0, err
	}
	readyTokens, err := s.client.ZRange(ctx, redisReadyQueueKey(eventID), 0, -1).Result()
	if err != nil {
		return 0, err
	}
	tokens := append(waitingTokens, readyTokens...)
	if len(tokens) == 0 {
		return 0, nil
	}

	cleaned := 0
	for _, token := range tokens {
		snapshot, loadErr := s.loadSnapshot(ctx, token)
		if loadErr != nil {
			continue
		}
		if !snapshot.ExpiredAt.Before(now) {
			continue
		}

		if snapshot.PurchaseToken != nil {
			if err := s.client.Del(ctx, redisPurchaseTokenKey(*snapshot.PurchaseToken)).Err(); err != nil {
				return cleaned, err
			}
		}
		if err := s.client.Del(ctx, redisUserEventKey(queueUserEventKey(snapshot.EventID, snapshot.UserID))).Err(); err != nil {
			return cleaned, err
		}
		if err := s.client.ZRem(ctx, redisWaitingQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
			return cleaned, err
		}
		if err := s.client.ZRem(ctx, redisReadyQueueKey(snapshot.EventID), snapshot.QueueToken).Err(); err != nil {
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

	if err := s.removeActiveEventIfEmpty(ctx, eventID); err != nil {
		return cleaned, err
	}

	return cleaned, nil
}

func (s *RedisQueueStore) removeActiveEventIfEmpty(ctx context.Context, eventID int64) error {
	waitingSize, waitingErr := s.client.ZCard(ctx, redisWaitingQueueKey(eventID)).Result()
	if waitingErr != nil {
		return waitingErr
	}
	readySize, readyErr := s.client.ZCard(ctx, redisReadyQueueKey(eventID)).Result()
	if readyErr != nil {
		return readyErr
	}
	if waitingSize > 0 || readySize > 0 {
		return nil
	}
	return s.client.SRem(ctx, redisActiveEventsKey(), strconv.FormatInt(eventID, 10)).Err()
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

func redisWaitingQueueKey(eventID int64) string {
	return "queue:waiting:" + strconv.FormatInt(eventID, 10)
}

func redisReadyQueueKey(eventID int64) string {
	return "queue:ready:" + strconv.FormatInt(eventID, 10)
}

func redisEventSeqKey(eventID int64) string {
	return "queue:event:" + strconv.FormatInt(eventID, 10) + ":seq"
}

func redisQueueTokenKey(queueToken string) string {
	return redisQueueTokenKeyPrefix() + queueToken
}

func redisQueueTokenKeyPrefix() string {
	return "queue:token:"
}

func redisPurchaseTokenKey(purchaseToken string) string {
	return redisPurchaseTokenKeyPrefix() + purchaseToken
}

func redisPurchaseTokenKeyPrefix() string {
	return "queue:purchase:"
}

func redisUserEventKey(key string) string {
	return "queue:user:event:" + key
}

func redisActiveEventsKey() string {
	return "queue:events:active"
}
