package floodcontrol

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strconv"
	"strings"
	"time"
)

var N, K int

// FloodControl интерфейс, который нужно реализовать.
// Рекомендуем создать директорию-пакет, в которой будет находиться реализация.
type FloodControl interface {
	// Check возвращает false если достигнут лимит максимально разрешенного
	// кол-ва запросов согласно заданным правилам флуд контроля.
	Check(ctx context.Context, userID int64) (bool, error)
	AddRequest(ctx context.Context, userID int64) error
	RemoveOldRequests(ctx context.Context) error
	GetTopRequest(ctx context.Context) (int64, int64, error)
	InsertRequest(ctx context.Context, userID int64, requestTime time.Time) error
}

type RedisFloodControl struct {
	redisClientTimeRequests *redis.Client
	redisClientRequestCount *redis.Client
}

func NewRedisFloodControl(redisAddr string, dbIndexTimeRequests int, dbIndexRequestCount int) (*RedisFloodControl, error) {
	redisClientTimeRequests := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   dbIndexTimeRequests,
	})

	if _, err := redisClientTimeRequests.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	redisClientRequestCount := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   dbIndexRequestCount,
	})

	if _, err := redisClientRequestCount.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisFloodControl{
		redisClientTimeRequests: redisClientTimeRequests,
		redisClientRequestCount: redisClientRequestCount,
	}, nil
}

func (fc *RedisFloodControl) Close() error {
	if err := fc.redisClientTimeRequests.Close(); err != nil {
		return err
	}
	return fc.redisClientRequestCount.Close()
}

func (fc *RedisFloodControl) Check(ctx context.Context, userID int64) (bool, error) {
	key := strconv.FormatInt(userID, 10)

	count, err := fc.redisClientRequestCount.Get(ctx, key).Int64()
	if err != nil {
		return false, err
	}

	if count > int64(K) {
		return false, nil
	}

	return true, nil
}

func (fc *RedisFloodControl) GetTopRequest(ctx context.Context) (int64, int64, error) {
	value, err := fc.redisClientTimeRequests.LIndex(ctx, "requests", 0).Result()
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("неправильный формат элемента списка: %s", value)
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	requestTime, err := strconv.ParseInt(parts[1], 10, 64)

	return userID, requestTime, err
}

func (fc *RedisFloodControl) RemoveOldRequests(ctx context.Context) error {
	length, err := fc.redisClientTimeRequests.LLen(ctx, "requests").Result()
	if err != nil {
		return err
	}

	if length == 0 {
		return nil
	}

	userID, requestTime, err := fc.GetTopRequest(ctx)
	if err != nil {
		panic(err)
	}

	now := int64(time.Now().Unix())

	for now-requestTime > int64(N) {
		if err := fc.redisClientTimeRequests.LPop(ctx, "requests").Err(); err != nil {
			return err
		}

		if err := fc.redisClientRequestCount.Decr(ctx, strconv.FormatInt(userID, 10)).Err(); err != nil {
			return err
		}

		userID, requestTime, err = fc.GetTopRequest(ctx)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (fc *RedisFloodControl) InsertRequest(ctx context.Context, userID int64, requestTime time.Time) error {
	err := fc.redisClientTimeRequests.RPush(ctx, "requests", fmt.Sprintf("%d:%d", userID, requestTime.Unix())).Err()
	if err != nil {
		return err
	}

	err = fc.redisClientRequestCount.Incr(ctx, fmt.Sprintf("count:%d", userID)).Err()
	if err != nil {
		return err
	}

	return nil
}
