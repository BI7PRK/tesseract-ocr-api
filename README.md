# Tesseract OCR API

这是一个基于 Go 和 Tesseract OCR 的轻量级 HTTP 图片文字识别服务。客户端通过 `POST /ocr` 上传图片，服务调用 Tesseract 识别图片中的文本，并以 JSON 返回识别结果、处理耗时和所用语言。

当前服务将识别语言固定为老挝语和英语（`lao+eng`），适用于包含这两种语言的图片或截图。Docker 镜像会安装 Tesseract OCR、英文语言数据，并下载老挝语语言数据；OCR 使用 LSTM 引擎（`--oem 1`）和适合单块文本的页面分割模式（`--psm 6`）。

## 使用 Docker 启动

在项目根目录构建镜像并启动容器：

```bash
docker build -t tesseract-ocr-api .
docker run --rm -p 8080:8080 tesseract-ocr-api
```

服务启动后，可通过 `http://localhost:8080/ocr` 访问 OCR 接口。

## API

### `POST /ocr`

使用 `multipart/form-data` 上传图片，表单字段名必须为 `image`。支持的图片格式取决于容器内 Tesseract/Leptonica 的构建支持。

| 项目 | 说明 |
| --- | --- |
| 方法 | `POST` |
| 路径 | `/ocr` |
| Content-Type | `multipart/form-data` |
| 表单字段 | `image`：待识别的图片文件 |
| 成功状态码 | `200 OK` |

### curl 示例

将 `./sample.jpg` 替换为本地图片路径：

```bash
curl -X POST "http://localhost:8080/ocr" \
	-F "image=@./sample.jpg"
```

Windows PowerShell 中也可以使用 `curl.exe`：

```powershell
curl.exe -X POST "http://localhost:8080/ocr" -F "image=@C:\path\to\sample.jpg"
```

成功时返回类似以下 JSON：

```json
{
	"text": "识别出的文本内容",
	"millisecond": 128,
	"lang": "lao+eng"
}
```

`millisecond` 是本次请求的处理耗时（毫秒），`text` 为 Tesseract 输出的识别文本。

### 错误示例

不带图片字段会返回 `400 Bad Request`：

```bash
curl -X POST "http://localhost:8080/ocr"
```

使用非 `POST` 方法会返回 `405 Method Not Allowed`：

```bash
curl -i "http://localhost:8080/ocr"
```

错误响应为 JSON，例如：

```json
{
	"error": "Failed to read 'image' from form data: ..."
}
```

## 注意事项

- 当前 API 没有提供语言参数；识别语言固定为 `lao+eng`。
- 镜像目前只安装英文和老挝语语言数据。若要识别中文等其他语言，需要在镜像中安装对应的 Tesseract `.traineddata` 文件，并调整 `main.go` 中的语言配置。
- 服务监听容器内的 `8080` 端口；运行容器时需要将该端口映射到宿主机。

