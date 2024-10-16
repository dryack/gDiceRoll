package dsl

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyPrefix    = "dice_roll:"
	diceRollListKey   = "dice_roll_keys"
	defaultExpiration = 24 * time.Hour
)

// TODO: rename these, it's possible we're going to expand this interface, and these names are meaningless
type Cache interface {
	Get(ctx context.Context, key string) (*CachedResult, error)
	Set(ctx context.Context, key string, value *CachedResult) error
	SetGeneral(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetGeneral(ctx context.Context, key string) (string, error)
}

type Database interface {
	Get(ctx context.Context, key string) (*CachedResult, error)
	Set(ctx context.Context, key string, value *CachedResult) error
}

type DragonflyCache struct {
	client          *redis.Client
	maxCacheEntries int
}

func NewDragonflyCache(client *redis.Client, maxCacheEntries int) *DragonflyCache {
	return &DragonflyCache{
		client:          client,
		maxCacheEntries: maxCacheEntries,
	}
}

func (c *DragonflyCache) Get(ctx context.Context, key string) (*CachedResult, error) {
	cacheKey := cacheKeyPrefix + key
	data, err := c.client.Get(ctx, cacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var result CachedResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	// Move this key to the front of the list (most recently used)
	c.client.LRem(ctx, diceRollListKey, 0, cacheKey)
	c.client.LPush(ctx, diceRollListKey, cacheKey)

	return &result, nil
}

func (c *DragonflyCache) Set(ctx context.Context, key string, value *CachedResult) error {
	cacheKey := cacheKeyPrefix + key
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Use a pipeline to perform multiple operations
	pipe := c.client.Pipeline()

	// Set the value
	pipe.Set(ctx, cacheKey, data, defaultExpiration)

	// Add the key to the front of the list
	pipe.LPush(ctx, diceRollListKey, cacheKey)

	// Trim the list to maintain max cache size
	pipe.LTrim(ctx, diceRollListKey, 0, int64(c.maxCacheEntries-1))

	// Remove any dice roll keys that are no longer in the list
	pipe.Eval(ctx, `
		local keys = redis.call('LRANGE', KEYS[1], ARGV[1], -1)
		if #keys > 0 then
			redis.call('DEL', unpack(keys))
		end
		redis.call('LTRIM', KEYS[1], 0, ARGV[1] - 1)
	`, []string{diceRollListKey}, c.maxCacheEntries)

	_, err = pipe.Exec(ctx)
	return err
}

func (c *DragonflyCache) SetGeneral(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *DragonflyCache) GetGeneral(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

type PostgresDB struct {
	pool *pgxpool.Pool
}

func NewPostgresDB(pool *pgxpool.Pool) *PostgresDB {
	return &PostgresDB{pool: pool}
}

func (db *PostgresDB) Get(ctx context.Context, key string) (*CachedResult, error) {
	var data []byte
	err := db.pool.QueryRow(ctx, "SELECT data FROM dice_results WHERE expression = $1", key).Scan(&data)
	if err != nil {
		return nil, err
	}

	var result CachedResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (db *PostgresDB) Set(ctx context.Context, key string, value *CachedResult) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = db.pool.Exec(ctx,
		"INSERT INTO dice_results (expression, data) VALUES ($1, $2) ON CONFLICT (expression) DO UPDATE SET data = $2",
		key, data)
	return err
}
