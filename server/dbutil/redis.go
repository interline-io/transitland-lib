package dbutil

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func OpenRedis(v string) (*redis.Client, error) {
	opts, err := getRedisOpts(v)
	if err != nil {
		return nil, err
	}
	redisClient := redis.NewClient(opts)
	return redisClient, nil
}

func getRedisOpts(v string) (*redis.Options, error) {
	a, err := url.Parse(v)
	if err != nil {
		return nil, err
	}
	if a.Scheme != "redis" {
		return nil, errors.New("redis URL must begin with redis://")
	}
	port := a.Port()
	if port == "" {
		port = "6379"
	}
	addr := fmt.Sprintf("%s:%s", a.Hostname(), port)
	dbNo := 0
	if len(a.Path) > 0 {
		var err error
		f := a.Path[1:len(a.Path)]
		dbNo, err = strconv.Atoi(f)
		if err != nil {
			return nil, err
		}
	}
	return &redis.Options{Addr: addr, DB: dbNo}, nil
}
