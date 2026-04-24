---
name: gater-conventions
description: Gater 项目开发规范与约定。当用户完成代码变更需要生成 commit message、编写或审查代码风格、撰写文档、或变更后需要自检时触发此 Skill。
---

# Gater 项目开发规范

## 概述

本 Skill 定义了 Gater 项目在开发过程中的核心规范，包括 Git 提交消息、代码风格、文档规范和变更后自检清单。所有代码变更和文档编写都应遵循这些约定。

---

## 1. Git Commit Message 规范

### 1.1 语言与格式

- **语言**：英文
- **格式**：Conventional Commits（不带 scope）

```
<type>: <description>

- Detail line 1
- Detail line 2
- Detail line 3
```

### 1.2 Type 列表

| Type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `refactor` | 代码重构（不改变功能） |
| `docs` | 文档变更 |
| `style` | 代码格式调整（不影响逻辑） |
| `test` | 测试相关 |
| `chore` | 构建/工具/依赖变更 |
| `perf` | 性能优化 |

### 1.3 示例

```
fix: auto-restore running test session on page refresh

- On page mount, fetch sessions first then auto-select the session
  with testStatus=running and isOnline=true
- This restores the charging panel, stop button, and charging info
  polling after a page refresh
- Watch selectedSession changes to keep isTestRunning in sync
  with backend testStatus from session list polling
```

```
fix: correct auth_test to match actual bytesToBCD and computeAuthHash behavior

- Fix TestBytesToBCD: expected values should be hex ASCII strings,
  not mathematical BCD decoded bytes (bytesToBCD converts bytes to
  uppercase hex ASCII per protocol requirement, verified with charger)
- Fix TestComputeAuthHash_AlgorithmSteps: add hex.DecodeString for
  fixedKey to match computeAuthHash step 2 logic
- These test failures were pre-existing, not introduced by recent changes
```

### 1.4 规则

- **不使用 scope**：写 `fix:` 而非 `fix(web):`
- description 首字母小写，不加句号
- 使用祈使句（imperative mood）：`add` 而非 `added`
- Body 使用 `- ` 列表，每条详细描述做了什么、为什么这样做
- 列表项可跨行，续行缩进 2 空格对齐
- 单次 commit 只做一件事，避免混合多种 type

---

## 2. 代码风格

### 2.1 Go 代码风格

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 命名规范：
  - 包名：小写单个单词，不使用下划线或驼峰
  - 导出函数/变量：大写字母开头（PascalCase）
  - 未导出函数/变量：小写字母开头（camelCase）
  - 接口名：以 `-er` 后缀表示行为（如 `Reader`, `Writer`）
  - 常量：使用 camelCase，不使用全大写
- 错误处理：
  - 不使用 panic，使用 error 返回值
  - 错误信息小写开头，不加句号
  - 使用 `fmt.Errorf("context: %w", err)` 包装错误
- 注释：
  - 导出符号必须有文档注释
  - 注释以符号名开头：`// ReadBCD reads ...`
  - 包注释写在 `doc.go` 或包内任意源文件顶部
- 函数：每个函数后面要保留一个空行，不要使用单行函数

### 2.2 Vue / TypeScript 代码风格

- 组件文件名：PascalCase（如 `PriceTable.vue`）
- 组件名称：PascalCase
- Props / Emits：使用 `defineProps` / `defineEmits`，添加 TypeScript 类型
- 变量/函数：camelCase
- 常量：UPPER_SNAKE_CASE
- 使用 Composition API + `<script setup>`
- 模板中属性顺序：`v-if` / `v-for` → `v-model` → 事件绑定 → 其他属性
- CSS 使用 `<style scoped>`，类名使用 kebab-case

### 2.3 通用规则

- 缩进：Go 用 Tab，Vue/TS 用 2 空格
- 行宽：软限制 120 字符
- 文件末尾保留一个空行
- 不提交 IDE 配置文件（`.idea/`, `.vscode/` 等，已在 .gitignore 中）

---

## 3. 文档规范

### 3.1 文档位置

- 设计文档：`docs/design/`
- API 文档：`docs/api/`
- 指南文档：`docs/guide/`
- 项目根目录：`README.md`, `ONBOARDING_GUIDE.md`

### 3.2 格式要求

- 使用 Markdown（`.md`）格式
- 中文文档：标题和正文使用中文，代码术语保留英文
- 代码示例使用带语言标注的代码块
- 表格使用 Markdown 表格语法
- 文件名使用中文或英文均可，保持目录内一致

### 3.3 内容要求

- 设计文档需包含：背景、目标、方案、示例
- 代码相关文档需包含：具体示例（输入 → 输出）
- 指南文档需面向新手，循序渐进

---

## 4. 变更后自检清单

每次完成代码变更后，按以下清单逐项检查：

### 4.1 代码质量

- [ ] 代码能编译通过（`go build ./...` / `npm run build`）
- [ ] 无 linter 错误（`go vet ./...` / `npm run lint`）
- [ ] 无明显逻辑错误
- [ ] 错误处理完善（不吞错误、不遗漏 error 返回值）

### 4.2 功能正确性

- [ ] 变更符合需求描述
- [ ] 边界条件已处理
- [ ] 无硬编码的测试值残留

### 4.3 代码风格

- [ ] 符合上述代码风格规范
- [ ] 新增导出符号有文档注释
- [ ] 无不必要的 `fmt.Println` / `console.log` 调试语句

### 4.4 Git 提交

- [ ] 变更分类正确（feat/fix/refactor/docs 等）
- [ ] Commit message 遵循 Conventional Commits 格式
- [ ] Commit message 使用英文
- [ ] 单次提交范围合理，不混合不相关变更

### 4.5 文档（如适用）

- [ ] 相关文档已同步更新
- [ ] 新增功能有对应文档说明
- [ ] 示例代码可运行

---

## 资源

本 Skill 不需要 scripts、references 或 assets 目录中的资源文件。
