package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type listV2 struct {
	XMLName     xml.Name `xml:"ListBucketResult"`
	Name        string   `xml:"Name"`
	Prefix      string   `xml:"Prefix"`
	KeyCount    int      `xml:"KeyCount"`
	MaxKeys     int      `xml:"MaxKeys"`
	IsTruncated bool     `xml:"IsTruncated"`
	Contents    []object `xml:"Contents"`
}

type object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

func fileMD5Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeHead(w http.ResponseWriter, full string, fi os.FileInfo) {
	ctype := mime.TypeByExtension(filepath.Ext(full))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
	if etag, err := fileMD5Hex(full); err == nil {
		w.Header().Set("ETag", etag)
	}
}

func handleListObjectsV2(w http.ResponseWriter, basePath, bucket, prefix string) {
	safe := filepath.Clean(prefix)
	if strings.HasPrefix(safe, "..") {
		http.Error(w, "Invalid prefix", http.StatusBadRequest)
		return
	}
	root := filepath.Join(basePath, safe)
	var files []object
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(basePath, path)
		etag, _ := fileMD5Hex(path)
		files = append(files, object{
			Key:          rel,
			LastModified: info.ModTime().UTC().Format(time.RFC3339),
			ETag:         fmt.Sprintf("\"%s\"", etag),
			Size:         info.Size(),
			StorageClass: "STANDARD",
		})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Key < files[j].Key })
	resp := listV2{Name: bucket, Prefix: safe, KeyCount: len(files), MaxKeys: len(files), IsTruncated: false, Contents: files}
	w.Header().Set("Content-Type", "application/xml")
	_ = xml.NewEncoder(w).Encode(resp)
}

func main() {
	basePath := flag.String("base-path", "/tmp/cnet_data", "Base data path")
	bucket := flag.String("bucket", "cnet", "Bucket name")
	port := flag.Int("port", 9090, "HTTP port")
	host := flag.String("host", "127.0.0.1", "Bind host")
	authToken := flag.String("auth-token", "", "Bearer token (optional)")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/s3/", func(w http.ResponseWriter, r *http.Request) {
		if *authToken != "" && r.Header.Get("Authorization") != "Bearer "+*authToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/s3/")
		seg := strings.SplitN(path, "/", 2)
		if len(seg) == 0 || seg[0] == "" {
			http.Error(w, "No bucket", http.StatusBadRequest)
			return
		}
		if seg[0] != *bucket {
			http.Error(w, "NoSuchBucket", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			prefix := r.URL.Query().Get("prefix")
			handleListObjectsV2(w, *basePath, *bucket, prefix)
			return
		}
		if len(seg) < 2 {
			http.Error(w, "No object key", http.StatusBadRequest)
			return
		}
		key := filepath.Clean(seg[1])
		if strings.HasPrefix(key, "..") {
			http.Error(w, "Invalid key", http.StatusBadRequest)
			return
		}
		full := filepath.Join(*basePath, key)
		fi, err := os.Stat(full)
		if err != nil || fi.IsDir() {
			http.Error(w, "NoSuchKey", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodHead {
			writeHead(w, full, fi)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
			return
		}
		writeHead(w, full, fi)
		f, err := os.Open(full)
		if err != nil {
			http.Error(w, "InternalError", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		_, _ = io.Copy(w, f)
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{Addr: addr, Handler: mux}
	fmt.Printf("[gateway] listening on %s, base=%s, bucket=%s\n", addr, *basePath, *bucket)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
