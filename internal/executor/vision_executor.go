package executor

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

// VisionExecutor 计算机视觉执行器（基于GoCV）
type VisionExecutor struct {
	logger       *logrus.Logger
	tasks        map[string]*visionTask
	modelCache   map[string]*gocv.Net               // 模型缓存
	cascadeCache map[string]*gocv.CascadeClassifier // Cascade分类器缓存
	mu           sync.RWMutex                       // 缓存锁
}

type visionTask struct {
	workload     *workload.VisionWorkload
	useCachedNet bool // 是否使用缓存的网络
	results      interface{}
}

// NewVisionExecutor 创建Vision执行器
// 模型会在首次使用时自动加载并缓存到内存
func NewVisionExecutor(logger *logrus.Logger) *VisionExecutor {
	logger.Info("Vision Executor initialized (models will be cached on first use)")
	return &VisionExecutor{
		logger:       logger,
		tasks:        make(map[string]*visionTask),
		modelCache:   make(map[string]*gocv.Net),
		cascadeCache: make(map[string]*gocv.CascadeClassifier),
	}
}

// getOrLoadModel 获取或加载模型（带缓存）
func (e *VisionExecutor) getOrLoadModel(modelPath, configPath string) (*gocv.Net, bool, error) {
	cacheKey := modelPath
	if configPath != "" {
		cacheKey = modelPath + "|" + configPath
	}

	// 先尝试从缓存获取
	e.mu.RLock()
	if cached, exists := e.modelCache[cacheKey]; exists {
		e.mu.RUnlock()
		e.logger.WithField("model", modelPath).Debug("Using cached model")
		return cached, true, nil // true表示使用缓存
	}
	e.mu.RUnlock()

	// 缓存未命中，加载模型
	e.mu.Lock()
	defer e.mu.Unlock()

	// 双重检查（避免并发加载）
	if cached, exists := e.modelCache[cacheKey]; exists {
		e.logger.WithField("model", modelPath).Debug("Using cached model (double check)")
		return cached, true, nil
	}

	e.logger.WithField("model", modelPath).Info("Loading model (cache miss)...")

	var net gocv.Net
	if configPath != "" {
		net = gocv.ReadNet(modelPath, configPath)
	} else {
		net = gocv.ReadNet(modelPath, "")
	}

	if net.Empty() {
		return nil, false, fmt.Errorf("failed to load model: %s", modelPath)
	}

	// 设置后端
	net.SetPreferableBackend(gocv.NetBackendDefault)
	net.SetPreferableTarget(gocv.NetTargetCPU)

	// 缓存模型
	e.modelCache[cacheKey] = &net

	e.logger.WithField("model", modelPath).Info("Model loaded and cached")

	return &net, true, nil
}

