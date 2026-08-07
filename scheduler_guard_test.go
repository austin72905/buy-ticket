package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalJobGuardsSkipsOverlappingSameJob(t *testing.T) {
	guards := newLocalJobGuards()
	started := make(chan struct{})
	release := make(chan struct{})
	var runs int32

	go guards.Run(context.Background(), "job-a", func(ctx context.Context) {
		atomic.AddInt32(&runs, 1)
		close(started)
		<-release
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first job did not start")
	}

	guards.Run(context.Background(), "job-a", func(ctx context.Context) {
		atomic.AddInt32(&runs, 1)
	})
	close(release)

	if got := atomic.LoadInt32(&runs); got != 1 {
		t.Fatalf("expected overlapping job to be skipped, got runs=%d", got)
	}
}

func TestLocalJobGuardsAllowsDifferentJobs(t *testing.T) {
	guards := newLocalJobGuards()
	var runs int32

	guards.Run(context.Background(), "job-a", func(ctx context.Context) {
		atomic.AddInt32(&runs, 1)
	})
	guards.Run(context.Background(), "job-b", func(ctx context.Context) {
		atomic.AddInt32(&runs, 1)
	})

	if got := atomic.LoadInt32(&runs); got != 2 {
		t.Fatalf("expected different jobs to run, got runs=%d", got)
	}
}
