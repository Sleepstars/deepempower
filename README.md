# DeepEmpower

DeepEmpower是一个结合Normal(Claude)和Reasoner(R1)两个模型能力的混合智能体系统。该系统通过统一的OpenAI兼容接口，提供更强大的综合性AI能力。

## 特性

- OpenAI兼容的API接口
- 多模型协同处理
- 流式思维链输出
- 可定制化处理流程
- 插件式架构设计

## 项目结构

```
DeepEmpower/
├── cmd/
│   └── server/          # 主服务器入口
├── configs/
│   ├── models.yaml      # 模型配置
│   └── prompts/         # prompt模板
├── internal/
│   ├── config/          # 配置定义
│   ├── models/          # 数据模型
│   └── orchestrator/    # 处理流程编排
└── docs/
    └── DESIGN.md        # 架构设计文档
```

## 配置说明

### 模型配置 (configs/models.yaml)
```yaml
models:
  Normal:
    api_base: "..."
    default_params:
      temperature: 0.7
  Reasoner:
    api_base: "..."
    special_handling:
      disabled_params: ["temperature", "top_p"]
```

### Prompt模板 (configs/prompts/)
- pre_process.md: 需求分析和预处理
- reasoning.md: 深度思考和推理
- post_process.md: 结果优化和总结

## API使用

### Chat Completions
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "hybrid_v1",
    "messages": [
      {"role": "user", "content": "你的问题"}
    ],
    "stream": true
  }'
```

## 开发指南

1. 添加新的处理阶段
   - 实现`PipelineStage`接口
   - 在`configs/models.yaml`中注册新阶段
   - 创建对应的prompt模板

2. 自定义Prompt
   - 在`configs/prompts/`目录下创建新的模板
   - 使用`{{.Variable}}`语法引用上下文变量

3. 错误处理
   - 实现了自动重试机制
   - 支持降级方案
   - 详细的错误日志

## 构建和运行

### 本地构建
```bash
# 构建
go build -o deepempower ./cmd/server

# 运行
./deepempower
```

### Docker 部署方式

#### 使用 Docker Compose（推荐）
```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 手动 Docker 构建和运行
```bash
# 构建 Docker 镜像
docker build -t deepempower:latest .

# 运行容器（使用默认配置）
docker run -d \
  --name deepempower \
  -p 8080:8080 \
  -v $(pwd)/configs:/etc/deepempower/configs \
  deepempower:latest

# 使用自定义配置路径运行
docker run -d \
  --name deepempower \
  -p 8080:8080 \
  -v $(pwd)/my-configs:/custom/config/path \
  -e CONFIG_PATH=/custom/config/path \
  deepempower:latest

# 查看日志
docker logs -f deepempower

# 停止容器
docker stop deepempower
```

### 环境变量
- `CONFIG_PATH`: 配置文件路径，默认为 `/etc/deepempower/configs`
- `LOG_LEVEL`: 日志级别，可选值：DEBUG, INFO, WARN, ERROR，默认为 INFO
- `PORT`: 服务端口，默认为 8080

### 配置挂载
配置文件可以通过以下方式挂载：
1. Docker Compose (推荐)：
   - 在 docker-compose.yml 中通过 volumes 配置
   - 通过环境变量 CONFIG_PATH 指定路径
2. Docker 运行：
   - 默认配置路径：`/etc/deepempower/configs`
   - 自定义路径：使用 -v 参数挂载并设置 CONFIG_PATH 环境变量

## 注意事项

1. 模型特性
   - Normal模型支持所有标准参数
   - Reasoner模型不支持temperature等参数

2. 性能优化
   - 使用流式处理减少延迟
   - 实现了上下文缓存
   - 支持并发请求处理

3. Docker 部署
   - 镜像基于 Alpine Linux，体积小巧
   - 支持配置文件外部挂载
   - 建议在生产环境使用 Docker 部署

## 许可证

MIT License
