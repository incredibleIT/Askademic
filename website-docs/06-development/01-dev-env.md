# 本地开发环境

## 一键启动

仓库根目录的总控脚本 start-weknora.sh 幂等地拉起全部本地服务：

1. PostgreSQL 与 Redis：Docker 容器（dev compose 文件）。
2. weknora-server：Go 后端，监听 8080，同时托管前端静态产物。
3. docreader：Python 解析服务，gRPC 50051。
4. embedding：本地向量服务 8791（可选）。

支持 status 查看状态、stop 仅停本地进程不动容器。

## 前端开发

frontend/ 为 Vue 3 + Vite 工程：开发期 vite dev 热更新；部署期 vite build 产出 dist，由后端按 WEKNORA_WEB_DIR 托管，无需单独起前端服务。

## Go 工具链

要求 Go 1.26+ 与 CGO（向量扩展依赖）。国内网络建议设置 GOPROXY=https://goproxy.cn,direct。构建：go build ./...；测试：go test ./...。

## 配置

后端读 config/config.yaml 与环境变量（.env）；提示词模板在 config/prompt_templates/，启动时从磁盘加载。
