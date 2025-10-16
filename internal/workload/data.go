package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DataType 数据类型
type DataType string

const (
	DataTypeFile      DataType = "file"      // 单个文件
	DataTypeDirectory DataType = "directory" // 目录
	DataTypeDataset   DataType = "dataset"   // 数据集
	DataTypeModel     DataType = "model"     // 模型文件
	DataTypeConfig    DataType = "config"    // 配置文件
	DataTypeStream    DataType = "stream"    // 数据流
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal  StorageType = "local"  // 本地文件系统
	StorageTypeSQLite StorageType = "sqlite" // SQLite + 文件系统
	StorageTypeBadger StorageType = "badger" // BadgerDB
	StorageTypeS3     StorageType = "s3"     // S3兼容对象存储
)

// AccessMode 访问模式
type AccessMode string

const (
	AccessModeReadOnly  AccessMode = "readonly"  // 只读
	AccessModeReadWrite AccessMode = "readwrite" // 读写
	AccessModeExclusive AccessMode = "exclusive" // 独占
)

// UploadMethod 上传方式
type UploadMethod string

const (
	UploadMethodFile      UploadMethod = "file"      // 文件上传
	UploadMethodURL       UploadMethod = "url"       // URL下载
	UploadMethodPath      UploadMethod = "path"      // 本地路径
	UploadMethodDirectory UploadMethod = "directory" // 目录上传
)

// DataWorkload 数据workload
type DataWorkload struct {
	BaseWorkload

	// 数据标识
	DataKey     string   `json:"data_key"`     // 数据唯一标识
	DataType    DataType `json:"data_type"`    // 数据类型
	ContentType string   `json:"content_type"` // MIME类型
	Size        int64    `json:"size"`         // 数据大小（字节）
	Hash        string   `json:"hash"`         // 数据校验和

	// 存储配置
	StorageType    StorageType `json:"storage_type"`    // 存储类型
	StorageBackend string      `json:"storage_backend"` // 存储后端
	FilePath       string      `json:"file_path"`       // 本地文件路径
	DirectoryPath  string      `json:"directory_path"`  // 目录路径（目录类型）

	// 上传配置
	UploadMethod UploadMethod `json:"upload_method"` // 上传方式
	SourceURL    string       `json:"source_url"`    // 源URL（URL下载）
	SourcePath   string       `json:"source_path"`   // 源路径（本地路径）

	// 访问配置
	AccessMode AccessMode        `json:"access_mode"` // 访问模式
	Tags       []string          `json:"tags"`        // 标签
	Metadata   map[string]string `json:"metadata"`    // 元数据

	// 目录相关（仅目录类型）
	FileCount int        `json:"file_count"` // 文件数量
	FileList  []FileInfo `json:"file_list"`  // 文件列表

	// 运行时信息
	Endpoint   string `json:"endpoint,omitempty"`    // 访问endpoint
	ProcessPID int    `json:"process_pid,omitempty"` // 相关进程PID
}

// DataGatewayWorkload 数据网关服务型workload（只读S3子集）
type DataGatewayWorkload struct {
	BaseWorkload

	// 服务配置
	ServicePort int    `json:"service_port"`
	ServiceHost string `json:"service_host"`

	// 网关配置
	BasePath  string `json:"base_path"` // 数据根目录（默认 storage.sqlite.data_path）
	Bucket    string `json:"bucket"`    // 命名空间（默认 cnet）
	ReadOnly  bool   `json:"read_only"`
	AuthToken string `json:"auth_token,omitempty"`

	// 运行时
	Endpoint   string `json:"endpoint,omitempty"`
	ProcessPID int    `json:"process_pid,omitempty"`
}

func NewDataGatewayWorkload(name string, req CreateWorkloadRequest) *DataGatewayWorkload {
	now := time.Now()
	w := &DataGatewayWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeDataGateway,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     make(map[string]interface{}),
		},
		ServicePort: 9080,
		ServiceHost: "127.0.0.1",
		BasePath:    "/tmp/cnet_data",
		Bucket:      "cnet",
		ReadOnly:    true,
	}
	if v, ok := req.Config["service_port"].(float64); ok {
		w.ServicePort = int(v)
	}
	if v, ok := req.Config["service_host"].(string); ok && v != "" {
		w.ServiceHost = v
	}
	if v, ok := req.Config["base_path"].(string); ok && v != "" {
		w.BasePath = v
	}
	if v, ok := req.Config["bucket"].(string); ok && v != "" {
		w.Bucket = v
	}
	if v, ok := req.Config["read_only"].(bool); ok {
		w.ReadOnly = v
	}
	if v, ok := req.Config["auth_token"].(string); ok && v != "" {
		w.AuthToken = v
	}
	return w
}

func (w *DataGatewayWorkload) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("data gateway name cannot be empty")
	}
	if w.ServicePort <= 0 {
		return fmt.Errorf("invalid service port")
	}
	if w.BasePath == "" {
		return fmt.Errorf("base_path cannot be empty")
	}
	return nil
}

