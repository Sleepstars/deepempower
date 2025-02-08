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

```bash
# 构建
go build -o deepempower ./cmd/server

# 运行
./deepempower
```

## 注意事项

1. 模型特性
   - Normal模型支持所有标准参数
   - Reasoner模型不支持temperature等参数

2. 性能优化
   - 使用流式处理减少延迟
   - 实现了上下文缓存
   - 支持并发请求处理

## 许可证

MIT License
