# 构建 Go 二进制文件的 Dockerfile
FROM golang:1.24.2-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates

# 复制 go mod 文件
COPY ../../../go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY ../../.. .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fgmm -buildvcs=false .

# 导出二进制文件（包含 Python 环境）
FROM python:3.12-slim-bookworm

# 安装系统依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 安装 Python 依赖
RUN pip3 install --no-cache-dir py-clob-client

# 复制二进制文件
COPY --from=builder /app/fgmm /fgmm

# 复制配置文件
COPY --from=builder /app/conf /conf

# 复制 Python 脚本
COPY --from=builder /app/infrastructure/http/polymarket/python /infrastructure/http/polymarket/python

# 设置可执行权限
RUN chmod +x /fgmm

# 暴露端口
EXPOSE 9994

# 启动命令
CMD ["/fgmm", "--run_bot=true"]
