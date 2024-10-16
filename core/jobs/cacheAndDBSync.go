package jobs

import (
	"context"
	"log"
	"time"

	"github.com/dryack/gDiceRoll/core/dsl"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	syncLockKey = "sync_job_lock"
	lockTTL     = 16 * time.Minute // Slightly longer than our sync interval
)

type SyncJob struct {
	cache           dsl.Cache
	db              *pgxpool.Pool
	maxCacheEntries int
}

func NewSyncJob(cache dsl.Cache, db *pgxpool.Pool, maxCacheEntries int) *SyncJob {
	return &SyncJob{
		cache:           cache,
		db:              db,
		maxCacheEntries: maxCacheEntries,
	}
}

func (j *SyncJob) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if j.shouldRunSync(ctx) {
				j.syncCacheAndDB(ctx)
			} else {
				log.Println("Skipping sync job run.")
			}
		}
	}
}

func (j *SyncJob) shouldRunSync(ctx context.Context) bool {
	// Try to acquire the lock
	err := j.cache.SetGeneral(ctx, syncLockKey, time.Now().Unix(), lockTTL)
	if err != nil {
		log.Printf("Error acquiring lock: %v", err)
		return false
	}

	// If we get here, we've successfully acquired the lock
	return true
}

func (j *SyncJob) syncCacheAndDB(ctx context.Context) {
	log.Println("Starting cache to DB sync...")

	// Implement sync logic here
	// 1. Get all keys from cache
	// 2. For each key, get the value from cache
	// 3. Update the database with the cache value
	// 4. Log the number of items synced

	log.Println("Cache to DB sync completed")
}
