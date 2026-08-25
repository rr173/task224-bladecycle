package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// DBTX 是 *sql.DB 与 *sql.Tx 共同满足的最小执行接口。
// 让存储在同一实例上切换"直接执行"或"事务执行"，便于将多步写入
// 包进单事务以保证幂等重算不破坏上一轮完整结果。
type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// nowStr 返回统一的时间戳字符串（RFC3339Nano）。
func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// parseTime 解析时间戳字符串，失败返回零值时间。
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// encodePeaks 序列化峰谷序列。
func encodePeaks(peaks []float64) (string, error) {
	b, err := json.Marshal(peaks)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodePeaks 反序列化峰谷序列。
func decodePeaks(s string) ([]float64, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []float64
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// encodeIDs 序列化 ID 切片。
func encodeIDs(ids []int64) (string, error) {
	b, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeIDs 反序列化 ID 切片。
func decodeIDs(s string) ([]int64, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []int64
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}
