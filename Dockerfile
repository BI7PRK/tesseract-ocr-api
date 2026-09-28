# Stage 1: 编译 Go 二进制文件
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init tesseract-api && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /tesseract-api main.go

# Stage 2: 构建最终运行环境
FROM alpine:3.20

# 安装 Tesseract 引擎和英文数据包
RUN apk add --no-cache tesseract-ocr tesseract-ocr-data-eng curl

# 使用您提供的准确官方直链下载老挝语字库，并加入强校验（文件必须大于 1MB）
RUN curl -L -o /usr/share/tessdata/lao.traineddata "https://github.com/tesseract-ocr/tessdata/raw/refs/heads/main/lao.traineddata" \
    && [ "$(stat -c %s /usr/share/tessdata/lao.traineddata)" -gt 1000000 ] || (echo "Lao data download corrupted!" && exit 1)

# 配置环境变量：显式指定字库路径，并彻底禁用引发 pixCloseBrick 警告的 OpenCL 硬件加速
ENV TESSDATA_PREFIX=/usr/share/tessdata/
ENV TESS_OPENCL_DEVICE=0
ENV TESS_OPENCL_WRITE_PROFILE=0

COPY --from=builder /tesseract-api /tesseract-api

EXPOSE 8080
ENTRYPOINT ["/tesseract-api"]
