package executor

import (
	"cnet/internal/storage"
	"cnet/internal/workload"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// DataExecutor 数据执行器
type DataExecutor struct {
	storageManager *storage.StorageManager
	logger         *logrus.Logger
}

// NewDataExecutor 创建数据执行器
func NewDataExecutor(storageManager *storage.StorageManager, logger *logrus.Logger) *DataExecutor {
	return &DataExecutor{
		storageManager: storageManager,
		logger:         logger,
	}
}

// Init 初始化数据执行器
func (e *DataExecutor) Init(ctx context.Context) error {
	e.logger.Info("DataExecutor initialized")
	return nil
}

// Execute 执行数据workload
func (e *DataExecutor) Execute(ctx context.Context, w workload.Workload) error {
	dataWorkload, ok := w.(*workload.DataWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type for DataExecutor")
	}

	e.logger.WithFields(logrus.Fields{
		"workload_id":   dataWorkload.GetID(),
		"data_key":      dataWorkload.GetDataKey(),
		"data_type":     dataWorkload.DataType,
		"upload_method": dataWorkload.UploadMethod,
	}).Info("开始执行数据workload")

	// 根据上传方式处理数据
	var err error
	switch dataWorkload.UploadMethod {
	case workload.UploadMethodFile:
		err = e.handleFileUpload(ctx, dataWorkload)
	case workload.UploadMethodURL:
		err = e.handleURLDownload(ctx, dataWorkload)
	case workload.UploadMethodPath:
		err = e.handleLocalPath(ctx, dataWorkload)
	case workload.UploadMethodDirectory:
		err = e.handleDirectoryUpload(ctx, dataWorkload)
	default:
		err = fmt.Errorf("unsupported upload method: %s", dataWorkload.UploadMethod)
	}

	if err != nil {
		dataWorkload.SetStatus(workload.StatusFailed)
		e.logger.WithError(err).WithFields(logrus.Fields{
			"workload_id": dataWorkload.GetID(),
			"data_key":    dataWorkload.GetDataKey(),
		}).Error("数据workload执行失败")
		return err
	}

	dataWorkload.SetStatus(workload.StatusCompleted)
	e.logger.WithFields(logrus.Fields{
		"workload_id": dataWorkload.GetID(),
		"data_key":    dataWorkload.GetDataKey(),
	}).Info("数据workload执行完成")

	return nil
}

// Stop 停止数据workload
func (e *DataExecutor) Stop(ctx context.Context, w workload.Workload) error {
	dataWorkload, ok := w.(*workload.DataWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type for DataExecutor")
	}

	e.logger.WithFields(logrus.Fields{
		"workload_id": dataWorkload.GetID(),
		"data_key":    dataWorkload.GetDataKey(),
	}).Info("停止数据workload")

	// 删除数据
	return e.storageManager.Delete(ctx, dataWorkload)
}

// GetLogs 获取日志
func (e *DataExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	// 数据workload通常没有持续运行的日志
	return []string{"数据workload已执行完成"}, nil
}

// GetStatus 获取状态
func (e *DataExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	dataWorkload, ok := w.(*workload.DataWorkload)
	if !ok {
		return workload.StatusFailed, fmt.Errorf("invalid workload type for DataExecutor")
	}

	// 检查数据是否存在
	exists, err := e.storageManager.Exists(ctx, dataWorkload.GetDataKey())
	if err != nil {
		return workload.StatusFailed, err
	}

	if exists {
		return workload.StatusCompleted, nil
	}

	return workload.StatusFailed, nil
}

// handleFileUpload 处理文件上传
func (e *DataExecutor) handleFileUpload(ctx context.Context, data *workload.DataWorkload) error {
	// 文件上传在API层已经保存到临时路径，这里需要移动到最终位置
	if data.FilePath == "" {
		return fmt.Errorf("file path not set for file upload")
	}

	// 检查文件是否存在
	if _, err := os.Stat(data.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("uploaded file not found: %s", data.FilePath)
	}

	// 计算文件hash
	hash, err := e.calculateFileHash(data.FilePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}
	data.Hash = hash

	// 获取文件信息
	fileInfo, err := os.Stat(data.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	data.Size = fileInfo.Size()

	// 设置内容类型
	if data.ContentType == "" {
		data.ContentType = getContentType(data.FilePath)
	}

	// 上传到存储管理器
	return e.storageManager.Upload(ctx, data)
}

// handleURLDownload 处理URL下载
func (e *DataExecutor) handleURLDownload(ctx context.Context, data *workload.DataWorkload) error {
	e.logger.WithFields(logrus.Fields{
		"data_key": data.GetDataKey(),
		"url":      data.SourceURL,
	}).Info("开始从URL下载数据")

	// 下载文件
	resp, err := http.Get(data.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 设置内容类型
	if data.ContentType == "" {
		data.ContentType = resp.Header.Get("Content-Type")
	}

	// 设置文件大小
	if data.Size == 0 {
		data.Size = resp.ContentLength
	}

	// 生成文件路径
	if data.FilePath == "" {
		fileName := filepath.Base(data.SourceURL)
		if data.Metadata["file_name"] != "" {
			fileName = data.Metadata["file_name"]
		}
		// 使用默认数据路径
		data.FilePath = filepath.Join("/tmp/cnet_data", data.GetDataKey(), fileName)
	}

	// 保存文件
	backend, err := e.storageManager.GetDefaultBackend()
	if err != nil {
		return fmt.Errorf("failed to get storage backend: %w", err)
	}

	sqliteBackend, ok := backend.(*storage.SQLiteBackend)
	if !ok {
		return fmt.Errorf("expected SQLiteBackend, got %T", backend)
	}

	err = sqliteBackend.SaveFile(ctx, data, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// 上传到存储管理器
	return e.storageManager.Upload(ctx, data)
}

// handleLocalPath 处理本地路径
func (e *DataExecutor) handleLocalPath(ctx context.Context, data *workload.DataWorkload) error {
	e.logger.WithFields(logrus.Fields{
		"data_key":    data.GetDataKey(),
		"source_path": data.SourcePath,
	}).Info("开始处理本地路径数据")

	// 检查源文件是否存在
	fileInfo, err := os.Stat(data.SourcePath)
	if err != nil {
		return fmt.Errorf("source file not found: %w", err)
	}

	// 设置文件信息
	if data.Size == 0 {
		data.Size = fileInfo.Size()
	}
	if data.ContentType == "" {
		data.ContentType = getContentType(data.SourcePath)
	}

	// 生成目标路径
	if data.FilePath == "" {
		fileName := filepath.Base(data.SourcePath)
		// 使用默认数据路径
		data.FilePath = filepath.Join("/tmp/cnet_data", data.GetDataKey(), fileName)
	}

	// 复制文件
	err = e.copyFile(data.SourcePath, data.FilePath)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// 计算文件hash
	hash, err := e.calculateFileHash(data.FilePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}
	data.Hash = hash

	// 上传到存储管理器
	return e.storageManager.Upload(ctx, data)
}

// handleDirectoryUpload 处理目录上传
func (e *DataExecutor) handleDirectoryUpload(ctx context.Context, data *workload.DataWorkload) error {
	e.logger.WithFields(logrus.Fields{
		"data_key":       data.GetDataKey(),
		"file_count":     data.FileCount,
		"directory_path": data.DirectoryPath,
	}).Info("开始处理目录上传")

	// 目录上传应该在API层处理，这里只是保存到存储
	return e.storageManager.Upload(ctx, data)
}

// copyFile 复制文件
func (e *DataExecutor) copyFile(src, dst string) error {
	// 确保目标目录存在
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// 复制数据
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

// calculateFileHash 计算文件hash
func (e *DataExecutor) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getContentType 根据文件扩展名获取内容类型
func getContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".tar", ".gz":
		return "application/x-tar"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".onnx":
		return "application/octet-stream"
	case ".pt", ".pth":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}
