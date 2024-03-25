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
	RemoveOldRequests(ctx context.Context, userID int64, limit int) error
	GetTopRequest(ctx context.Context) (int64, string, error)
	InsertRequest(ctx context.Context, userID int64, requestTime time.Time) error
}

type RedisFloodControl struct {
	redisClient *redis.Client
}

func NewRedisFloodControl(redisAddr string, dbIndexTimeRequests, dbIndexRequestCount int) (*RedisFloodControl, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   dbIndexTimeRequests,
	})

	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	return &RedisFloodControl{redisClient: redisClient}, nil
}

func (fc *RedisFloodControl) Check(ctx context.Context, userID int64) (bool, error) {
	key := strconv.FormatInt(userID, 10)

	count, err := fc.redisClient.LLen(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count > int64(K) {
		return false, nil
	}

	return true, nil
}

func (fc *RedisFloodControl) GetTopRequest(ctx context.Context) (int64, string, error) {
	value, err := fc.redisClient.LIndex(ctx, "requests", 0).Result()
	if err != nil {
		return 0, "", err
	}

	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("неправильный формат элемента списка: %s", value)
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	requestTime := parts[1]

	return userID, requestTime, err
}

func (fc *RedisFloodControl) RemoveOldRequests(ctx context.Context, userID int64) error {
	length, err := fc.redisClient.LLen(ctx, "requests").Result()
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

	now := time.Now()
	for now - requestTime > N {
		err := fc.redisClient.LPop(ctx, "requests").Err()
		if err != nil {
			panic(err)
		}

		userID, requestTime, err = fc.GetTopRequest(ctx)
	}

	return nil
}

func (fc *RedisFloodControl) InsertRequest(ctx context.Context, userID int64, requestTime time.Time) error {
	err := fc.redisClient.RPush(ctx, "requests", fmt.Sprintf("%s:%d", userID, requestTime)).Err()
	if err != nil {
		return err
	}

	return nil
}
