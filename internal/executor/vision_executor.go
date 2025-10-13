package executor

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

// VisionExecutor 计算机视觉执行器（基于GoCV）
type VisionExecutor struct {
	logger *logrus.Logger
	tasks  map[string]*visionTask
}

type visionTask struct {
	workload *workload.VisionWorkload
	net      *gocv.Net
	cascade  *gocv.CascadeClassifier
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
	}).Info("Starting vision task")

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
	case workload.TaskTracking:
		err = e.executeTracking(ctx, vw, task)
	default:
		err = fmt.Errorf("unsupported task type: %s", vw.Task)
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

// executeDetection 执行目标检测
func (e *VisionExecutor) executeDetection(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	// 读取图像
	img := gocv.IMRead(vw.InputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to read image: %s", vw.InputPath)
	}
	defer img.Close()

	e.logger.WithFields(logrus.Fields{
		"width":  img.Cols(),
		"height": img.Rows(),
	}).Info("Image loaded")

	var results []map[string]interface{}
	var err error

	// 根据模型类型执行检测
	switch vw.ModelType {
	case workload.ModelDNN:
		results, err = e.detectWithDNN(img, vw, task)
	case workload.ModelYOLO:
		results, err = e.detectWithYOLO(img, vw, task)
	default:
		return fmt.Errorf("unsupported model type for detection: %s (支持: dnn, yolo)", vw.ModelType)
	}

	if err != nil {
		return err
	}

	// 保存结果
	vw.Results = results
	task.results = results

	// 如果指定了输出路径，保存标注后的图像
	if vw.OutputPath != "" {
		if err := e.saveAnnotatedImage(img, results, vw.OutputPath); err != nil {
			e.logger.WithError(err).Warn("Failed to save annotated image")
		} else {
			e.logger.WithField("output", vw.OutputPath).Info("Annotated image saved")
		}
	}

	e.logger.WithField("detections", len(results)).Info("Detection completed")

	return nil
}

// executeFaceDetection 执行人脸检测
func (e *VisionExecutor) executeFaceDetection(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	// 读取图像
	img := gocv.IMRead(vw.InputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to read image: %s", vw.InputPath)
	}
	defer img.Close()

	// 加载Haar级联分类器
	cascadePath := vw.ModelPath
	if cascadePath == "" {
		// 尝试常见的默认路径
		possiblePaths := []string{
			"/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
			"/opt/homebrew/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
			"/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				cascadePath = path
				break
			}
		}

		if cascadePath == "" {
			return fmt.Errorf("cascade model not found, please specify model_path")
		}
	}

	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(cascadePath) {
		return fmt.Errorf("failed to load cascade classifier: %s", cascadePath)
	}

	task.cascade = &classifier

	// 转换为灰度图
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// 检测人脸
	faces := classifier.DetectMultiScale(gray)

	e.logger.WithField("faces", len(faces)).Info("Face detection completed")

	// 转换结果
	var results []map[string]interface{}
	for i, face := range faces {
		results = append(results, map[string]interface{}{
			"id":         i,
			"class":      "face",
			"confidence": 1.0, // Cascade分类器不提供置信度
			"bbox": map[string]int{
				"x":      face.Min.X,
				"y":      face.Min.Y,
				"width":  face.Dx(),
				"height": face.Dy(),
			},
		})

		// 在图像上绘制矩形框
		gocv.Rectangle(&img, face, color.RGBA{0, 255, 0, 0}, 3)

		// 添加标签
		label := fmt.Sprintf("Face #%d", i+1)
		gocv.PutText(&img, label, image.Pt(face.Min.X, face.Min.Y-10),
			gocv.FontHersheyPlain, 1.5, color.RGBA{0, 255, 0, 0}, 2)
	}

	vw.Results = results
	task.results = results

	// 保存结果图像
	if vw.OutputPath != "" {
		if ok := gocv.IMWrite(vw.OutputPath, img); !ok {
			return fmt.Errorf("failed to write output image: %s", vw.OutputPath)
		}
		e.logger.WithField("output", vw.OutputPath).Info("Annotated image saved")
	}

	return nil
}

