package model

import "errors"

// 领域错误集合，供 service / store / httpapi 统一映射为 HTTP 状态码。
var (
	// ErrNotFound 实体不存在。
	ErrNotFound = errors.New("entity not found")
	// ErrInvalidInput 输入参数非法（字段缺失、单位未知、重复样本等）。
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidState 状态机流转非法。
	ErrInvalidState = errors.New("invalid state transition")
	// ErrConflict 并发或互斥约束冲突。
	ErrConflict = errors.New("conflict")
	// ErrDuplicate 重复实体（样本重复、证据重复等）。
	ErrDuplicate = errors.New("duplicate entity")
	// ErrForbidden 操作被禁止（如已冻结版本直接修改而非创建替代版本）。
	ErrForbidden = errors.New("operation forbidden")
)

// Is 便捷判定，便于调用方用 errors.Is。
func IsNotFound(err error) bool   { return errors.Is(err, ErrNotFound) }
func IsInvalidInput(err error) bool { return errors.Is(err, ErrInvalidInput) }
func IsInvalidState(err error) bool { return errors.Is(err, ErrInvalidState) }
func IsConflict(err error) bool    { return errors.Is(err, ErrConflict) }
func IsDuplicate(err error) bool   { return errors.Is(err, ErrDuplicate) }
func IsForbidden(err error) bool   { return errors.Is(err, ErrForbidden) }
