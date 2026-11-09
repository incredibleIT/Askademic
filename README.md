<div align="center">

# Askademic · 实验室部署版

**一句话**：把学校制度文档和资料传进知识库，实验室成员用自然语言直接问，回答有出处。

[部署步骤](#部署步骤) · [日常使用](#日常使用) · [账号管理](#账号管理) · [常见问题](#常见问题)

</div>

> 本版本面向实验室 / 社团内部部署使用，基于 [Askademic](https://github.com/incredibleIT/Askademic)（主仓库）构建。Askademic 是腾讯开源 [WeKnora](https://github.com/Tencent/WeKnora)（MIT License）的教务场景二次开发版。

## 部署步骤

前置要求：一台 Linux/macOS 机器，装好 Docker 与 Docker Compose，能访问大模型 API（出网）。

```bash
git clone -b ustil https://github.com/incredibleIT/Askademic.git askademic
cd askademic
cp .env.example .env
```

编辑 `.env`，至少确认这几项：

| 配置 | 说明 |
|---|---|
| `JWT_SECRET` / `SYSTEM_AES_KEY` / `SYSTEM_SIGNING_KEY` | 三个密钥**必须改成随机值**（`openssl rand -hex 32`） |
| `DB_PASSWORD` / `REDIS_PASSWORD` | 改掉示例密码 |
| `FRONTEND_PORT` | Web 入口端口，默认 80，被占用就改（如 8081） |
| `APP_PORT` | API 端口，默认 8080 |

启动与初始化：

```bash
docker compose up -d          # 拉起 app / frontend / postgres / redis / docreader 全套
docker compose ps             # 全部 Up/healthy 即成功
```

首次配置（浏览器打开 `http://<主机IP>:<FRONTEND_PORT>`）：

1. 注册第一个账号（即管理员）。
2. 「设置 → 模型管理」接入大模型与 Embedding（任意 OpenAI 兼容 API，Key 加密存库）。
3. 「知识库」新建知识库，上传制度文档（PDF / Word / 扫描件均可，内置 OCR）。

## 日常使用

- **提问**：首页输入框直接问；回答附知识库出处，可点开核对原文。
- **建知识库**：按课题/方向分库；「共享给」可以把库共享给组织内其他成员（viewer 只读 / contributor 可传文档）。
- **选智能体**：内置「快速问答」（RAG 直答）与「智能推理」（多步推理 + 工具调用），在「智能体」页查看与配置。
- **数据备份**：定期 `docker compose exec postgres pg_dump -U <DB_USER> <DB_NAME> > backup.sql`，文档文件按 `STORAGE_TYPE` 配置的位置一并备份。

## 账号管理

- 注册入口默认开放；如需封闭注册（仅管理员建号），在前端构建时设 `VITE_ENABLE_REGISTER=false` 或联系仓库维护者获取封闭版配置。
- 成员权限按「空间」隔离：每人有自己的个人空间，组织空间由管理员管理。
- 忘记密码：管理员可在「设置 → 用户管理」重置。

## 常见问题

| 问题 | 处理 |
|---|---|
| 上传文档一直解析中 | docreader 容器未就绪：`docker compose logs docreader`；扫描件解析较慢属正常 |
| 回答不引用上传的文档 | 确认文档解析完成、提问时选中了对应知识库 |
| 模型连接失败 | 「设置 → 模型管理」里点「连通性测试」；检查 API Key、出网代理 |
| 页面打不开 | `docker compose ps` 看 frontend/app 状态；端口冲突改 `FRONTEND_PORT` |

## 开源来源

基于腾讯开源 [WeKnora](https://github.com/Tencent/WeKnora)（MIT License）二次开发；原始许可文本保留于 [LICENSE](LICENSE)。上游提交历史不随本仓库分发。
