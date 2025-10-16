package storage

import (
	"cnet/internal/workload"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteBackend SQLite存储后端
type SQLiteBackend struct {
	db       *sql.DB
	dataPath string
}

// NewSQLiteBackend 创建SQLite存储后端
func NewSQLiteBackend(dbPath, dataPath string) (*SQLiteBackend, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &SQLiteBackend{
		db:       db,
		dataPath: dataPath,
	}

	// 初始化数据库表
	if err := backend.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return backend, nil
}

// initTables 初始化数据库表
func (s *SQLiteBackend) initTables() error {
	// 创建数据对象表
	createObjectsTable := `
	CREATE TABLE IF NOT EXISTS objects (
		data_key TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		data_type TEXT NOT NULL,
		content_type TEXT,
		size INTEGER NOT NULL,
		hash TEXT,
		file_path TEXT,
		directory_path TEXT,
		file_count INTEGER DEFAULT 0,
		access_mode TEXT DEFAULT 'readonly',
		tags TEXT,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	// 创建文件列表表（用于目录类型）
	createFilesTable := `
	CREATE TABLE IF NOT EXISTS object_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		data_key TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		file_hash TEXT,
		content_type TEXT,
		last_modified DATETIME,
		FOREIGN KEY (data_key) REFERENCES objects (data_key) ON DELETE CASCADE
	)`

	// 创建索引
	createIndexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_objects_name ON objects (name)",
		"CREATE INDEX IF NOT EXISTS idx_objects_data_type ON objects (data_type)",
		"CREATE INDEX IF NOT EXISTS idx_objects_created_at ON objects (created_at)",
		"CREATE INDEX IF NOT EXISTS idx_object_files_data_key ON object_files (data_key)",
	}

	// 执行创建语句
	statements := []string{createObjectsTable, createFilesTable}
	statements = append(statements, createIndexes...)

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}

// Upload 上传数据
func (s *SQLiteBackend) Upload(ctx context.Context, data *workload.DataWorkload) error {
	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 准备数据
	tagsJSON, _ := json.Marshal(data.Tags)
	metadataJSON, _ := json.Marshal(data.Metadata)

	// 插入或更新对象记录
	upsertObject := `
	INSERT OR REPLACE INTO objects (
		data_key, name, data_type, content_type, size, hash,
		file_path, directory_path, file_count, access_mode,
		tags, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	_, err = tx.Exec(upsertObject,
		data.DataKey, data.Name, string(data.DataType), data.ContentType,
		data.Size, data.Hash, data.FilePath, data.DirectoryPath,
		data.FileCount, string(data.AccessMode), string(tagsJSON),
		string(metadataJSON), data.CreatedAt, now)
	if err != nil {
		return fmt.Errorf("failed to upsert object: %w", err)
	}

	// 如果是目录类型，插入文件列表
	if data.IsDirectory() {
		// 先删除旧的文件记录
		_, err = tx.Exec("DELETE FROM object_files WHERE data_key = ?", data.DataKey)
		if err != nil {
			return fmt.Errorf("failed to delete old file records: %w", err)
		}

		// 插入新的文件记录
		insertFile := `
		INSERT INTO object_files (
			data_key, file_path, file_size, file_hash, content_type, last_modified
		) VALUES (?, ?, ?, ?, ?, ?)`

		for _, file := range data.FileList {
			_, err = tx.Exec(insertFile,
				data.DataKey, file.Path, file.Size, file.Hash,
				file.ContentType, file.LastModified)
			if err != nil {
				return fmt.Errorf("failed to insert file record: %w", err)
			}
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Download 下载数据
func (s *SQLiteBackend) Download(ctx context.Context, data *workload.DataWorkload) error {
	// 检查数据是否存在
	exists, err := s.Exists(ctx, data.DataKey)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("data not found: %s", data.DataKey)
	}

	// 对于文件类型，直接返回文件路径
	if data.DataType == workload.DataTypeFile {
		// 文件已存储在本地，无需额外下载
		return nil
	}

	// 对于目录类型，文件已存储在本地
	if data.DataType == workload.DataTypeDirectory {
		// 目录已存储在本地，无需额外下载
		return nil
	}

	return nil
}

// Delete 删除数据
func (s *SQLiteBackend) Delete(ctx context.Context, data *workload.DataWorkload) error {
	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 删除文件记录（外键约束会自动删除）
	_, err = tx.Exec("DELETE FROM object_files WHERE data_key = ?", data.DataKey)
	if err != nil {
		return fmt.Errorf("failed to delete file records: %w", err)
	}

	// 删除对象记录
	_, err = tx.Exec("DELETE FROM objects WHERE data_key = ?", data.DataKey)
	if err != nil {
		return fmt.Errorf("failed to delete object record: %w", err)
	}

	// 删除物理文件
	if data.FilePath != "" {
		if err := os.RemoveAll(data.FilePath); err != nil {
			// 记录错误但不阻止删除
			fmt.Printf("Warning: failed to remove file %s: %v\n", data.FilePath, err)
		}
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetInfo 获取数据信息
func (s *SQLiteBackend) GetInfo(ctx context.Context, data *workload.DataWorkload) (*DataInfo, error) {
	query := `
	SELECT data_key, name, data_type, content_type, size, hash,
		   file_path, directory_path, file_count, access_mode,
		   tags, metadata, created_at, updated_at
	FROM objects WHERE data_key = ?`

	var info DataInfo
	var tagsJSON, metadataJSON string
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, data.DataKey).Scan(
		&info.DataKey, &info.Name, &info.DataType, &info.ContentType,
		&info.Size, &info.Hash, &info.FilePath, &info.DirectoryPath,
		&info.FileCount, &info.AccessMode, &tagsJSON, &metadataJSON,
		&createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data not found: %s", data.DataKey)
		}
		return nil, fmt.Errorf("failed to query data info: %w", err)
	}

	// 解析JSON字段
	json.Unmarshal([]byte(tagsJSON), &info.Tags)
	json.Unmarshal([]byte(metadataJSON), &info.Metadata)
	info.CreatedAt = createdAt
	info.UpdatedAt = updatedAt

	// 如果是目录类型，获取文件列表
	if info.DataType == workload.DataTypeDirectory {
		files, err := s.getFileList(ctx, data.DataKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get file list: %w", err)
		}
		// 注意：这里需要扩展DataInfo结构来包含文件列表
		_ = files // 暂时忽略，后续可以扩展
	}

	return &info, nil
}

// getFileList 获取文件列表
func (s *SQLiteBackend) getFileList(ctx context.Context, dataKey string) ([]workload.FileInfo, error) {
	query := `
	SELECT file_path, file_size, file_hash, content_type, last_modified
	FROM object_files WHERE data_key = ? ORDER BY file_path`

	rows, err := s.db.QueryContext(ctx, query, dataKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []workload.FileInfo
	for rows.Next() {
		var file workload.FileInfo
		var lastModified time.Time

		err := rows.Scan(&file.Path, &file.Size, &file.Hash,
			&file.ContentType, &lastModified)
		if err != nil {
			return nil, err
		}

		file.LastModified = lastModified
		files = append(files, file)
	}

	return files, nil
}

// List 列出数据
func (s *SQLiteBackend) List(ctx context.Context, prefix string) ([]*DataInfo, error) {
	query := `
	SELECT data_key, name, data_type, content_type, size, hash,
		   file_path, directory_path, file_count, access_mode,
		   tags, metadata, created_at, updated_at
	FROM objects`
	args := []interface{}{}

	if prefix != "" {
		query += " WHERE data_key LIKE ?"
		args = append(args, prefix+"%")
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query objects: %w", err)
	}
	defer rows.Close()

	var infos []*DataInfo
	for rows.Next() {
		var info DataInfo
		var tagsJSON, metadataJSON string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&info.DataKey, &info.Name, &info.DataType, &info.ContentType,
			&info.Size, &info.Hash, &info.FilePath, &info.DirectoryPath,
			&info.FileCount, &info.AccessMode, &tagsJSON, &metadataJSON,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan object: %w", err)
		}

		// 解析JSON字段
		json.Unmarshal([]byte(tagsJSON), &info.Tags)
		json.Unmarshal([]byte(metadataJSON), &info.Metadata)
		info.CreatedAt = createdAt
		info.UpdatedAt = updatedAt

		infos = append(infos, &info)
	}

	return infos, nil
}

// Exists 检查数据是否存在
func (s *SQLiteBackend) Exists(ctx context.Context, dataKey string) (bool, error) {
	query := "SELECT 1 FROM objects WHERE data_key = ? LIMIT 1"
	var exists int
	err := s.db.QueryRowContext(ctx, query, dataKey).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return true, nil
}

// SaveFile 保存文件到本地存储
func (s *SQLiteBackend) SaveFile(ctx context.Context, data *workload.DataWorkload, reader io.Reader) error {
	// 确保目录存在
	dir := filepath.Dir(data.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(data.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 计算hash并复制数据
	hash := md5.New()
	multiWriter := io.MultiWriter(file, hash)

	_, err = io.Copy(multiWriter, reader)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// 设置hash
	data.Hash = hex.EncodeToString(hash.Sum(nil))

	return nil
}

// SaveDirectory 保存目录到本地存储
func (s *SQLiteBackend) SaveDirectory(ctx context.Context, data *workload.DataWorkload, files map[string]io.Reader) error {
	// 确保目录存在
	if err := os.MkdirAll(data.DirectoryPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 保存每个文件
	for relativePath, reader := range files {
		filePath := filepath.Join(data.DirectoryPath, relativePath)

		// 确保子目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create subdirectory: %w", err)
		}

		// 创建文件
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", relativePath, err)
		}

		// 复制数据
		_, err = io.Copy(file, reader)
		file.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", relativePath, err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (s *SQLiteBackend) Close() error {
	return s.db.Close()
}
