package store

import (
	"encoding/json"
	"time"
)

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
