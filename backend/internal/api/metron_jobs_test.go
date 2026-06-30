package api

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMetronImportJobProgressAndCancel(t *testing.T) {
	store := newMetronImportJobStore()
	running := make(chan struct{})

	job := store.start("series", 123, "Starting...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(1, 10, "Imported one issue.")
		close(running)
		<-ctx.Done()
		return ctx.Err()
	})

	select {
	case <-running:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}

	current, ok := store.get(job.ID)
	if !ok {
		t.Fatal("job missing")
	}
	if current.Completed != 1 || current.Total != 10 {
		t.Fatalf("progress = %d/%d; want 1/10", current.Completed, current.Total)
	}

	canceled, ok := store.cancelJob(job.ID)
	if !ok {
		t.Fatal("cancel returned missing job")
	}
	if canceled.Status != "canceling" {
		t.Fatalf("cancel status = %q; want canceling", canceled.Status)
	}

	deadline := time.After(time.Second)
	for {
		current, _ = store.get(job.ID)
		if current.Status == "canceled" {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("status = %q; want canceled", current.Status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestMetronImportJobCancelingDoesNotBecomeSucceeded(t *testing.T) {
	store := newMetronImportJobStore()
	release := make(chan struct{})
	running := make(chan struct{})

	job := store.start("series", 123, "Starting...", func(ctx context.Context, progress func(int, int, string)) error {
		close(running)
		<-release
		return nil
	})

	select {
	case <-running:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}

	if canceled, ok := store.cancelJob(job.ID); !ok {
		t.Fatal("cancel returned missing job")
	} else if canceled.Status != "canceling" {
		t.Fatalf("cancel status = %q; want canceling", canceled.Status)
	}
	close(release)

	deadline := time.After(time.Second)
	for {
		current, _ := store.get(job.ID)
		if current.Status == "canceled" {
			return
		}
		if current.Status == "succeeded" {
			t.Fatal("canceled job became succeeded")
		}
		select {
		case <-deadline:
			t.Fatalf("status = %q; want canceled", current.Status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestContextCanceledStringIsTreatedAsCancellation(t *testing.T) {
	err := fmt.Errorf(`Get "https://metron.cloud/api/issue/335/": context canceled`)
	if !isContextCanceledError(err) {
		t.Fatal("wrapped context canceled string was not treated as cancellation")
	}
}
