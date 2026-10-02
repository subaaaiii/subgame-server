package helpers

import (
	"bnsp2/server/redis"
	"context"
)

func DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64

	for {
		keys, nextCursor, err := redis.RedisClient.Scan(
			ctx,
			cursor,
			pattern,
			100,
		).Result()

		if err != nil {
			return err
		}

		if len(keys) > 0 {
			redis.RedisClient.Del(ctx, keys...)
		}

		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return nil
}
