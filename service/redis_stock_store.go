package service

import (
	"context"
	"strconv"

	"buy-ticket/domain"
	infraredis "github.com/austin72905/go-infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

var reserveStockScript = goredis.NewScript(`
local quantity = tonumber(ARGV[1])
local initialAvailable = tonumber(ARGV[2])
local current = redis.call("GET", KEYS[1])

if not current then
	current = initialAvailable
	redis.call("SET", KEYS[1], initialAvailable)
else
	current = tonumber(current)
end

if current < quantity then
	return 0
end

redis.call("DECRBY", KEYS[1], quantity)
return 1
`)

var releaseStockScript = goredis.NewScript(`
local quantity = tonumber(ARGV[1])
local fallbackAvailable = tonumber(ARGV[2])
local current = redis.call("GET", KEYS[1])

if not current then
	redis.call("SET", KEYS[1], fallbackAvailable)
	return fallbackAvailable
end

return redis.call("INCRBY", KEYS[1], quantity)
`)

type RedisStockStore struct {
	client *goredis.Client
}

func NewRedisStockStore(client *infraredis.Client) *RedisStockStore {
	return &RedisStockStore{client: client.Raw()}
}

func (s *RedisStockStore) Reserve(ctx context.Context, section domain.Section, quantity int) error {
	if quantity <= 0 {
		return ErrInsufficientStock
	}

	result, err := reserveStockScript.Run(
		ctx,
		s.client,
		[]string{redisSectionStockKey(section.EventID, section.ID)},
		quantity,
		section.AvailableQuantity(),
	).Int()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrInsufficientStock
	}

	return nil
}

func (s *RedisStockStore) Release(ctx context.Context, section domain.Section, quantity int) error {
	if quantity <= 0 {
		return nil
	}

	return releaseStockScript.Run(
		ctx,
		s.client,
		[]string{redisSectionStockKey(section.EventID, section.ID)},
		quantity,
		section.AvailableQuantity(),
	).Err()
}

func (s *RedisStockStore) RebuildAll(ctx context.Context, sections []domain.Section) error {
	pipe := s.client.Pipeline()
	for _, section := range sections {
		pipe.Set(ctx, redisSectionStockKey(section.EventID, section.ID), section.AvailableQuantity(), 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStockStore) ReconcileAll(ctx context.Context, sections []domain.Section) (StockReconcileResult, error) {
	result := StockReconcileResult{Checked: len(sections)}

	for _, section := range sections {
		key := redisSectionStockKey(section.EventID, section.ID)
		expectedAvailable := section.AvailableQuantity()

		currentAvailable, err := s.client.Get(ctx, key).Int()
		if err == goredis.Nil {
			if err := s.client.Set(ctx, key, expectedAvailable, 0).Err(); err != nil {
				return result, err
			}
			result.Fixed++
			continue
		}
		if err != nil {
			return result, err
		}
		if currentAvailable == expectedAvailable {
			continue
		}

		if err := s.client.Set(ctx, key, expectedAvailable, 0).Err(); err != nil {
			return result, err
		}
		result.Fixed++
	}

	return result, nil
}

func redisSectionStockKey(eventID, sectionID int64) string {
	return "stock:event:" + strconv.FormatInt(eventID, 10) + ":section:" + strconv.FormatInt(sectionID, 10)
}