// executeClassification 执行图像分类
func (e *VisionExecutor) executeClassification(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	if vw.ModelPath == "" {
		return fmt.Errorf("model_path is required for classification")
	}

	// 读取图像
	img := gocv.IMRead(vw.InputPath, gocv.IMReadColor)
	if img.Empty() {
		return fmt.Errorf("failed to read image: %s", vw.InputPath)
	}
	defer img.Close()

	// 加载DNN模型
	net := gocv.ReadNet(vw.ModelPath, "")
	if net.Empty() {
		return fmt.Errorf("failed to load model: %s", vw.ModelPath)
	}
	defer net.Close()

	task.net = &net

	// 预处理图像
	blob := gocv.BlobFromImage(img, 1.0, image.Pt(224, 224), gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	// 设置输入并前向传播
	net.SetInput(blob, "")
	prob := net.Forward("")
	defer prob.Close()

	// 获取分类结果
	_, maxVal, _, maxLoc := gocv.MinMaxLoc(prob)

	results := map[string]interface{}{
		"class_id":   maxLoc.X,
		"confidence": maxVal,
		"input":      vw.InputPath,
	}

	vw.Results = results
	task.results = results

	e.logger.WithFields(logrus.Fields{
		"class_id":   maxLoc.X,
		"confidence": maxVal,
	}).Info("Classification completed")

	return nil
}

// executeTracking 执行目标跟踪
func (e *VisionExecutor) executeTracking(ctx context.Context, vw *workload.VisionWorkload, task *visionTask) error {
	// 检查输入是否是视频
	if !e.isVideoFile(vw.InputPath) {
		return fmt.Errorf("tracking requires video input")
	}

	// 打开视频
	video, err := gocv.OpenVideoCapture(vw.InputPath)
	if err != nil {
		return fmt.Errorf("failed to open video: %w", err)
	}
	defer video.Close()

	// 获取视频信息
	fps := video.Get(gocv.VideoCaptureFPS)
	frameCount := int(video.Get(gocv.VideoCaptureFrameCount))

	e.logger.WithFields(logrus.Fields{
		"fps":         fps,
		"frame_count": frameCount,
	}).Info("Video opened")

	// 简化实现：只返回视频信息
	// 完整的跟踪功能需要更新版本的GoCV或自定义实现
	vw.Results = map[string]interface{}{
		"total_frames": frameCount,
		"fps":          fps,
		"note":         "目标跟踪功能需要GoCV tracker模块支持",
	}
	task.results = vw.Results

	e.logger.Info("Video info extracted (tracking not fully implemented)")

	return nil
}

// detectWithDNN 使用DNN模型检测
func (e *VisionExecutor) detectWithDNN(img gocv.Mat, vw *workload.VisionWorkload, task *visionTask) ([]map[string]interface{}, error) {
	if vw.ModelPath == "" {
		return nil, fmt.Errorf("model_path is required for DNN detection")
	}

	// 检查是否有配置文件
	configPath := vw.Config["config_path"]
	if configPath == "" {
		// 尝试同名的.pbtxt或.prototxt文件
		basePath := strings.TrimSuffix(vw.ModelPath, filepath.Ext(vw.ModelPath))
		for _, ext := range []string{".pbtxt", ".prototxt", ".txt"} {
			testPath := basePath + ext
			if _, err := os.Stat(testPath); err == nil {
				configPath = testPath
				break
			}
		}
	}

	// 加载DNN模型
	net := gocv.ReadNet(vw.ModelPath, configPath)
	if net.Empty() {
		return nil, fmt.Errorf("failed to load DNN model from: %s", vw.ModelPath)
	}
	defer net.Close()

	task.net = &net

	// 设置后端（优先使用CUDA，fallback到CPU）
	net.SetPreferableBackend(gocv.NetBackendDefault)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	// 准备输入blob（根据模型调整尺寸）
	inputSize := image.Pt(300, 300) // 默认
	if size, ok := vw.Config["input_size"]; ok {
		// 解析自定义尺寸，格式如 "416x416"
		var w, h int
		if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err == nil {
			inputSize = image.Pt(w, h)
		}
	}

	blob := gocv.BlobFromImage(img, 1.0/255.0, inputSize,
		gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// 设置输入
	net.SetInput(blob, "")

	// 前向传播
	prob := net.Forward("")
	defer prob.Close()

	// 解析检测结果
	results := e.parseDNNOutput(prob, img.Cols(), img.Rows(), vw.Confidence)

	e.logger.WithField("detections", len(results)).Info("DNN detection completed")

	return results, nil
}

// detectWithYOLO 使用YOLO模型检测
func (e *VisionExecutor) detectWithYOLO(img gocv.Mat, vw *workload.VisionWorkload, task *visionTask) ([]map[string]interface{}, error) {
	if vw.ModelPath == "" {
		return nil, fmt.Errorf("model_path is required for YOLO detection")
	}

	// 加载YOLO模型
	// 支持 .weights + .cfg 或 .onnx 格式
	var net gocv.Net
	configPath := vw.Config["config_path"]

	if strings.HasSuffix(vw.ModelPath, ".onnx") || strings.HasSuffix(vw.ModelPath, ".pb") {
		net = gocv.ReadNet(vw.ModelPath, "")
	} else {
		// Darknet格式（.weights + .cfg）
		if configPath == "" {
			return nil, fmt.Errorf("config_path is required for Darknet YOLO models")
		}
		net = gocv.ReadNet(vw.ModelPath, configPath)
	}

	if net.Empty() {
		return nil, fmt.Errorf("failed to load YOLO model")
	}
	defer net.Close()

	task.net = &net

	// 设置后端
	net.SetPreferableBackend(gocv.NetBackendDefault)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	// 准备输入blob（YOLO通常使用416x416或608x608）
	inputSize := 416
	if size, ok := vw.Config["input_size"]; ok {
		fmt.Sscanf(size, "%d", &inputSize)
	}

	blob := gocv.BlobFromImage(img, 1.0/255.0, image.Pt(inputSize, inputSize),
		gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// 设置输入
	net.SetInput(blob, "")

	// 获取输出层名称
	layerNames := net.GetLayerNames()
	outLayers := []string{}
	for _, name := range layerNames {
		outLayers = append(outLayers, name)
	}

	// 如果没有输出层，使用默认前向传播
	var probs []gocv.Mat
	if len(outLayers) == 0 {
		prob := net.Forward("")
		probs = []gocv.Mat{prob}
	} else {
		// 前向传播到输出层
		prob := net.Forward("")
		probs = []gocv.Mat{prob}
	}
	defer func() {
		for _, prob := range probs {
			prob.Close()
		}
	}()

	// 解析YOLO输出
	results := e.parseYOLOOutput(probs, img.Cols(), img.Rows(), vw.Confidence, vw.NMSThreshold)

	e.logger.WithField("detections", len(results)).Info("YOLO detection completed")

	return results, nil
}

// parseDNNOutput 解析DNN输出
func (e *VisionExecutor) parseDNNOutput(output gocv.Mat, width, height int, confidence float32) []map[string]interface{} {
	var results []map[string]interface{}

	// 标准的SSD输出格式：[1, 1, N, 7]
	// 每个检测：[image_id, class_id, confidence, x_min, y_min, x_max, y_max]
	if output.Total() == 0 {
		return results
	}

	for i := 0; i < output.Rows(); i++ {
		conf := output.GetFloatAt(i, 2)
		if conf > confidence {
			classID := int(output.GetFloatAt(i, 1))
			xMin := int(output.GetFloatAt(i, 3) * float32(width))
			yMin := int(output.GetFloatAt(i, 4) * float32(height))
			xMax := int(output.GetFloatAt(i, 5) * float32(width))
			yMax := int(output.GetFloatAt(i, 6) * float32(height))

			results = append(results, map[string]interface{}{
				"class_id":   classID,
				"class":      fmt.Sprintf("class_%d", classID),
				"confidence": conf,
				"bbox": map[string]int{
					"x":      xMin,
					"y":      yMin,
					"width":  xMax - xMin,
					"height": yMax - yMin,
				},
			})
		}
	}

	return results
}

// parseYOLOOutput 解析YOLO输出
func (e *VisionExecutor) parseYOLOOutput(outputs []gocv.Mat, width, height int, confidence, nmsThreshold float32) []map[string]interface{} {
	var classIDs []int
	var confidences []float32
	var boxes []image.Rectangle

	// 遍历所有输出层
	for _, output := range outputs {
		for i := 0; i < output.Rows(); i++ {
			// YOLO输出格式：[center_x, center_y, width, height, objectness, class_scores...]
			objectness := output.GetFloatAt(i, 4)

			if objectness > confidence {
				// 找到最高分类分数
				var maxScore float32
				var maxClassID int

				for j := 5; j < output.Cols(); j++ {
					score := output.GetFloatAt(i, j) * objectness
					if score > maxScore {
						maxScore = score
						maxClassID = j - 5
					}
				}

				if maxScore > confidence {
					centerX := int(output.GetFloatAt(i, 0) * float32(width))
					centerY := int(output.GetFloatAt(i, 1) * float32(height))
					w := int(output.GetFloatAt(i, 2) * float32(width))
					h := int(output.GetFloatAt(i, 3) * float32(height))

					x := centerX - w/2
					y := centerY - h/2

					classIDs = append(classIDs, maxClassID)
					confidences = append(confidences, maxScore)
					boxes = append(boxes, image.Rect(x, y, x+w, y+h))
				}
			}
		}
	}

	// NMS（非极大值抑制）
	indices := gocv.NMSBoxes(boxes, confidences, confidence, nmsThreshold)

	var results []map[string]interface{}
	for _, idx := range indices {
		box := boxes[idx]
		results = append(results, map[string]interface{}{
			"class_id":   classIDs[idx],
			"class":      fmt.Sprintf("class_%d", classIDs[idx]),
			"confidence": confidences[idx],
			"bbox": map[string]int{
				"x":      box.Min.X,
				"y":      box.Min.Y,
				"width":  box.Dx(),
				"height": box.Dy(),
			},
		})
	}

	return results
}

// saveAnnotatedImage 保存标注后的图像
func (e *VisionExecutor) saveAnnotatedImage(img gocv.Mat, results []map[string]interface{}, outputPath string) error {
	// 确保输出目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 在图像上绘制检测框
	for _, result := range results {
		if bbox, ok := result["bbox"].(map[string]int); ok {
			x := bbox["x"]
			y := bbox["y"]
			w := bbox["width"]
			h := bbox["height"]

			rect := image.Rect(x, y, x+w, y+h)
			gocv.Rectangle(&img, rect, color.RGBA{0, 255, 0, 0}, 2)

			// 添加标签
			label := "object"
			if class, ok := result["class"].(string); ok {
				label = class
			}
			if conf, ok := result["confidence"].(float32); ok {
				label = fmt.Sprintf("%s: %.2f", label, conf)
			}

			gocv.PutText(&img, label, image.Pt(x, y-10),
				gocv.FontHersheyPlain, 1.2, color.RGBA{0, 255, 0, 0}, 2)
		}
	}

	// 保存图像
	if ok := gocv.IMWrite(outputPath, img); !ok {
		return fmt.Errorf("failed to write image: %s", outputPath)
	}

	return nil
}

// isVideoFile 检查是否是视频文件
func (e *VisionExecutor) isVideoFile(path string) bool {
	videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv"}
	ext := strings.ToLower(filepath.Ext(path))
	for _, videoExt := range videoExts {
		if ext == videoExt {
			return true
		}
	}
	return false
}

// Stop 停止Vision workload
func (e *VisionExecutor) Stop(ctx context.Context, w workload.Workload) error {
	task, exists := e.tasks[w.GetID()]
	if !exists {
		return fmt.Errorf("vision task not found: %s", w.GetID())
	}

	// 清理资源
	if task.net != nil {
		task.net.Close()
	}
	if task.cascade != nil {
		task.cascade.Close()
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
		"=== Vision Task Logs ===",
		fmt.Sprintf("Task Type: %s", vw.Task),
		fmt.Sprintf("Input: %s", vw.InputPath),
		fmt.Sprintf("Output: %s", vw.OutputPath),
		fmt.Sprintf("Model: %s (%s)", vw.ModelPath, vw.ModelType),
		fmt.Sprintf("Confidence Threshold: %.2f", vw.Confidence),
		fmt.Sprintf("NMS Threshold: %.2f", vw.NMSThreshold),
		fmt.Sprintf("Status: %s", vw.GetStatus()),
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
