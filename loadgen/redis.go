package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// newClient tạo kết nối tới Redis.
// PoolSize = Concurrency: mỗi worker có sẵn một kết nối riêng,
// để worker không phải xếp hàng chờ kết nối (nếu không, chính load generator
// sẽ thành nút cổ chai, chứ không phải Redis).
func newClient(cfg Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		PoolSize:     cfg.Concurrency,
		MinIdleConns: cfg.Concurrency, // mở sẵn kết nối từ đầu, không mở dần trong lúc đo
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
}

// checkConnection gửi PING. Redis đang chạy thì trả về "PONG".
func checkConnection(ctx context.Context, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("cannot reach Redis at %s: %w", rdb.Options().Addr, err)
	}
	fmt.Println("redis says:", pong)
	return nil
}

// readPersistenceConfig hỏi Redis đang dùng cấu hình lưu đĩa nào (C1..C4),
// để ghi lại cùng kết quả. Như vậy file kết quả tự chứng minh nó được đo
// với cấu hình nào, không phụ thuộc vào trí nhớ của bạn.
func readPersistenceConfig(ctx context.Context, rdb *redis.Client) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range []string{"appendonly", "appendfsync", "save"} {
		res, err := rdb.ConfigGet(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("CONFIG GET %s: %w", key, err)
		}
		out[key] = res[key]
	}
	return out, nil
}
