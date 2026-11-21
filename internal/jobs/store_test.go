package jobs

import (
	"atad-project/internal/gitlog"
	"context"
	"testing"
	"time"
)

func TestResetCancelsActiveHistory(t *testing.T) {
	s := New(nil)
	done := make(chan struct{})
	job := s.Start("count", func(ctx context.Context, visit func(gitlog.Commit)) error {
		<-ctx.Done()
		close(done)
		return ctx.Err()
	})
	s.Reset()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if _, found := s.Get(job.ID); found {
		t.Fatal("reset retained job")
	}
}
