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
	CascadeType string
	CascadePath string
	Port        int
	Host        string
}

// OpenCVServer OpenCV推理服务器
type OpenCVServer struct {
	config    *Config
	classifier *gocv.CascadeClassifier
	logger    *log.Logger
}

// PredictRequest 推理请求
type PredictRequest struct {
	Image          string  `json:"image"`           // base64编码的图片
	ScaleFactor    float64 `json:"scale_factor"`    // 缩放因子，默认1.1
	MinNeighbors   int     `json:"min_neighbors"`   // 最小邻居数，默认3
	MinSize        int     `json:"min_size"`        // 最小尺寸，默认30
}

// Detection 检测结果
type Detection struct {
	Type string `json:"type"` // face, eye, smile
	BBox [4]int `json:"bbox"` // [x, y, width, height]
}

// PredictResponse 推理响应
type PredictResponse struct {
	Detections []Detection `json:"detections"`
	Count      int         `json:"count"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status       string `json:"status"`
	CascadeType  string `json:"cascade_type"`
}

// InfoResponse 信息响应
type InfoResponse struct {
	ServiceType  string `json:"service_type"`
	CascadeType  string `json:"cascade_type"`
	CascadePath  string `json:"cascade_path"`
	Version      string `json:"version"`
	Loaded       bool   `json:"loaded"`
}

func main() {
	// 解析命令行参数
	cascadeType := flag.String("cascade-type", "face", "Cascade type: face, eye, smile")
	cascadePath := flag.String("cascade-path", "", "Custom cascade file path")
	port := flag.Int("port", 9000, "HTTP server port")
	host := flag.String("host", "0.0.0.0", "HTTP server host")
	flag.Parse()

	// 创建配置
	config := &Config{
		CascadeType: *cascadeType,
		CascadePath: *cascadePath,
		Port:        *port,
		Host:        *host,
	}

	// 创建服务器
	server := NewOpenCVServer(config)

	// 加载Cascade分类器
	if err := server.LoadCascade(); err != nil {
		log.Fatalf("Failed to load cascade: %v", err)
	}

	// 启动HTTP服务器
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// NewOpenCVServer 创建OpenCV服务器
func NewOpenCVServer(config *Config) *OpenCVServer {
	return &OpenCVServer{
		config: config,
		logger: log.New(os.Stdout, "[OpenCV] ", log.LstdFlags),
	}
}

// LoadCascade 加载Cascade分类器
func (s *OpenCVServer) LoadCascade() error {
	var cascadePath string

	// 如果指定了自定义路径，使用自定义路径
	if s.config.CascadePath != "" {
		cascadePath = s.config.CascadePath
	} else {
		// 否则使用默认的Cascade文件
		switch s.config.CascadeType {
		case "face":
			// 尝试常见的Haar Cascade路径
			possiblePaths := []string{
				"models/haarcascade_frontalface_default.xml",
				"/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
				"/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
			}
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					cascadePath = path
					break
				}
			}
		case "eye":
			cascadePath = "models/haarcascade_eye.xml"
		case "smile":
			cascadePath = "models/haarcascade_smile.xml"
		default:
			return fmt.Errorf("unsupported cascade type: %s", s.config.CascadeType)
		}
	}

	if cascadePath == "" {
		return fmt.Errorf("cascade file not found for type: %s", s.config.CascadeType)
	}

	s.logger.Printf("Loading cascade from: %s", cascadePath)

	// 加载Cascade分类器
	classifier := gocv.NewCascadeClassifier()
	if !classifier.Load(cascadePath) {
		return fmt.Errorf("failed to load cascade from %s", cascadePath)
	}

	s.classifier = &classifier
	s.logger.Printf("Cascade loaded successfully")

	return nil
}

// Start 启动HTTP服务器
func (s *OpenCVServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 注册路由
	http.HandleFunc("/predict", s.handlePredict)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/info", s.handleInfo)

	s.logger.Printf("Starting OpenCV inference server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handlePredict 处理推理请求
func (s *OpenCVServer) handlePredict(w http.ResponseWriter, r *http.Request) {
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
	if req.ScaleFactor == 0 {
		req.ScaleFactor = 1.1
	}
	if req.MinNeighbors == 0 {
		req.MinNeighbors = 3
	}
	if req.MinSize == 0 {
		req.MinSize = 30
	}

	// 解码图片
	img, err := s.decodeImage(req.Image)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode image: %v", err), http.StatusBadRequest)
		return
	}
	defer img.Close()

	// 执行检测
	detections := s.detect(&img, req.ScaleFactor, req.MinNeighbors, req.MinSize)

	// 返回结果
	resp := PredictResponse{
		Detections: detections,
		Count:      len(detections),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	s.logger.Printf("Detection completed: %d %ss found", len(detections), s.config.CascadeType)
}

// handleHealth 健康检查
func (s *OpenCVServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:      "healthy",
		CascadeType: s.config.CascadeType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleInfo 返回服务信息
func (s *OpenCVServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	resp := InfoResponse{
		ServiceType: "opencv",
		CascadeType: s.config.CascadeType,
		CascadePath: s.config.CascadePath,
		Version:     "1.0",
		Loaded:      s.classifier != nil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// decodeImage 解码base64图片
func (s *OpenCVServer) decodeImage(imageData string) (gocv.Mat, error) {
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

// detect 执行检测
func (s *OpenCVServer) detect(img *gocv.Mat, scaleFactor float64, minNeighbors, minSize int) []Detection {
	if s.classifier == nil {
		return []Detection{}
	}

	// 检测对象
	rects := s.classifier.DetectMultiScale(*img)
	
	// 转换为Detection格式
	var detections []Detection
	for _, rect := range rects {
		// 过滤太小的检测
		if rect.Dx() < minSize || rect.Dy() < minSize {
			continue
		}
		
		detections = append(detections, Detection{
			Type: s.config.CascadeType,
			BBox: [4]int{rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy()},
		})
	}

	return detections
}

// handlePredictFile 支持从文件上传
func (s *OpenCVServer) handlePredictFile(w http.ResponseWriter, r *http.Request) {
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

	// 执行检测
	detections := s.detect(&img, 1.1, 3, 30)

	// 返回结果
	resp := PredictResponse{
		Detections: detections,
		Count:      len(detections),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

