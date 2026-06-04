# 贡献指南

感谢参与 `oss-maintainer-kit`。本项目优先接受能提升开源维护效率的改动。

## 开发流程

1. Fork 仓库并创建功能分支。
2. 修改代码前先补充或更新测试。
3. 运行格式化和测试。
4. 提交 PR，并说明变更动机、测试结果和潜在风险。

```bash
gofmt -w .
go test ./...
go build ./cmd/oss-maintainer-kit
```

## 适合贡献的方向

- 新增 issue 分类规则。
- 改进发布说明分组逻辑。
- 支持 GitHub API 输入。
- 增加 Markdown、JSON、SARIF 等输出格式。
- 补充边界条件测试。

## PR 要求

- 保持最小改动。
- 不引入不必要依赖。
- 新行为必须有测试覆盖。
- CLI 输出变更需要同步更新 README 示例。
