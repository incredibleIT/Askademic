# 代码结构

## 后端（Go）

- cmd/：服务入口（server、desktop 等）。
- internal/container/：依赖注入与组件装配。
- internal/router/ 与 internal/handler/：路由与 HTTP 层。
- internal/application/service/：业务服务层（知识库、会话、智能体、数据源、记忆等）。
- internal/agent/：Agent 引擎、工具注册、提示词组装、审批与协作。
- internal/datasource/：数据源连接器框架与飞书、钉钉实现。
- internal/types/：领域类型与配置结构。
- docreader/：Python 解析服务。

## 前端（Vue 3）

- frontend/src/views/：页面（首页、知识库、会话、智能体、设置等）。
- frontend/src/components/：通用组件（品牌字标、引用抽屉、工具结果等）。
- frontend/src/api/：后端接口封装。
- frontend/src/i18n/：六语言包与键审计测试。

## 文档与工具

- docs/：项目文档（前端状态、剪裁台账、棕地分析、swagger 契约）。
- config/：运行配置、内置智能体定义、提示词模板。
- website-docs/：本语料（嵌入二进制供智能体检索）。