// getOrLoadCascade 获取或加载Cascade（带缓存）
func (e *VisionExecutor) getOrLoadCascade(cascadePath string) (*gocv.CascadeClassifier, bool, error) {
	// 先尝试从缓存获取
	e.mu.RLock()
	if cached, exists := e.cascadeCache[cascadePath]; exists {
		e.mu.RUnlock()
		e.logger.WithField("cascade", cascadePath).Debug("Using cached cascade")
		return cached, true, nil
	}
	e.mu.RUnlock()

	// 缓存未命中，加载分类器
	e.mu.Lock()
	defer e.mu.Unlock()

	// 双重检查
	if cached, exists := e.cascadeCache[cascadePath]; exists {
		return cached, true, nil
	}

	e.logger.WithField("cascade", cascadePath).Info("Loading cascade (cache miss)...")

	classifier := gocv.NewCascadeClassifier()
	if !classifier.Load(cascadePath) {
		return nil, false, fmt.Errorf("failed to load cascade: %s", cascadePath)
	}

	e.cascadeCache[cascadePath] = &classifier

	e.logger.WithField("cascade", cascadePath).Info("Cascade loaded and cached")

	return &classifier, true, nil
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

	// 使用缓存加载Cascade
	classifier, useCached, err := e.getOrLoadCascade(cascadePath)
	if err != nil {
		return err
	}

	// 注意：不要defer Close缓存的分类器！
	task.useCachedNet = useCached

	// 转换为灰度图
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// 检测人脸（使用缓存的分类器）
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

	// 使用缓存加载DNN模型
	net, useCached, err := e.getOrLoadModel(vw.ModelPath, "")
	if err != nil {
		return err
	}

	task.useCachedNet = useCached

	// 预处理图像
	blob := gocv.BlobFromImage(img, 1.0, image.Pt(224, 224), gocv.NewScalar(0, 0, 0, 0), false, false)
	defer blob.Close()

	// 设置输入并前向传播（使用缓存的网络）
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

	// 使用缓存加载DNN模型
	net, useCached, err := e.getOrLoadModel(vw.ModelPath, configPath)
	if err != nil {
		return nil, err
	}

	task.useCachedNet = useCached

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

	// 准备配置路径
	configPath := vw.Config["config_path"]

	// Darknet格式需要配置文件
	if !strings.HasSuffix(vw.ModelPath, ".onnx") && !strings.HasSuffix(vw.ModelPath, ".pb") {
		if configPath == "" {
			return nil, fmt.Errorf("config_path is required for Darknet YOLO models")
		}
	}

	// 使用缓存加载YOLO模型
	net, useCached, err := e.getOrLoadModel(vw.ModelPath, configPath)
	if err != nil {
		return nil, err
	}

	task.useCachedNet = useCached

	// 准备输入blob（YOLO v5/v8/v11 使用640x640）
	inputSize := 640
	if size, ok := vw.Config["input_size"]; ok {
		fmt.Sscanf(size, "%d", &inputSize)
	}

	e.logger.WithField("input_size", inputSize).Debug("Creating blob")

	blob := gocv.BlobFromImage(img, 1.0/255.0, image.Pt(inputSize, inputSize),
		gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// 设置输入
	net.SetInput(blob, "")

	e.logger.Debug("Running forward pass")

	// YOLOv5s ONNX模型的前向传播
	var probs []gocv.Mat

	// 获取输出层名称
	layerNames := net.GetLayerNames()
	e.logger.WithField("layer_names", layerNames).Debug("YOLO model layers")

	// YOLOv5s ONNX模型使用默认前向传播
	// 输出格式通常是 [1, 25200, 85] (batch, detections, features)
	prob := net.Forward("")
	probs = []gocv.Mat{prob}

	e.logger.WithFields(logrus.Fields{
		"rows":     prob.Rows(),
		"cols":     prob.Cols(),
		"channels": prob.Channels(),
		"total":    prob.Total(),
	}).Info("YOLO output shape")
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

// parseYOLOOutput 解析YOLO输出（使用参考代码的方法）
func (e *VisionExecutor) parseYOLOOutput(outputs []gocv.Mat, width, height int, confidence, nmsThreshold float32) []map[string]interface{} {
	var classIDs []int
	var confidences []float32
	var boxes []image.Rectangle

	e.logger.WithFields(logrus.Fields{
		"num_outputs":   len(outputs),
		"confidence":    confidence,
		"nms_threshold": nmsThreshold,
	}).Debug("Parsing YOLO outputs")

	// 遍历所有输出层
	for layerIdx, output := range outputs {
		e.logger.WithFields(logrus.Fields{
			"layer": layerIdx,
			"rows":  output.Rows(),
			"cols":  output.Cols(),
			"total": output.Total(),
		}).Debug("Processing YOLO output layer")

		// 使用参考代码的方法：直接访问原始数据
		// YOLO输出格式：[x, y, w, h, conf, class_scores...] 每85个元素为一行
		rows := output.Total() / 85
		if rows == 0 {
			e.logger.WithField("layer", layerIdx).Warn("No valid detections in output")
			continue
		}

		data, err := output.DataPtrFloat32()
		if err != nil {
			e.logger.WithError(err).WithField("layer", layerIdx).Error("Failed to get data pointer")
			continue
		}

		e.logger.WithFields(logrus.Fields{
			"layer":       layerIdx,
			"rows":        rows,
			"data_length": len(data),
		}).Debug("Processing YOLO detections")

		imgW, imgH := float32(width), float32(height)

		for i := 0; i < int(rows); i++ {
			row := data[i*85 : (i+1)*85]

			// YOLO输出格式：[x, y, w, h, conf, class_scores...]
			centerX := row[0]
			centerY := row[1]
			bboxW := row[2]
			bboxH := row[3]
			conf := row[4]

			if conf < confidence {
				continue
			}

			// 找到最高分类分数
			classID, score := e.argmax(row[5:])
			finalScore := score * conf

			if finalScore > confidence {
				// 转换归一化坐标到像素坐标
				cx := centerX * imgW
				cy := centerY * imgH
				w := bboxW * imgW
				h := bboxH * imgH

				left := int(cx - w/2)
				top := int(cy - h/2)
				right := left + int(w)
				bottom := top + int(h)

				// 确保坐标在图像范围内
				if left < 0 {
					left = 0
				}
				if top < 0 {
					top = 0
				}
				if right >= width {
					right = width - 1
				}
				if bottom >= height {
					bottom = height - 1
				}

				e.logger.WithFields(logrus.Fields{
					"class_id":   classID,
					"confidence": finalScore,
					"bbox":       fmt.Sprintf("(%d,%d,%d,%d)", left, top, right, bottom),
				}).Info("Valid detection found")

				classIDs = append(classIDs, classID)
				confidences = append(confidences, finalScore)
				boxes = append(boxes, image.Rect(left, top, right, bottom))
			}
		}
	}

	// 如果没有检测结果，直接返回空数组
	if len(boxes) == 0 {
		return []map[string]interface{}{}
	}

	e.logger.WithFields(logrus.Fields{
		"boxes_before_nms": len(boxes),
		"nms_threshold":    nmsThreshold,
	}).Debug("Applying NMS and box fusion")

	// 首先使用标准NMS进行同类别的初步过滤
	indices := gocv.NMSBoxes(boxes, confidences, confidence, nmsThreshold)

	e.logger.WithField("boxes_after_nms", len(indices)).Debug("NMS completed")

	// 收集NMS后的结果
	var nmsResults []map[string]interface{}
	for _, idx := range indices {
		box := boxes[idx]
		nmsResults = append(nmsResults, map[string]interface{}{
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

	// 应用WBF (Weighted Box Fusion) 进行跨类别合并
	results := e.applyWeightedBoxFusion(nmsResults, 0.3) // IoU阈值0.3，更宽松的合并条件

	e.logger.WithFields(logrus.Fields{
		"boxes_after_wbf": len(results),
	}).Debug("WBF completed")

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
	_, exists := e.tasks[w.GetID()]
	if !exists {
		return fmt.Errorf("vision task not found: %s", w.GetID())
	}

	// 注意：不要关闭缓存的模型！
	// 模型会被多个任务共享，在executor销毁时统一释放

	delete(e.tasks, w.GetID())
	w.SetStatus(workload.StatusStopped)

	e.logger.WithField("workload_id", w.GetID()).Info("Vision task stopped")

	return nil
}

// Cleanup 清理所有缓存的模型（在executor销毁时调用）
func (e *VisionExecutor) Cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 关闭所有缓存的DNN模型
	for path, net := range e.modelCache {
		if net != nil {
			net.Close()
			e.logger.WithField("model", path).Info("Model cache released")
		}
	}

	// 关闭所有缓存的Cascade分类器
	for path, cascade := range e.cascadeCache {
		if cascade != nil {
			cascade.Close()
			e.logger.WithField("cascade", path).Info("Cascade cache released")
		}
	}

	e.modelCache = make(map[string]*gocv.Net)
	e.cascadeCache = make(map[string]*gocv.CascadeClassifier)

	e.logger.Info("All vision models released from cache")
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

// argmax 找最大概率类别
func (e *VisionExecutor) argmax(scores []float32) (int, float32) {
	maxVal := float32(-math.MaxFloat32)
	maxIdx := -1
	for i, v := range scores {
		if v > maxVal {
			maxVal = v
			maxIdx = i
		}
	}
	return maxIdx, maxVal
}

// applyWeightedBoxFusion 应用加权框融合 (WBF) 进行跨类别合并
func (e *VisionExecutor) applyWeightedBoxFusion(detections []map[string]interface{}, iouThreshold float64) []map[string]interface{} {
	if len(detections) <= 1 {
		return detections
	}

	// 将检测结果转换为内部格式

	var dets []Detection
	for _, det := range detections {
		bbox := det["bbox"].(map[string]int)
		rect := image.Rect(bbox["x"], bbox["y"], bbox["x"]+bbox["width"], bbox["y"]+bbox["height"])
		dets = append(dets, Detection{
			Bbox:       rect,
			Confidence: float64(det["confidence"].(float32)),
			ClassID:    det["class_id"].(int),
			Class:      det["class"].(string),
		})
	}

	// WBF算法实现
	var clusters [][]Detection
	used := make([]bool, len(dets))

	for i := 0; i < len(dets); i++ {
		if used[i] {
			continue
		}

		// 创建新聚类
		cluster := []Detection{dets[i]}
		used[i] = true

		// 寻找与当前框IoU超过阈值的框
		for j := i + 1; j < len(dets); j++ {
			if used[j] {
				continue
			}

			iou := e.calculateIoU(dets[i].Bbox, dets[j].Bbox)
			if iou > iouThreshold {
				cluster = append(cluster, dets[j])
				used[j] = true
			}
		}

		clusters = append(clusters, cluster)
	}

	// 合并每个聚类
	var results []map[string]interface{}
	for _, cluster := range clusters {
		if len(cluster) == 1 {
			// 单个框，直接添加
			det := cluster[0]
			results = append(results, map[string]interface{}{
				"class_id":   det.ClassID,
				"class":      det.Class,
				"confidence": float32(det.Confidence),
				"bbox": map[string]int{
					"x":      det.Bbox.Min.X,
					"y":      det.Bbox.Min.Y,
					"width":  det.Bbox.Dx(),
					"height": det.Bbox.Dy(),
				},
			})
		} else {
			// 多个框，进行加权融合
			fused := e.fuseBoxes(cluster)
			results = append(results, fused)
		}
	}

	return results
}

// calculateIoU 计算两个边界框的IoU
func (e *VisionExecutor) calculateIoU(box1, box2 image.Rectangle) float64 {
	// 计算交集
	intersection := box1.Intersect(box2)
	if intersection.Empty() {
		return 0.0
	}

	intersectionArea := intersection.Dx() * intersection.Dy()
	unionArea := (box1.Dx()*box1.Dy() + box2.Dx()*box2.Dy()) - intersectionArea

	if unionArea <= 0 {
		return 0.0
	}

	return float64(intersectionArea) / float64(unionArea)
}

// Detection 内部检测结构体
type Detection struct {
	Bbox       image.Rectangle
	Confidence float64
	ClassID    int
	Class      string
}

// fuseBoxes 融合多个检测框
func (e *VisionExecutor) fuseBoxes(cluster []Detection) map[string]interface{} {
	if len(cluster) == 0 {
		return nil
	}

	// 找到置信度最高的类别
	maxConf := 0.0
	bestClassID := 0
	bestClass := ""
	totalWeight := 0.0

	for _, det := range cluster {
		if det.Confidence > maxConf {
			maxConf = det.Confidence
			bestClassID = det.ClassID
			bestClass = det.Class
		}
		totalWeight += det.Confidence
	}

	// 加权融合边界框
	var weightedX1, weightedY1, weightedX2, weightedY2 float64
	for _, det := range cluster {
		weight := det.Confidence / totalWeight
		weightedX1 += float64(det.Bbox.Min.X) * weight
		weightedY1 += float64(det.Bbox.Min.Y) * weight
		weightedX2 += float64(det.Bbox.Max.X) * weight
		weightedY2 += float64(det.Bbox.Max.Y) * weight
	}

	// 计算融合后的置信度（取最高置信度）
	finalConf := maxConf

	return map[string]interface{}{
		"class_id":   bestClassID,
		"class":      bestClass,
		"confidence": float32(finalConf),
		"bbox": map[string]int{
			"x":      int(weightedX1),
			"y":      int(weightedY1),
			"width":  int(weightedX2 - weightedX1),
			"height": int(weightedY2 - weightedY1),
		},
	}
}
