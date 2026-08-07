package redis

import (
	"context"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/valkeyrie/testsuite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 60 * time.Second

const testAddress = "localhost:6379"

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

	kv, err := valkeyrie.NewStore(ctx, StoreName, []string{testAddress}, nil)
	require.NoError(t, err)
	assert.NotNil(t, kv)

	assert.IsTypef(t, kv, new(Store), "Error registering and initializing Redis")
}

func TestRedisStore(t *testing.T) {
	kv := makeRedisClient(t, []string{testAddress}, nil)
	lockTTL := makeRedisClient(t, []string{testAddress}, nil)
	kvTTL := makeRedisClient(t, []string{testAddress}, nil)

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

func TestRedisSentinelStore_withClientCluster(t *testing.T) {
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

func TestWatchLoopLeadingAndTrailingDebounce(t *testing.T) {
	ctx := t.Context()

	debounce := 500 * time.Millisecond

	r := &Store{
		opts: storeOptions{
			WatchDebounce: debounce,
		},
	}

	msgCh := make(chan *redis.Message)
	pushCh := make(chan any, 4)

	get := getter(func() (any, error) {
		return nil, store.ErrKeyNotFound
	})
	push := pusher(func(value any) {
		pushCh <- value
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.watchLoop(ctx, msgCh, get, push)
	}()

	require.IsType(t, &store.KVPair{}, receiveWatchValue(t, debounce, pushCh))

	msgCh <- &redis.Message{Payload: "set"}
	assert.Nil(t, receiveWatchValue(t, debounce, pushCh))

	msgCh <- &redis.Message{Payload: "set"}
	time.Sleep(debounce / 2)
	msgCh <- &redis.Message{Payload: "del"}

	select {
	case value := <-pushCh:
		t.Fatalf("received trailing value before debounce elapsed: %#v", value)
	case <-time.After(debounce / 2):
	}

	require.IsType(t, &store.KVPair{}, receiveWatchValue(t, debounce, pushCh))

	close(msgCh)
	require.NoError(t, <-errCh)
}

func receiveWatchValue(t *testing.T, debounce time.Duration, values <-chan any) any {
	t.Helper()

	select {
	case value := <-values:
		return value
	case <-time.After(2 * debounce):
		t.Fatal("timed out waiting for watched value")
		return nil
	}
}
