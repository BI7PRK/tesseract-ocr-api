package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// OCRResponse 是 /ocr 接口的 JSON 返回结构
type OCRResponse struct {
	Text        string `json:"text"`
	Millisecond int64  `json:"millisecond"`
	Lang        string `json:"lang"`
}

const defaultLang = "lao+eng"

// availableLangs 是系统已安装的 Tesseract 语言白名单
var (
	availableLangs map[string]bool
	langsOnce      sync.Once
)

func main() {
	loadAvailableLangs()

	http.HandleFunc("/ocr", ocrHandler)

	fmt.Println("Server successfully started on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// loadAvailableLangs 调用 tesseract --list-langs，把语言名缓存到 map
func loadAvailableLangs() {
	out, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to list tesseract languages: %v\n%s", err, out)
	}

	availableLangs = make(map[string]bool)

	// 输出格式：
	// List of available languages in "/usr/share/tesseract-ocr/5/tessdata/" (3):
	// eng
	// lao
	// osd
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 跳过第一行说明文字
		if strings.HasPrefix(line, "List of available languages") {
			continue
		}
		availableLangs[line] = true
	}

	log.Printf("Loaded %d tesseract languages: %v", len(availableLangs), keys(availableLangs))
}

// validateLang 校验 "chi_sim+eng" 这种组合，每个子语言都必须在白名单里
func validateLang(lang string) error {
	parts := strings.Split(lang, "+")
	if len(parts) == 0 {
		return fmt.Errorf("lang must not be empty")
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return fmt.Errorf("invalid lang format: %q", lang)
		}
		if !availableLangs[p] {
			return fmt.Errorf("unsupported language: %q", p)
		}
	}
	return nil
}

// writeJSON 是统一的 JSON 响应工具函数
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[JSON ENCODE ERROR] %v", err)
	}
}

func ocrHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Only POST method is allowed",
		})
		return
	}

	// 0. 解析并校验 lang 参数
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = defaultLang
	}
	if err := validateLang(lang); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
			"hint":  "available languages: " + strings.Join(keys(availableLangs), ", "),
		})
		return
	}

	// 1. 从 multipart form 中读取上传的图片
	file, _, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Failed to read 'image' from form data: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 2. 保存到临时文件
	tempFile, err := os.CreateTemp("", "ocr-upload-*.jpg")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create temp file: " + err.Error(),
		})
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err = io.Copy(tempFile, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to save uploaded image: " + err.Error(),
		})
		return
	}

	// 3. 调用 Tesseract CLI
	cmd := exec.Command("tesseract", tempFile.Name(), "stdout",
		"-l", lang,
		"--oem", "1",
		"--psm", "6",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("[OCR ERROR] %v\nstderr: %s", err, stderr.String())
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "OCR execution failed: " + err.Error(),
		})
		return
	}

	if stderr.Len() > 0 {
		log.Printf("[tesseract stderr] %s", stderr.String())
	}

	// 4. 返回 JSON
	elapsed := time.Since(start).Milliseconds()

	writeJSON(w, http.StatusOK, OCRResponse{
		Text:        stdout.String(),
		Millisecond: elapsed,
		Lang:        lang,
	})
}

// keys 是 map[string]bool 的辅助函数，返回排序前的键列表
func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