// FileInfo 文件信息
type FileInfo struct {
	Path         string    `json:"path"`          // 相对路径
	Size         int64     `json:"size"`          // 文件大小
	Hash         string    `json:"hash"`          // 文件校验和
	ContentType  string    `json:"content_type"`  // MIME类型
	LastModified time.Time `json:"last_modified"` // 最后修改时间
}

// NewDataWorkload 创建数据workload
func NewDataWorkload(name string, req CreateWorkloadRequest) *DataWorkload {
	now := time.Now()

	workload := &DataWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeData,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     make(map[string]interface{}),
		},
		// 默认生成数据键，若后续从config覆盖则以config为准
		DataKey:        uuid.New().String(),
		DataType:       DataTypeFile,
		StorageType:    StorageTypeSQLite,
		StorageBackend: "sqlite",
		AccessMode:     AccessModeReadOnly,
		Tags:           []string{},
		Metadata:       make(map[string]string),
		FileList:       []FileInfo{},
	}

	// 从config中提取配置
	if v, ok := req.Config["data_key"].(string); ok && v != "" {
		workload.DataKey = v
	}
	if config, ok := req.Config["data_type"].(string); ok {
		workload.DataType = DataType(config)
	}
	if config, ok := req.Config["storage_type"].(string); ok {
		workload.StorageType = StorageType(config)
	}
	if config, ok := req.Config["storage_backend"].(string); ok {
		workload.StorageBackend = config
	}
	if config, ok := req.Config["access_mode"].(string); ok {
		workload.AccessMode = AccessMode(config)
	}
	if config, ok := req.Config["upload_method"].(string); ok {
		workload.UploadMethod = UploadMethod(config)
	}
	if config, ok := req.Config["source_url"].(string); ok {
		workload.SourceURL = config
	}
	if config, ok := req.Config["source_path"].(string); ok {
		workload.SourcePath = config
	}
	if config, ok := req.Config["tags"].([]interface{}); ok {
		for _, tag := range config {
			if tagStr, ok := tag.(string); ok {
				workload.Tags = append(workload.Tags, tagStr)
			}
		}
	}

	return workload
}

// Validate 验证数据workload配置
func (w *DataWorkload) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("data workload name cannot be empty")
	}

	if w.DataKey == "" {
		return fmt.Errorf("data key cannot be empty")
	}

	if w.Size < 0 {
		return fmt.Errorf("data size cannot be negative")
	}

	// 验证数据类型
	switch w.DataType {
	case DataTypeFile, DataTypeDirectory, DataTypeDataset, DataTypeModel, DataTypeConfig, DataTypeStream:
		// 有效类型
	default:
		return fmt.Errorf("invalid data type: %s", w.DataType)
	}

	// 验证存储类型
	switch w.StorageType {
	case StorageTypeLocal, StorageTypeSQLite, StorageTypeBadger, StorageTypeS3:
		// 有效类型
	default:
		return fmt.Errorf("invalid storage type: %s", w.StorageType)
	}

	// 验证访问模式
	switch w.AccessMode {
	case AccessModeReadOnly, AccessModeReadWrite, AccessModeExclusive:
		// 有效模式
	default:
		return fmt.Errorf("invalid access mode: %s", w.AccessMode)
	}

	// 验证上传方式
	switch w.UploadMethod {
	case UploadMethodFile, UploadMethodURL, UploadMethodPath, UploadMethodDirectory:
		// 有效方式
	default:
		return fmt.Errorf("invalid upload method: %s", w.UploadMethod)
	}

	// 目录类型需要文件列表
	if w.DataType == DataTypeDirectory && len(w.FileList) == 0 {
		return fmt.Errorf("directory data type requires file list")
	}

	return nil
}

// GetDataKey 获取数据键
func (w *DataWorkload) GetDataKey() string {
	return w.DataKey
}

// SetDataKey 设置数据键
func (w *DataWorkload) SetDataKey(key string) {
	w.DataKey = key
}

// GetEndpoint 获取访问endpoint
func (w *DataWorkload) GetEndpoint() string {
	return w.Endpoint
}

// SetEndpoint 设置访问endpoint
func (w *DataWorkload) SetEndpoint(endpoint string) {
	w.Endpoint = endpoint
}

// AddFile 添加文件信息（目录类型）
func (w *DataWorkload) AddFile(fileInfo FileInfo) {
	w.FileList = append(w.FileList, fileInfo)
	w.FileCount = len(w.FileList)
}

// GetTotalSize 获取总大小
func (w *DataWorkload) GetTotalSize() int64 {
	if w.DataType == DataTypeDirectory {
		total := int64(0)
		for _, file := range w.FileList {
			total += file.Size
		}
		return total
	}
	return w.Size
}

// IsDirectory 是否为目录类型
func (w *DataWorkload) IsDirectory() bool {
	return w.DataType == DataTypeDirectory
}

// GetFileCount 获取文件数量
func (w *DataWorkload) GetFileCount() int {
	if w.IsDirectory() {
		return w.FileCount
	}
	return 1
}
