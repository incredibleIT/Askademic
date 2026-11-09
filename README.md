<div align="center">

# Askademic · 校内部署管理员手册

**面向信息中心 / 网络中心**：组件拓扑、部署步骤、配置说明、运维与安全清单。

[系统概述](#系统概述) · [部署步骤](#部署步骤) · [运维](#运维) · [安全清单](#安全清单)

</div>

> 本版本面向学校信息化部门校内部署与运维使用，基于 [Askademic](https://github.com/incredibleIT/Askademic)（主仓库）构建。Askademic 是腾讯开源 [WeKnora](https://github.com/Tencent/WeKnora)（MIT License）的教务场景二次开发版。

## 系统概述

Askademic 是基于 RAG（检索增强生成）与智能体的高校教务智能问答系统：文档知识库问答、回答附出处、按空间与角色隔离数据。

组件拓扑（全部容器化，`docker-compose.yml` 单机编排）：

| 服务 | 容器名 | 端口 | 职责 |
|---|---|---|---|
| frontend | WeKnora-frontend | `FRONTEND_PORT`(默认 80) | Web 入口（Nginx，反代 API） |
| app | WeKnora-app | `APP_PORT`(默认 8080) | Go 后端：会话/知识库/权限/模型接入 |
| postgres | WeKnora-postgres | 5432（容器网内） | PostgreSQL (ParadeDB)：业务数据 + 检索 |
| redis | WeKnora-redis | 6379（容器网内） | 缓存 / 队列 |
| docreader | WeKnora-docreader | 内部 | 文档解析 + 内置本地 OCR |
| minio | WeKnora-minio | 9000/9001 | 对象存储（`STORAGE_TYPE=minio` 时） |

资源建议：4 核 CPU / 8 GB 内存 / 50 GB 磁盘起步（文档量大时按存储增速扩容）。**出网要求**：app 容器需能访问所配置的大模型 API（LLM / Embedding）；OCR 与解析全部本地运行，不依赖外部服务。

## 部署步骤

```bash
git clone -b info-center https://github.com/incredibleIT/Askademic.git askademic
cd askademic
cp .env.example .env
```

`.env` 关键配置（完整项见文件内注释）：

| 配置 | 必改 | 说明 |
|---|---|---|
| `JWT_SECRET` / `SYSTEM_AES_KEY` / `SYSTEM_SIGNING_KEY` | ✅ | 三个密钥改为随机值（`openssl rand -hex 32`）；`SYSTEM_AES_KEY` 用于 API Key 等敏感字段落库加密 |
| `DB_PASSWORD` / `REDIS_PASSWORD` | ✅ | 改掉示例密码 |
| `FRONTEND_PORT` / `APP_PORT` | 视情况 | Web 与 API 端口，避开校内端口冲突 |
| `STORAGE_TYPE` | 视情况 | `local`（默认，挂载本机目录）或 `minio` |
| `TZ` | 建议 | `Asia/Shanghai` |

启动与健康检查：

```bash
docker compose up -d
docker compose ps                    # 各服务 Up / healthy
curl -s http://127.0.0.1:${APP_PORT:-8080}/ -o /dev/null -w "%{http_code}\n"   # API 存活
```

首次初始化（Web 入口）：

1. 注册第一个账号（即系统管理员）。
2. 「设置 → 模型管理」接入大模型与 Embedding 服务；「连通性测试」通过后保存。
3. 创建知识库、上传文档，验证解析与问答链路。

## 运维

| 事项 | 操作 |
|---|---|
| 日志 | `docker compose logs -f app`（后端）；`logs docreader`（解析）；按需 `--tail 200` |
| 数据库备份 | `docker compose exec postgres pg_dump -U $DB_USER $DB_NAME > backup-$(date +%F).sql`，建议每日定时 |
| 文档文件备份 | `STORAGE_TYPE=local` 时备份 `LOCAL_STORAGE_BASE_DIR` 指向的挂载卷；`minio` 时备份其数据卷 |
| 升级 | 拉取新版本后 `docker compose up -d`（数据库结构自动迁移；升级前先做备份） |
| 服务重启 | `docker compose restart app` 单服务；`docker compose down && up -d` 全套（数据在卷中，不丢失） |
| 容量监控 | 关注 postgres 数据卷与文档存储卷的增长；`docker system df` 看总体占用 |

## 安全清单

- [ ] 三个系统密钥与数据库/Redis 密码已全部改为随机值（不得使用 `.env.example` 示例值上线）
- [ ] 管理员账号使用强密码；注册入口按需封闭（仅管理员建号）
- [ ] Web 入口置于校内网或经统一身份/反向代理加 HTTPS 后暴露
- [ ] 数据库 / Redis 端口**不**对校园网暴露（仅容器网内互通）
- [ ] 模型 API Key 只录入「设置 → 模型管理」（AES 加密落库），不写入配置文件或代码仓库
- [ ] 定期核查「设置 → 用户管理」中的账号与空间成员列表

## 开源来源

基于腾讯开源 [WeKnora](https://github.com/Tencent/WeKnora)（MIT License）二次开发；原始许可文本保留于 [LICENSE](LICENSE)。上游提交历史不随本仓库分发。
