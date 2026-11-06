# WeKnora 用户视角功能盘点(改造基线)

> 2026-11-04 · 由前端路由(frontend/src/router/index.ts)+ 后端路由(internal/router/、internal/handler/)+ README 交叉核实。
> 用途:二次开发改造(校园教务智能助手)的"家底清单"——先知道有什么,再决定留什么、砍什么、改什么。
> FE = frontend/src,BE = internal。FE=frontend/src,BE=internal。系统/空间设置均为 Settings 模态(`views/settings/Settings.vue`,`/platform/settings?section=<key>`)。

## 功能清单(按用户视角分组)

### 知识库与文档
- 知识库列表(置顶/复制/克隆/删除)— `/platform/knowledge-bases`,FE `views/knowledge/KnowledgeBaseList.vue` — BE `handler/knowledgebase.go` + `router/routes_knowledge.go` — 文档/FAQ/Wiki 三类库
- 文档上传(文件/URL/手动)、文件夹树、批量重解析/删除/移动/下载、解析进度时间线 — `views/knowledge/KnowledgeBase.vue` — `handler/knowledge.go` + `routes_knowledge.go` — 支持整目录上传保留结构
- 库级设置:解析引擎/分块/多模态图片管线/向量库/存储/图谱 — FE `views/knowledge/settings/KBParserSettings.vue`、`GraphSettings.vue` 等 — `handler/knowledgebase_parser_patch.go` 等 — 可覆盖全局默认
- 分块编辑、版本历史与回滚、自动生成候选问题 — 同详情页 — `handler/chunk.go`(PUT /chunks/:kid/:id/revert、questions/regenerate)
- FAQ 问答对管理(相似问/批量导入/导出,Excel)— FE `i18n/zh-CN.ts:5562` — `handler/faq.go`(go.mod excelize)
- 图片画廊 — FE `views/knowledge/gallery/ImageGallery.vue` — `handler/knowledge_images*`
- 数据源接入(飞书/Confluence/GitLab/Notion/语雀/钉钉/Seafile/RSS 等 11 种)与同步日志 — FE `views/knowledge/settings/DataSourceSettings.vue` — `handler/datasource.go` + `routes_infra.go:319-350` 【改造:建议删除】

### 问答与检索
- 会话(快速问答=知识库 RAG / 智能推理=Agent)— `views/creatChat/creatChat.vue`,对话 `/platform/chat/:chatid` → `views/chat/index.vue` — `handler/session/`、`handler/message.go` + `routes_chat.go`
- 混合检索(关键词+向量+重排序)— ⌘K 全局命令面板 — `handler/knowledgebase.go` HybridSearch
- 知识图谱检索 GraphRAG(Neo4j)— FE `GraphSettings.vue`、`views/chat/components/tool-results/GraphQueryResults.vue` — `handler/graph_search.go` + `router/routes_graph.go` + `application/repository/retriever/neo4j/`【加分项:保留】
- 会话操作:分叉/回滚/steer 插话/置顶/停止/语音转写/临时附件 — `views/chat/` — `routes_chat.go:51-136`
- 消息追问建议 — `handler/message_suggestion.go`
- 跨会话长期记忆(需用户确认条目)— FE `views/settings/MemorySettings.vue`、`MemoryWorkspaceSettings.vue`、`views/chat/components/ChatMemoryStep.vue` — `handler/memory.go`、`memory_admin.go` + `routes_memory.go`

### Agent
- 自定义 Agent(建/复制/编辑提示词/模型/工具集/头像/推理档位)— `/platform/agents`,FE `views/agent/AgentList.vue` + `AgentEditorModal.vue` — `handler/custom_agent.go` + `routes_agent.go:24-52`
- 内置工具约 25 个(`internal/agent/tools/definitions.go`):search_knowledge、read_document、list_documents、query_knowledge_graph、search_conversations、search_memory、**database_query**、**data_analysis(duckdb)**、web_search、web_fetch、shell_exec(+async)、沙箱/技能文件读写、wiki 读写 8 件、ask_user;外部工具=MCP 服务
- MCP 服务接入(工具开关、逐工具审批、OAuth)— `/platform/toolbox/mcp`,FE `views/toolbox/Toolbox.vue` — `handler/mcp_service.go` + `routes_infra.go:159-196`
- 技能目录(ClawHub/SkillHub/Git/ZIP 安装)— `/platform/toolbox/skills` — `handler/skill_handler.go`、`skill_catalog.go`【改造:保留不动(③)】
- 沙箱(Docker/E2B/Cube;网络策略;交互终端 WS + 图形桌面 WS)— Toolbox + `views/chat/` — `handler/sandbox_config.go`、`routes_chat.go:178-191`【改造:保留不动(③)】
- 产物库 — `/platform/artifacts`,FE `views/artifacts/ArtifactLibrary.vue` — `routes_chat.go:130-136`
- 自动化(一次性/周期计划)— `/platform/automations`,FE `views/automation/AutomationList.vue` — `handler/automation.go`【改造:建议删除(②)】
- 执行权限与审批(只读/修改前确认/自动执行,会话授权 grants)— `views/chat/components/ToolApprovalCard.vue` — `handler/agent_grant.go`

