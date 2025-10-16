package storage

import "errors"

// 存储相关错误
var (
	ErrBackendNotFound = errors.New("storage backend not found")
	ErrDataNotFound    = errors.New("data not found")
	ErrInvalidConfig   = errors.New("invalid storage configuration")
	ErrUploadFailed    = errors.New("upload failed")
	ErrDownloadFailed  = errors.New("download failed")
	ErrDeleteFailed    = errors.New("delete failed")
)
