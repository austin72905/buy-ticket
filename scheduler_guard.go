package main

import (
	"context"
	"sync"

	"buy-ticket/observability"
)

type localJobGuards struct {
	mu      sync.Mutex
	running map[string]struct{}
}

func newLocalJobGuards() *localJobGuards {
	return &localJobGuards{
		running: make(map[string]struct{}),
	}
}

func (g *localJobGuards) Run(ctx context.Context, jobName string, run func(context.Context)) {
	if !g.tryStart(jobName) {
		observability.Info(ctx, "scheduler job skipped", "job_name", jobName, "reason", "already_running")
		return
	}
	defer g.finish(jobName)

	run(ctx)
}

func (g *localJobGuards) tryStart(jobName string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.running[jobName]; exists {
		return false
	}
	// 如果目前沒在跑，就把 job name 放進 map，表示「這個 job 現在開始執行了」。
	g.running[jobName] = struct{}{}
	return true
}

func (g *localJobGuards) finish(jobName string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.running, jobName)
}