### Wiki
- 自动 Wiki(从文档抽取实体生成带引用页面,目录/搜索/跨库搜索)— FE `views/knowledge/wiki/WikiBrowser.vue` — `handler/wiki_page.go` + `routes_knowledge.go:358-396`【加分项:保留】
- 版本历史与回滚 — `WikiRevisionDrawer.vue`
- 图谱视图、链接重建、lint/自动修复/issue 跟踪 — 同页
- Agent 可用 wiki_* 工具直接改页 — `internal/agent/tools/definitions.go:81-98`

### 空间与权限
- 多空间(租户)创建/切换/onboarding — FE `views/auth/WorkspaceOnboarding.vue` — `handler/tenant.go` + `routes_auth_tenant.go:61-123`【改造:保留,重映射为"班级"】
- 成员管理 + 邀请(邮箱邀请、**分享链接注册**、我的邀请待办)— FE `views/settings/TenantMembers.vue`(Share-link generator ~L90-136)— `handler/tenant_invitation.go`、`tenant_invite_link.go`;公开路由 `routes_auth_tenant.go:221-222`(`/auth/invitations/lookup`、`/auth/register-by-invite`)
- 四级角色 RBAC:owner / admin / contributor / viewer(`internal/types/tenant_member.go:21-32`);路由级角色门 `internal/middleware/rbac.go`(RequireRole)+ `internal/router/rbac.go`(KB/Chunk/Wiki 细粒度门 + API Key 能力策略)
- 组织(跨空间协作):建组织/邀请码/申请加入与审批/成员角色(admin/editor/viewer,`internal/types/organization.go:14-18`)— FE `views/organization/OrganizationList.vue` — `handler/organization.go`【改造:重映射为"学校/院系"层】
- 跨空间共享:**KB/Agent 共享给组织**(只读/可编辑)— FE `views/knowledge/settings/KBShareSettings.vue` — `routes_agent.go:191-216` + `internal/handler/kb_access.go`【改造:教务处校级库共享给组织 = 全校可读】
- 审计日志(空间级)— settings `system-audit-log` 区 — `handler/audit_log.go`【改造:教务处监督演示点】
- 权限范围 API Key — FE `views/system/PlatformAPIKeys.vue` — `routes_auth_tenant.go:104-110` + `router/rbac.go`

### 系统管理后台(系统管理员)
- 平台用户管理(建号/停用/重置密码/授予系统管理员)— FE `views/system/UserManagement.vue`、`CreateUserDialog.vue` — `handler/system.go` + `routes_auth_tenant.go:296-316`【改造:教务处角色】
- 模型目录(全平台发布)— FE `views/system/ModelCatalog.vue` — `handler/system_model_catalog.go`
- 运行时:任务队列面板/死信重试/解析进度 — FE `views/system/RuntimeQueues.vue`、`DocumentProgress.vue` — `router/task.go`、`task_inspector.go`
- 全局系统设置(存储/向量库/解析引擎连通性、配额)— FE `views/system/SystemSettings.vue`

