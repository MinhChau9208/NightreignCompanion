package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRunsRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nrc.db")
	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	older := Run{Mode: "normal", Nightlord: "gladius", Result: "win", CreatedAt: time.Now().Add(-time.Hour)}
	newer := Run{Mode: "deep", Depth: 3, Night1Boss: "a", Night2Boss: "b", Result: "loss"}
	for _, r := range []*Run{&older, &newer} {
		if err := db.InsertRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	if newer.ID == "" {
		t.Fatal("InsertRun must assign an ID")
	}

	runs, err := db.ListRuns(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 || runs[0].ID != newer.ID || runs[1].Nightlord != "gladius" || runs[0].Depth != 3 {
		t.Fatalf("unexpected runs: %+v", runs)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nrc.db")
	for i := 0; i < 2; i++ {
		db, err := Open(ctx, path)
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		db.Close()
	}
}
