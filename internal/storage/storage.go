package storage

import (
	"cnet/internal/workload"
	"context"
	"time"
)

// StorageBackend 存储后端接口
type StorageBackend interface {
	// Upload 上传数据
	Upload(ctx context.Context, data *workload.DataWorkload) error

	// Download 下载数据
	Download(ctx context.Context, data *workload.DataWorkload) error

	// Delete 删除数据
	Delete(ctx context.Context, data *workload.DataWorkload) error

	// GetInfo 获取数据信息
	GetInfo(ctx context.Context, data *workload.DataWorkload) (*DataInfo, error)

	// List 列出数据
	List(ctx context.Context, prefix string) ([]*DataInfo, error)

	// Exists 检查数据是否存在
	Exists(ctx context.Context, dataKey string) (bool, error)
}

// DataInfo 数据信息
type DataInfo struct {
	DataKey       string              `json:"data_key"`
	Name          string              `json:"name"`
	DataType      workload.DataType   `json:"data_type"`
	ContentType   string              `json:"content_type"`
	Size          int64               `json:"size"`
	Hash          string              `json:"hash"`
	FilePath      string              `json:"file_path"`
	DirectoryPath string              `json:"directory_path,omitempty"`
	FileCount     int                 `json:"file_count,omitempty"`
	AccessMode    workload.AccessMode `json:"access_mode"`
	Tags          []string            `json:"tags"`
	Metadata      map[string]string   `json:"metadata"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type   string       `yaml:"type"` // 存储类型
	SQLite SQLiteConfig `yaml:"sqlite"`
	Local  LocalConfig  `yaml:"local"`
	Badger BadgerConfig `yaml:"badger"`
}

// SQLiteConfig SQLite配置
type SQLiteConfig struct {
	DBPath   string `yaml:"db_path"`   // 数据库文件路径
	DataPath string `yaml:"data_path"` // 数据文件存储路径
}

// LocalConfig 本地文件系统配置
type LocalConfig struct {
	BasePath string `yaml:"base_path"` // 基础路径
}

// BadgerConfig BadgerDB配置
type BadgerConfig struct {
	DataDir string `yaml:"data_dir"` // 数据目录
}

// StorageManager 存储管理器
type StorageManager struct {
	backends map[string]StorageBackend
	config   StorageConfig
}

// NewStorageManager 创建存储管理器
func NewStorageManager(config StorageConfig) *StorageManager {
	return &StorageManager{
		backends: make(map[string]StorageBackend),
		config:   config,
	}
}

// RegisterBackend 注册存储后端
func (sm *StorageManager) RegisterBackend(name string, backend StorageBackend) {
	sm.backends[name] = backend
}

// GetBackend 获取存储后端
func (sm *StorageManager) GetBackend(name string) (StorageBackend, bool) {
	backend, exists := sm.backends[name]
	return backend, exists
}

// GetDefaultBackend 获取默认存储后端
func (sm *StorageManager) GetDefaultBackend() (StorageBackend, error) {
	backend, exists := sm.backends[sm.config.Type]
	if !exists {
		return nil, ErrBackendNotFound
	}
	return backend, nil
}

// Upload 上传数据
func (sm *StorageManager) Upload(ctx context.Context, data *workload.DataWorkload) error {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return err
	}
	return backend.Upload(ctx, data)
}

// Download 下载数据
func (sm *StorageManager) Download(ctx context.Context, data *workload.DataWorkload) error {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return err
	}
	return backend.Download(ctx, data)
}

// Delete 删除数据
func (sm *StorageManager) Delete(ctx context.Context, data *workload.DataWorkload) error {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return err
	}
	return backend.Delete(ctx, data)
}

// GetInfo 获取数据信息
func (sm *StorageManager) GetInfo(ctx context.Context, data *workload.DataWorkload) (*DataInfo, error) {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return nil, err
	}
	return backend.GetInfo(ctx, data)
}

// List 列出数据
func (sm *StorageManager) List(ctx context.Context, prefix string) ([]*DataInfo, error) {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return nil, err
	}
	return backend.List(ctx, prefix)
}

// Exists 检查数据是否存在
func (sm *StorageManager) Exists(ctx context.Context, dataKey string) (bool, error) {
	backend, err := sm.GetDefaultBackend()
	if err != nil {
		return false, err
	}
	return backend.Exists(ctx, dataKey)
}