### 集成与渠道
- **网站嵌入 Widget**(iframe 与 script 气泡两种、域名白名单、frame-ancestors CSP、token 轮换、统计、预览)— FE `components/AgentEmbedChannelPanel.vue` + 公开页 `views/embed/EmbedPage.vue`(embed.html) — `handler/embed_channel.go` + `routes_agent.go:250-296`
- IM 渠道(企业微信/飞书/QQBot/Slack/Telegram/钉钉/Mattermost/微信/云之家)— settings integration-im — `handler/im.go`【改造:建议删除(②)】
- 内置 MCP Server(Streamable HTTP,open 给 Cursor 等)— FE `views/integrations/McpServerIntegrationSettings.vue` — `handler/mcp_endpoint.go`(公开面 /mcp/:endpoint_id)+ `internal/mcpserver/`【加分项:保留】;`mcp-server/` Python 版已弃用(README_CN.md:351)【改造:目录删除】
- OpenAPI/SDK/CLI 集成页与 SSE Playground — FE `views/integrations/ApiIntegrationSettings.vue`、`CliIntegrationLanding.vue`【CLI 相关:随 CLI 删除(②)】
- 浏览器扩展/BrowserSkill — FE `views/integrations/ChromeExtensionLanding.vue` — `router.go:207-209`【改造:保留不动(③)】
- 网络搜索提供商(14 家)— FE `views/settings/WebSearchSettings.vue` — `handler/web_search_provider.go`【改造:教务场景可默认关闭,代码保留】

### 其他
- 模型管理(29 厂商、凭据 AES 加密、调试)— FE `views/settings/ModelSettings.vue` — `handler/model.go`、`model_credentials.go`【改造:演示配 1-2 家 API】
- 评测(召回命中率/BLEU/ROUGE)— `handler/evaluation.go` + `routes_infra.go:96-99`【改造:捞回保留——毕设实验章节的现成工具】
- 收藏(跨资源)— `api/user-favorites.ts` — `handler/user_resource_favorite.go`
- Langfuse 追踪(可关)— `internal/tracing/langfuse/`【改造:保留不动(③)】
- 客户端生态:`cli/`【删②】、`miniprogram/`(微信小程序)【删②+脱敏】、`client/`(Go SDK)【删②】、`examples/`

## 特别确认项
- (a) 空间邀请链接:**有**。FE `TenantMembers.vue`(Share-link generator)+ BE `tenant_invite_link.go`;注册流 `views/auth/Login.vue`(router/index.ts:59-68),公开 API `POST /auth/invitations/lookup`、`/auth/register-by-invite`。→ 学生进班级空间零开发。
- (b) 空间共享:**有**,实现为"KB/Agent 共享给组织"。FE `KBShareSettings.vue` + BE `organization.go`、`routes_agent.go:191-216`、校验 `kb_access.go`。→ 校级知识库共享给组织(全校)。
- (c) 知识图谱检索:**有**(Neo4j GraphRAG,见上)。【加分项保留】
- (d) 内置 MCP Server:**有**(Streamable HTTP;Python 版已弃用)。
- (e) 嵌入 Widget:**有**(iframe + script 双模式、CSP、白名单)。
- (f) Wiki:**有**(自动生成+编辑+版本回滚+lint)。
- (g) Agent 自定义/工具体系:**有**(约 25 内置工具 + MCP 外部工具 + 技能)。
- (h) 四级角色:owner / admin / contributor / viewer(`types/tenant_member.go:21-32`);组织级 admin/editor/viewer(`types/organization.go:14-18`)。
- (i) 批量导入用户:**无**(仅单个建号;批量只有 FAQ Excel 导入)。→ 不需要,邀请链接已覆盖。
- (j) 会话记忆/跨会话长期记忆:**有**(个人 + 空间级,确认制)。
- (k) 上传格式:PDF/Word/PPT/Excel/CSV/TXT/Markdown/HTML/EPUB/MHTML/JSON/XMind/图片;OCR=RapidOCR(`docreader/parser/image_parser.py`);表格=`excel_parser.py`、`xlsx_merge.py` + PDF 表格。

## 隐藏功能(README 未重点宣传,对教务场景可能有用)
1. 产物库(沙箱文件统一保存/预览/下载)— `/platform/artifacts`
2. 会话语音输入转写 — POST /sessions/:id/transcriptions(语音提问演示点)
3. 消息追问建议 — `handler/message_suggestion.go`
4. 图片画廊 + 图片证据编辑 — `views/knowledge/gallery/`
5. 跨资源收藏夹 — /user/favorites
6. 文档跨知识库移动 + 知识库复制/克隆 — /knowledge/move、/kb/copy、/:id/duplicate(校级模板库→各班克隆)
7. Wiki 质量工具(lint/auto-fix/issue)— `routes_knowledge.go:386-391`
8. API Playground(页内 SSE 调试)— `views/integrations/apiPlaygroundSSE.ts`
