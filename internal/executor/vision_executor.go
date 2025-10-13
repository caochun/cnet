package executor

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// VisionExecutor 计算机视觉执行器（简化版）
// 注意：这是不依赖OpenCV的简化实现
// 如需完整功能，需要安装 OpenCV 和 GoCV
type VisionExecutor struct {
	logger *logrus.Logger
	tasks  map[string]*visionTask
}

type visionTask struct {
	workload *workload.VisionWorkload
	results  interface{}
}

// NewVisionExecutor 创建Vision执行器
func NewVisionExecutor(logger *logrus.Logger) *VisionExecutor {
	return &VisionExecutor{
		logger: logger,
		tasks:  make(map[string]*visionTask),
	}
}

// Execute 执行Vision workload
func (e *VisionExecutor) Execute(ctx context.Context, w workload.Workload) error {
	vw, ok := w.(*workload.VisionWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected VisionWorkload")
	}
	
	e.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"task":        vw.Task,
		"input":       vw.InputPath,
		"model_type":  vw.ModelType,
	}).Info("Starting vision task (simplified implementation)")
	
	vw.SetStatus(workload.StatusRunning)
	
	// 创建任务记录
	task := &visionTask{
		workload: vw,
	}
	e.tasks[w.GetID()] = task
	
	// 验证输入文件
	if _, err := os.Stat(vw.InputPath); os.IsNotExist(err) {
		vw.SetStatus(workload.StatusFailed)
		return fmt.Errorf("input file not found: %s", vw.InputPath)
	}
	
	// 根据任务类型执行
	var err error
	switch vw.Task {
	case workload.TaskDetection:
		err = e.executeDetection(ctx, vw, task)
	case workload.TaskFaceDetection:
		err = e.executeFaceDetection(ctx, vw, task)
	case workload.TaskClassification:
		err = e.executeClassification(ctx, vw, task)
	default:
		err = fmt.Errorf("unsupported task type: %s (需要完整的GoCV支持)", vw.Task)
	}
	
	if err != nil {
		vw.SetStatus(workload.StatusFailed)
		e.logger.WithError(err).Error("Vision task failed")
		return err
	}
	
	vw.SetStatus(workload.StatusCompleted)
	e.logger.WithField("workload_id", w.GetID()).Info("Vision task completed")
	
	return nil
}

