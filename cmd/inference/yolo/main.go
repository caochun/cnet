package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"gocv.io/x/gocv"
)

// Config 服务配置
type Config struct {
	ModelPath string
	Port      int
	Host      string
}

// YOLOServer YOLO推理服务器
type YOLOServer struct {
	config *Config
	net    *gocv.Net
	logger *log.Logger
}

// PredictRequest 推理请求
type PredictRequest struct {
	Image        string  `json:"image"`         // base64编码的图片
	Confidence   float32 `json:"confidence"`    // 置信度阈值，默认0.5
	IOUThreshold float32 `json:"iou_threshold"` // NMS IoU阈值，默认0.4
}

// Detection 检测结果
type Detection struct {
	Class      string  `json:"class"`
	Confidence float32 `json:"confidence"`
	BBox       [4]int  `json:"bbox"` // [x, y, width, height]
}

// PredictResponse 推理响应
type PredictResponse struct {
	Detections []Detection `json:"detections"`
	Count      int         `json:"count"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status string `json:"status"`
	Model  string `json:"model"`
}

// InfoResponse 信息响应
type InfoResponse struct {
	ModelType string `json:"model_type"`
	ModelPath string `json:"model_path"`
	Version   string `json:"version"`
	Loaded    bool   `json:"loaded"`
}

func main() {
	// 解析命令行参数
	modelPath := flag.String("model", "", "Path to YOLO model file (.onnx)")
	port := flag.Int("port", 9001, "HTTP server port")
	host := flag.String("host", "0.0.0.0", "HTTP server host")
	flag.Parse()

	if *modelPath == "" {
		log.Fatal("Model path is required (--model)")
	}

	// 创建配置
	config := &Config{
		ModelPath: *modelPath,
		Port:      *port,
		Host:      *host,
	}

	// 创建服务器
	server := NewYOLOServer(config)

	// 加载模型
	if err := server.LoadModel(); err != nil {
		log.Fatalf("Failed to load model: %v", err)
	}

	// 启动HTTP服务器
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// NewYOLOServer 创建YOLO服务器
func NewYOLOServer(config *Config) *YOLOServer {
	return &YOLOServer{
		config: config,
		logger: log.New(os.Stdout, "[YOLO] ", log.LstdFlags),
	}
}

// LoadModel 加载YOLO模型
func (s *YOLOServer) LoadModel() error {
	s.logger.Printf("Loading YOLO model from: %s", s.config.ModelPath)

	// 使用GoCV加载ONNX模型
	net := gocv.ReadNet(s.config.ModelPath, "")
	if net.Empty() {
		return fmt.Errorf("failed to load model from %s", s.config.ModelPath)
	}

	s.net = &net
	s.logger.Printf("Model loaded successfully")

	return nil
}

// Start 启动HTTP服务器
func (s *YOLOServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 注册路由
	http.HandleFunc("/predict", s.handlePredict)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/info", s.handleInfo)

	s.logger.Printf("Starting YOLO inference server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handlePredict 处理推理请求
func (s *YOLOServer) handlePredict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// 设置默认值
	if req.Confidence == 0 {
		req.Confidence = 0.5
	}
	if req.IOUThreshold == 0 {
		req.IOUThreshold = 0.4
	}

	// 解码图片
	img, err := s.decodeImage(req.Image)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode image: %v", err), http.StatusBadRequest)
		return
	}
	defer img.Close()

	// 执行推理
	detections, err := s.predict(&img, req.Confidence, req.IOUThreshold)
	if err != nil {
		http.Error(w, fmt.Sprintf("Prediction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	resp := PredictResponse{
		Detections: detections,
		Count:      len(detections),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	s.logger.Printf("Prediction completed: %d detections", len(detections))
}

// handleHealth 健康检查
func (s *YOLOServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status: "healthy",
		Model:  s.config.ModelPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleInfo 返回服务信息
func (s *YOLOServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	resp := InfoResponse{
		ModelType: "yolo",
		ModelPath: s.config.ModelPath,
		Version:   "1.0",
		Loaded:    s.net != nil && !s.net.Empty(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// decodeImage 解码base64图片
func (s *YOLOServer) decodeImage(imageData string) (gocv.Mat, error) {
	// 如果是data URL格式，去掉前缀
	if strings.HasPrefix(imageData, "data:image") {
		parts := strings.Split(imageData, ",")
		if len(parts) == 2 {
			imageData = parts[1]
		}
	}

	// Base64解码
	imgBytes, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("base64 decode failed: %w", err)
	}

	// 解码为图片
	img, _, err := image.Decode(strings.NewReader(string(imgBytes)))
	if err != nil {
		// 尝试使用GoCV IMDecode
		mat, err2 := gocv.IMDecode(imgBytes, gocv.IMReadColor)
		if err2 != nil {
			return gocv.Mat{}, fmt.Errorf("image decode failed: %w", err)
		}
		return mat, nil
	}

	// 转换为GoCV Mat
	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("failed to convert to mat: %w", err)
	}

	return mat, nil
}

// predict 执行YOLO推理
func (s *YOLOServer) predict(img *gocv.Mat, confidence, iouThreshold float32) ([]Detection, error) {
	if s.net == nil || s.net.Empty() {
		return nil, fmt.Errorf("model not loaded")
	}

	// 预处理图片
	blob := gocv.BlobFromImage(*img, 1.0/255.0, image.Pt(640, 640), gocv.NewScalar(0, 0, 0, 0), true, false)
	defer blob.Close()

	// 设置输入
	s.net.SetInput(blob, "")

	// 前向推理
	output := s.net.Forward("")
	defer output.Close()

	// 解析输出
	detections := s.parseYOLOOutput(&output, img.Cols(), img.Rows(), confidence, iouThreshold)

	return detections, nil
}

// parseYOLOOutput 解析YOLO输出
func (s *YOLOServer) parseYOLOOutput(output *gocv.Mat, imgWidth, imgHeight int, confThreshold, iouThreshold float32) []Detection {
	// 获取输出数据
	data, _ := output.DataPtrFloat32()
	if len(data) == 0 {
		s.logger.Println("Warning: Empty output from model")
		return []Detection{}
	}

	// 计算行数和列数
	rows := int(output.Total()) / 85 // YOLO输出格式: [x, y, w, h, conf, class_scores...]

	s.logger.Printf("Output shape: rows=%d, total=%d", rows, output.Total())

	var boxes []image.Rectangle
	var confidences []float32
	var classIDs []int

	// 解析每个检测
	for i := 0; i < rows; i++ {
		offset := i * 85

		// 获置信度
		confidence := data[offset+4]

		if confidence < confThreshold {
			continue
		}

		// 获取类别分数
		classScores := data[offset+5 : offset+85]
		classID := argmax(classScores)
		classScore := classScores[classID]

		finalConf := confidence * classScore
		if finalConf < confThreshold {
			continue
		}

		// 获取边界框（归一化坐标）
		x := data[offset+0]
		y := data[offset+1]
		w := data[offset+2]
		h := data[offset+3]

		// 转换为像素坐标
		left := int((x - w/2) * float32(imgWidth))
		top := int((y - h/2) * float32(imgHeight))
		width := int(w * float32(imgWidth))
		height := int(h * float32(imgHeight))

		boxes = append(boxes, image.Rect(left, top, left+width, top+height))
		confidences = append(confidences, finalConf)
		classIDs = append(classIDs, classID)
	}

	// 应用NMS
	if len(boxes) == 0 {
		return []Detection{}
	}

	indices := gocv.NMSBoxes(boxes, confidences, confThreshold, iouThreshold)

	// 构建最终结果
	var detections []Detection
	for _, idx := range indices {
		box := boxes[idx]
		detections = append(detections, Detection{
			Class:      fmt.Sprintf("class_%d", classIDs[idx]),
			Confidence: confidences[idx],
			BBox:       [4]int{box.Min.X, box.Min.Y, box.Dx(), box.Dy()},
		})
	}

	return detections
}

// argmax 返回最大值的索引
func argmax(data []float32) int {
	if len(data) == 0 {
		return -1
	}

	maxIdx := 0
	maxVal := data[0]

	for i := 1; i < len(data); i++ {
		if data[i] > maxVal {
			maxVal = data[i]
			maxIdx = i
		}
	}

	return maxIdx
}

// 支持从文件上传的handlePredictFile
func (s *YOLOServer) handlePredictFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 读取文件内容
	imgBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}

	// 解码图片
	img, err := gocv.IMDecode(imgBytes, gocv.IMReadColor)
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusBadRequest)
		return
	}
	defer img.Close()

	// 执行推理
	detections, err := s.predict(&img, 0.5, 0.4)
	if err != nil {
		http.Error(w, fmt.Sprintf("Prediction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回结果
	resp := PredictResponse{
		Detections: detections,
		Count:      len(detections),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
