package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// keyName trả về tên key thứ i, ví dụ keyName(42) = "key:000042".
// Worker ở bước 5 sẽ dùng đúng hàm này, nên GET luôn trúng key đã nạp.
func keyName(i int) string {
	return fmt.Sprintf("key:%06d", i)
}

// randomValue tạo một chuỗi byte ngẫu nhiên dài size byte.
// Dùng dữ liệu ngẫu nhiên (không phải "aaaa...") để giống dữ liệu thật hơn.
func randomValue(size int) []byte {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// preload xoá dữ liệu cũ rồi nạp đủ cfg.Keyspace key trước khi đo.
// Dùng pipeline: gom 1000 lệnh SET gửi một lần thay vì gửi từng lệnh,
// nên nạp 100.000 key chỉ mất khoảng 1 đến 2 giây.
func preload(ctx context.Context, rdb *redis.Client, cfg Config) error {
	const batchSize = 1000
	start := time.Now()

	// Bắt đầu từ trạng thái sạch: xoá mọi key cũ trong database hiện tại.
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("FLUSHDB: %w", err)
	}

	value := randomValue(cfg.ValueSize)
	for from := 0; from < cfg.Keyspace; from += batchSize {
		to := min(from+batchSize, cfg.Keyspace)
		pipe := rdb.Pipeline()
		for i := from; i < to; i++ {
			pipe.Set(ctx, keyName(i), value, 0) // 0 = không hết hạn
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("preload batch %d-%d: %w", from, to, err)
		}
	}

	// Kiểm tra lại: Redis phải có đúng cfg.Keyspace key.
	n, err := rdb.DBSize(ctx).Result()
	if err != nil {
		return fmt.Errorf("DBSIZE: %w", err)
	}
	if n != int64(cfg.Keyspace) {
		return fmt.Errorf("preload: expected %d keys, Redis has %d", cfg.Keyspace, n)
	}
	fmt.Printf("preloaded %d keys of %d bytes in %v\n", n, cfg.ValueSize, time.Since(start).Round(time.Millisecond))
	return nil
}
