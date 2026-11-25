package main

import (
	"flag"
	"fmt"
	"time"
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
	flag.DurationVar(&c.Duration, "duration", 30*time.Minute, "measurement duration (e.g. 60s, 2m)")
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
