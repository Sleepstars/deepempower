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

### 多平台构建
```bash
# 构建所有支持的平台
make release-build

# 支持的平台包括：
# - linux/amd64
# - linux/arm64
# - darwin/amd64
# - darwin/arm64
# - windows/amd64
```

### Docker 构建和运行
```bash
# 构建 Docker 镜像
docker build -t deepempower:latest .

# 运行容器
docker run -d \
  --name deepempower \
  -p 8080:8080 \
  -v $(pwd)/configs:/etc/deepempower/configs \
  deepempower:latest

# 查看日志
docker logs -f deepempower

# 停止容器
docker stop deepempower
```

### GitHub Actions CI/CD

本项目使用 GitHub Actions 实现自动化的测试、构建和发布流程：

1. **自动化测试和构建**
   - 每次提交都会触发测试
   - 自动构建多平台二进制文件
   - 构建结果保存为 Artifacts

2. **Docker 镜像发布**
   - 当创建新的版本标签（v*）时触发
   - 自动构建多架构 Docker 镜像
   - 推送至 GitHub Container Registry (ghcr.io)

### 发布新版本

1. 使用 make 命令创建新的版本标签：
```bash
make release-tag TAG=v1.0.0
```

2. 等待 GitHub Actions 完成：
   - 自动构建多平台二进制文件
   - 构建并推送 Docker 镜像到 ghcr.io
   - 在 GitHub Releases 页面查看发布状态

3. 拉取最新版本 Docker 镜像：
```bash
docker pull ghcr.io/[username]/deepempower:1.0.0
```

### 环境变量
- `CONFIG_PATH`: 配置文件路径，默认为 `/etc/deepempower/configs`
- `LOG_LEVEL`: 日志级别，可选值：DEBUG, INFO, WARN, ERROR，默认为 INFO
- `PORT`: 服务端口，默认为 8080

### 配置挂载
使用 Docker 运行时，建议将配置文件挂载到容器中：
- 配置文件路径：`/etc/deepempower/configs`
- 示例：`-v $(pwd)/configs:/etc/deepempower/configs`

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
