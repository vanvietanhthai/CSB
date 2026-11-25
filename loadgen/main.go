// Redis load generator cho môn Cloud Service Benchmarking (TU Berlin).
// Thư viện dùng: github.com/redis/go-redis/v9 (Redis client chính thức cho Go).
package main

import (
	"context"
	"fmt"
	"os"
)

// main chỉ in lỗi và đặt exit code. Phần việc chính nằm trong run,
// vì os.Exit bỏ qua mọi defer (ví dụ rdb.Close).
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run thực hiện toàn bộ một lần chạy và trả về lỗi đầu tiên gặp phải.
func run() error {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	fmt.Printf("config: %+v\n", cfg)

	ctx := context.Background()
	rdb := newClient(cfg)
	defer rdb.Close()

	if err := checkConnection(ctx, rdb); err != nil {
		return err
	}

	persist, err := readPersistenceConfig(ctx, rdb)
	if err != nil {
		return err
	}
	fmt.Printf("redis persistence: appendonly=%s appendfsync=%s save=%q\n",
		persist["appendonly"], persist["appendfsync"], persist["save"])

	return preload(ctx, rdb, cfg)
}
