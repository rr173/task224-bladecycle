// Package httpapi 是 HTTP 层，路由统一以 /api 开头，JSON 请求/响应。
// 使用 Go 标准库 net/http 的增强路由（method + path wildcard）。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"task224-bladecycle/internal/model"
)

// writeJSON 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将领域错误映射为 HTTP 状态码。
func writeError(w http.ResponseWriter, err error) {
	switch {
	case model.IsCanceled(err):
		// 客户端在写入完成前取消：未落库，不返回成功。
		writeJSON(w, statusClientClosedRequest, map[string]string{"error": err.Error()})
	case model.IsNotFound(err):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case model.IsConflict(err):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case model.IsInvalidArgument(err):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case model.IsInvalidState(err):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

// statusClientClosedRequest 表示客户端在响应完成前关闭了连接（nginx 沿用值）。
// 标准库未定义该状态码，用于把“取消且未落库”与 4xx 失败区分开。
const statusClientClosedRequest = 499

// parseID 解析路径参数为正整数 ID。
func parseID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + name})
		return 0, false
	}
	return id, true
}

// decodeJSON 解析 JSON 请求体。
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

// isBadRequest 判断是否请求体解析错误。
func isBadRequest(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.As(err, &syntaxErr) || errors.As(err, &typeErr)
}
