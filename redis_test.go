package redis

import (
	"context"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/valkeyrie/testsuite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 60 * time.Second

func makeRedisClient(t *testing.T, endpoints []string, config *Config) store.Store {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kv, err := NewWithCodec(ctx, endpoints, config, nil)
	require.NoError(t, err)

	// NOTE: please turn on redis's notification
	// before you using watch/watchtree/lock related features.
	kv.client.ConfigSet(ctx, "notify-keyspace-events", "KA")

	return kv
}

func TestRegister(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	kv, err := valkeyrie.NewStore(ctx, StoreName, []string{"localhost:6379"}, nil)
	require.NoError(t, err)
	assert.NotNil(t, kv)

	assert.IsTypef(t, kv, new(Store), "Error registering and initializing Redis")
}

func TestRedisStore(t *testing.T) {
	kv := makeRedisClient(t, []string{"localhost:6379"}, nil)
	lockTTL := makeRedisClient(t, []string{"localhost:6379"}, nil)
	kvTTL := makeRedisClient(t, []string{"localhost:6379"}, nil)

	t.Cleanup(func() {
		testsuite.RunCleanup(t, kv)
	})

	testsuite.RunTestCommon(t, kv)
	testsuite.RunTestAtomic(t, kv)
	testsuite.RunTestWatch(t, kv)
	testsuite.RunTestLock(t, kv)
	testsuite.RunTestLockTTL(t, kv, lockTTL)
	testsuite.RunTestTTL(t, kv, kvTTL)
}

func TestRedisSentinelStore(t *testing.T) {
	endpoints := []string{"localhost:26379", "localhost:36379", "localhost:46379"}
	config := &Config{Sentinel: &Sentinel{MasterName: "mymaster"}}

	kv := makeRedisClient(t, endpoints, config)
	lockTTL := makeRedisClient(t, endpoints, config)
	kvTTL := makeRedisClient(t, endpoints, config)

	t.Cleanup(func() {
		testsuite.RunCleanup(t, kv)
	})

	testsuite.RunTestCommon(t, kv)
	testsuite.RunTestAtomic(t, kv)
	testsuite.RunTestWatch(t, kv)
	testsuite.RunTestLock(t, kv)
	testsuite.RunTestLockTTL(t, kv, lockTTL)
	testsuite.RunTestTTL(t, kv, kvTTL)
}

func TestRedisSentinelStore_WithClientCluster(t *testing.T) {
	endpoints := []string{"localhost:26379", "localhost:36379", "localhost:46379"}
	config := &Config{Sentinel: &Sentinel{MasterName: "mymaster", ClusterClient: true}}

	kv := makeRedisClient(t, endpoints, config)
	lockTTL := makeRedisClient(t, endpoints, config)
	kvTTL := makeRedisClient(t, endpoints, config)

	t.Cleanup(func() {
		testsuite.RunCleanup(t, kv)
	})

	testsuite.RunTestCommon(t, kv)
	testsuite.RunTestAtomic(t, kv)
	testsuite.RunTestWatch(t, kv)
	testsuite.RunTestLock(t, kv)
	testsuite.RunTestLockTTL(t, kv, lockTTL)
	testsuite.RunTestTTL(t, kv, kvTTL)
}
