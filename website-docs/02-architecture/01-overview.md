# 总体架构

Askademic 采用前后端分离 + 多辅助服务的架构，全部可容器化部署。

## 组件

- Web 前端：Vue 3 + TDesign，构建产物由后端静态托管，单端口对外服务。
- Go 后端：Gin 框架，承载 REST API、SSE 流式问答、任务队列与权限体系。
- DocReader：Python 解析服务，负责 PDF、Office、图片 OCR 等格式的文本抽取。
- PostgreSQL：业务数据主库；向量检索支持 Elasticsearch、OpenSearch 及多种向量库后端。
- Neo4j（可选）：知识图谱存储，支撑 GraphRAG 检索。
- Embedding 服务：本地向量模型服务（可选，亦可调用云端 API）。

## 两条主线

系统的一切能力围绕两条主线组织：

1. 知识线（RAG）：资料从上传到可被引用的完整流水线，详见 [RAG 管线](02-rag-pipeline.md)。
2. 执行线（Agent）：智能体的推理与工具执行引擎，详见 [Agent 引擎](03-agent-engine.md)。

两条线的交汇点是内置检索工具：智能体在推理过程中调用混合检索获取知识库证据。

## 部署形态

- 本地开发：总控脚本一键拉起数据库容器、后端、解析服务与向量服务。
- Docker Compose：全部组件容器化，适合演示与答辩环境。
- Lite 单二进制：内置简化依赖的最小形态。