// executeDetection 执行目标检测（简化版）
func (e *VisionExecutor) executeDetection(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	// 读取图像
	img, err := e.loadImage(vw.InputPath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	
	bounds := img.Bounds()
	e.logger.WithFields(logrus.Fields{
		"width":  bounds.Dx(),
		"height": bounds.Dy(),
	}).Info("Image loaded")
	
	// 简化实现：返回模拟的检测结果
	results := []map[string]interface{}{
		{
			"class":      "object",
			"confidence": 0.85,
			"bbox":       map[string]int{"x": 100, "y": 100, "width": 200, "height": 150},
			"note":       "简化实现 - 需要完整GoCV支持以获得真实检测",
		},
	}
	
	vw.Results = results
	task.results = results
	
	// 如果指定了输出路径，保存标注图像
	if vw.OutputPath != "" {
		if err := e.saveAnnotatedImage(img, results, vw.OutputPath); err != nil {
			e.logger.WithError(err).Warn("Failed to save annotated image")
		} else {
			e.logger.WithField("output", vw.OutputPath).Info("Annotated image saved")
		}
	}
	
	e.logger.Info("Detection completed (simplified)")
	
	return nil
}

// executeFaceDetection 执行人脸检测（简化版）
func (e *VisionExecutor) executeFaceDetection(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	// 读取图像
	img, err := e.loadImage(vw.InputPath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	
	bounds := img.Bounds()
	e.logger.WithFields(logrus.Fields{
		"width":  bounds.Dx(),
		"height": bounds.Dy(),
	}).Info("Image loaded for face detection")
	
	// 简化实现：返回模拟结果
	results := []map[string]interface{}{
		{
			"id":         0,
			"class":      "face",
			"confidence": 0.92,
			"bbox":       map[string]int{"x": 150, "y": 80, "width": 100, "height": 120},
			"note":       "简化实现 - 需要完整GoCV + Haar Cascade",
		},
	}
	
	vw.Results = results
	task.results = results
	
	e.logger.WithField("faces", len(results)).Info("Face detection completed (simplified)")
	
	return nil
}

// executeClassification 执行图像分类
func (e *VisionExecutor) executeClassification(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	results := map[string]interface{}{
		"message":    "Classification task received",
		"note":       "需要完整GoCV支持",
		"input_path": vw.InputPath,
	}
	
	vw.Results = results
	task.results = results
	
	e.logger.Info("Classification completed (simplified)")
	return nil
}

// loadImage 加载图像
func (e *VisionExecutor) loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	// 根据文件扩展名解码
	ext := strings.ToLower(filepath.Ext(path))
	var img image.Image
	
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		// 尝试自动检测
		img, _, err = image.Decode(file)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	return img, nil
}

// saveAnnotatedImage 保存标注后的图像
func (e *VisionExecutor) saveAnnotatedImage(img image.Image, results []map[string]interface{}, outputPath string) error {
	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// 创建新图像用于绘制
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)
	
	// 绘制检测框（简化版，只画矩形）
	green := color.RGBA{0, 255, 0, 255}
	for _, result := range results {
		if bbox, ok := result["bbox"].(map[string]int); ok {
			x := bbox["x"]
			y := bbox["y"]
			w := bbox["width"]
			h := bbox["height"]
			
			// 画矩形框（简单实现）
			e.drawRect(dst, x, y, w, h, green)
		}
	}
	
	// 保存图像
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()
	
	// 根据扩展名编码
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 90})
	case ".png":
		err = png.Encode(outFile, dst)
	default:
		err = png.Encode(outFile, dst)
	}
	
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}
	
	return nil
}

// drawRect 绘制矩形（简单实现）
func (e *VisionExecutor) drawRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	// 画顶边
	for i := x; i < x+w; i++ {
		img.Set(i, y, c)
		img.Set(i, y+1, c)
	}
	// 画底边
	for i := x; i < x+w; i++ {
		img.Set(i, y+h, c)
		img.Set(i, y+h-1, c)
	}
	// 画左边
	for i := y; i < y+h; i++ {
		img.Set(x, i, c)
		img.Set(x+1, i, c)
	}
	// 画右边
	for i := y; i < y+h; i++ {
		img.Set(x+w, i, c)
		img.Set(x+w-1, i, c)
	}
}

// Stop 停止Vision workload
func (e *VisionExecutor) Stop(ctx context.Context, w workload.Workload) error {
	_, exists := e.tasks[w.GetID()]
	if !exists {
		return fmt.Errorf("vision task not found: %s", w.GetID())
	}
	
	delete(e.tasks, w.GetID())
	w.SetStatus(workload.StatusStopped)
	
	e.logger.WithField("workload_id", w.GetID()).Info("Vision task stopped")
	
	return nil
}

// GetLogs 获取Vision workload日志
func (e *VisionExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	vw, ok := w.(*workload.VisionWorkload)
	if !ok {
		return nil, fmt.Errorf("invalid workload type")
	}
	
	logs := []string{
		fmt.Sprintf("Vision Task: %s", vw.Task),
		fmt.Sprintf("Input: %s", vw.InputPath),
		fmt.Sprintf("Output: %s", vw.OutputPath),
		fmt.Sprintf("Model: %s (%s)", vw.ModelPath, vw.ModelType),
		fmt.Sprintf("Status: %s", vw.GetStatus()),
		"Note: 当前为简化实现，需要安装OpenCV和GoCV以获得完整功能",
	}
	
	if vw.Results != nil {
		logs = append(logs, fmt.Sprintf("Results: %+v", vw.Results))
	}
	
	return logs, nil
}

// GetStatus 获取Vision workload状态
func (e *VisionExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	return w.GetStatus(), nil
}
