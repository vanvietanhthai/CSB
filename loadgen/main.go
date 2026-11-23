// Redis load generator cho môn Cloud Service Benchmarking (TU Berlin).
// Thư viện dùng: github.com/redis/go-redis/v9 (Redis client chính thức cho Go).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config chứa mọi tham số của một lần chạy benchmark.
type Config struct {
	Addr        string        // địa chỉ Redis, ví dụ "10.0.0.5:6379"
	Concurrency int           // số worker (goroutine) chạy song song
	Duration    time.Duration // thời gian đo chính thức
	Warmup      time.Duration // thời gian khởi động, KHÔNG tính vào kết quả
	ReadRatio   float64       // tỉ lệ GET, 0.5 = 50% GET, 50% SET (workload W-Mixed)
	ValueSize   int           // kích thước value (byte) khi SET
	Keyspace    int           // số lượng key khác nhau
	Label       string        // tên thí nghiệm, dùng đặt tên file kết quả
	Run         int           // số thứ tự lần lặp lại (repetition)
}

func parseFlags() Config {
	var c Config
	flag.StringVar(&c.Addr, "addr", "localhost:6379", "Redis address host:port")
	flag.IntVar(&c.Concurrency, "concurrency", 10, "number of concurrent workers")
	flag.DurationVar(&c.Duration, "duration", 60*time.Second, "measurement duration (e.g. 60s, 2m)")
	flag.DurationVar(&c.Warmup, "warmup", 10*time.Second, "warmup duration, excluded from results")
	flag.Float64Var(&c.ReadRatio, "read-ratio", 0.5, "fraction of GET operations (0.0 - 1.0)")
	flag.IntVar(&c.ValueSize, "value-size", 1024, "value size in bytes for SET")
	flag.IntVar(&c.Keyspace, "keyspace", 100000, "number of distinct keys")
	flag.StringVar(&c.Label, "label", "test", "experiment label, used in output file names")
	flag.IntVar(&c.Run, "run", 1, "repetition number of this experiment")
	flag.Parse()
	return c
}

// validate kiểm tra tham số hợp lệ trước khi chạy, tránh lãng phí thời gian VM.
func (c Config) validate() error {
	if c.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be > 0")
	}
	if c.Warmup < 0 {
		return fmt.Errorf("warmup must be >= 0")
	}
	if c.ReadRatio < 0 || c.ReadRatio > 1 {
		return fmt.Errorf("read-ratio must be between 0 and 1")
	}
	if c.ValueSize < 1 {
		return fmt.Errorf("value-size must be >= 1")
	}
	if c.Keyspace < 1 {
		return fmt.Errorf("keyspace must be >= 1")
	}
	return nil
}

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

func main() {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config:", err)
		os.Exit(1)
	}
	fmt.Printf("config: %+v\n", cfg)

	ctx := context.Background()
	rdb := newClient(cfg)
	defer rdb.Close()

	if err := checkConnection(ctx, rdb); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	persist, err := readPersistenceConfig(ctx, rdb)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("redis persistence: appendonly=%s appendfsync=%s save=%q\n",
		persist["appendonly"], persist["appendfsync"], persist["save"])
}
