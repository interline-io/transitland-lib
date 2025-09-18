package testutil

import (
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/interline-io/transitland-lib/server/dbutil"
)

func CheckTestRedisClient() (string, bool) {
	_, a, ok := CheckEnv("TL_TEST_REDIS_URL")
	return a, ok
}

func MustOpenTestRedisClient(t testing.TB) *redis.Client {
	redisClient, err := dbutil.OpenRedis(os.Getenv("TL_TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return redisClient
}
