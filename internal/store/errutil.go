package store

import (
	"strings"

	"task215-fiberlineage/internal/model"
)

// isUniqueErr 判断底层驱动错误是否为唯一约束冲突，并映射为领域 ErrDuplicate。
func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "already exists")
}

// mapErr 将底层错误统一映射为领域错误，便于上层 HTTP 状态码判定。
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if isUniqueErr(err) {
		return model.ErrDuplicate
	}
	return err
}
