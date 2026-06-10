package service

import (
	"context"
	"log"
	"time"
)

type QueueDispatcher struct {
	store    QueueStore
	interval time.Duration
	cancel   context.CancelFunc
}

func NewQueueDispatcher(store QueueStore, interval time.Duration) *QueueDispatcher {
	if interval <= 0 {
		interval = time.Second
	}

	return &QueueDispatcher{
		store:    store,
		interval: interval,
	}
}

func (d *QueueDispatcher) Start() {
	if d == nil || d.store == nil || d.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	go func() {
		ticker := time.NewTicker(d.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := d.store.PromoteReady(ctx, now); err != nil {
					log.Printf("queue dispatcher promote failed: %v", err)
				}
			}
		}
	}()
}

func (d *QueueDispatcher) Stop() {
	if d == nil || d.cancel == nil {
		return
	}

	d.cancel()
	d.cancel = nil
}
