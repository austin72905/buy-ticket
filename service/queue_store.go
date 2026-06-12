package service

import (
	"context"
	"time"
)

type QueueStore interface {
	Join(ctx context.Context, input JoinQueueInput, now time.Time) (*QueueStatusSnapshot, error)
	Get(ctx context.Context, queueToken string, now time.Time) (*QueueStatusSnapshot, error)
	ValidatePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error)
	ConsumePurchaseToken(ctx context.Context, purchaseToken string, eventID, userID int64, now time.Time) (*QueueStatusSnapshot, error)
	RestorePurchaseToken(ctx context.Context, snapshot QueueStatusSnapshot) error
	SaveSnapshot(ctx context.Context, snapshot QueueStatusSnapshot) error
	PromoteReady(ctx context.Context, now time.Time) error
	CleanupExpiredPurchaseTokens(ctx context.Context, now time.Time) (int, error)
	CleanupExpiredQueues(ctx context.Context, now time.Time) (int, error)
}
