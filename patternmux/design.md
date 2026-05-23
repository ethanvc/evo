# patternmux 设计

通用 pattern 匹配引擎：注册 pattern 串，对输入串做匹配，返回挂载值、捕获段、以及条件性的 Converted 输出串。

与仓库内其他包的关系：

| 包 | 关系 |
|----|------|
| `httpmux` | HTTP method + `/` 分隔 + replace-only 的特化；未来可 built on `patternmux` |
| `httpsvr/ginradix` | 旧 path radix 实现；`patternmux` radix 后端可逐步替代 |

---

## 1. 目标与非目标

### 目标

- 支持字面量 + `{表达式}` 混合的 pattern 语法
- 泛型挂载值：`Mux[T]`，`Register(pattern, value)` / `Lookup(input)`
- 匹配结果包含：
  - `Node` 元信息（Raw / Canonical / HasKeep / CachedConverted）与挂载值
  - `Captures`（key 可为空串；Lookup 运行时返回，可 pool 复用）
  - `Converted`（输出串；replace-only 可缓存，含 keep 则每次 Lookup 计算）
- 不限于 HTTP path；分隔符不限于 `/`

### 非目标（v1）

- HTTP method 维度（由 `httpmux` 负责）
- 正则级通用规则引擎（不做「每个 pattern 编译成一个 regex」）
- `{keep}` + scan 后端的完整性能优化（v2）

---

## 2. 核心概念

本包引入以下概念，后文 §3 起展开语法与实现细节。

| 概念 | 说明 |
|------|------|
| **Pattern（模式串）** | 注册时写入的模板：字面量子串与 `{表达式}` 交替组成，用于描述「如何匹配输入、挂载何值」。 |
| **Expression（表达式）** | `{action[:name];rule1[;rule2...]}` 形式的占位段；从当前位置**消费**一段连续子串，多个 rule **同时**约束该段。 |
| **Action** | `replace`：wildcard 段，Canonical 变为 `:name` / `*name`，捕获值进 Captures；`keep`：匹配子串保留在 Canonical，Converted 中填入实际值。 |
| **Rule（消费规则）** | 描述如何消费后续子串的边界或字符约束（如 `until-slash`、`digit`）；至少一个，未知 rule 在 Register 时报错。 |
| **Canonical** | Register 期编译出的规范化模板（`:name`、`*name`、保留 `{keep;...}` 原文）；用于索引与 replace-only 输出。 |
| **Converted** | Lookup 成功时的输出串：replace-only 等于 Canonical 并可缓存；含 keep 时每次按输入现场拼装。 |
| **Node[T]** | 注册句柄，持有挂载值 `T` 与 Raw / Canonical / HasKeep / CachedConverted 等元信息；Lookup 返回匹配到的 leaf。 |
| **Captures** | Lookup 期提取的 `(key, value)` 列表，key 可为空；pool 分配，调用方须 `PutCaptures` 归还。 |
| **MatchBackend** | 按 rule 组合选择的匹配后端：**Radix**（`until-slash` / `rest` → radix 索引）与 **Scan**（`digit` / `hexdigit` / `keep` 等 → 线性扫描，v2）。 |
| **Mux[T]** | 泛型入口：`Register(pattern, value)` 建索引，`Lookup(input)` 返回 Node、Captures、Converted。 |

---

## 3. Pattern 语法

模式串由 **字面量** 与 **`{表达式}`** 交替组成。

### 3.1 表达式

```
{action[:name];rule1[;rule2;...]}
```

表达式以 **`;`** 分段：

| 位置 | 段 | 说明 |
|------|-----|------|
| 第 1 段 | `action[:name]` | `action` 必填；`name` 可选，以 `:` 与 action 分隔 |
| 第 2..n 段 | `rule` | **消费字符串的规则**，可列出 **多个**；**同时**作用于同一次消费 |

示例：

| 表达式 | action | name | rules（消费规则，同时生效） |
|--------|--------|------|----------------|
| `{replace::user-id;until-slash}` | replace | `:user-id` | `[until-slash]` |
| `{replace:*path;rest}` | replace | `*path` | `[rest]` |
| `{keep;digit}` | keep | — | `[digit]` |
| `{replace;hexdigit}` | replace | — | `[hexdigit]` |
| `{keep;digit;hexdigit}` | keep | — | `[digit, hexdigit]`（示意：多条消费规则组合） |

| 字段 | 必填 | 说明 |
|------|------|------|
| `action` | 是 | `replace` 或 `keep` |
| `name` | replace 可选 | `:ident` 或 `*ident`；仅作为结果串（Canonical / Converted）的**占位符格式**，本身不附带段分隔或 catch-all 语义——消费行为完全由 rule 控制 |
| `rule` | 至少 1 个 | 指定如何消费**本表达式之后**的连续子串；多个 rule **同时**作用于同一次消费 |

**多 rule 语义**：

- **rules 指定的是消费字符串的规则**：从当前位置起，本表达式要消费哪一段、消费多长、允许哪些字符，均由 rule 定义。
- 从当前位置出发，表达式**只消费一段连续**的后续子串。
- 列出多个 rule 时，它们**同时**参与这次消费，共同约束该段子串（长度边界 + 字符约束等）。
- **不是**管道式逐段消费（不是 rule1 消费一段、rule2 再消费下一段）。
- 匹配成功 ⟺ 存在一段子串，使**所有 rule 对该子串的消费判定同时为真**。
- rule 的书写顺序仅作声明顺序，不改变「同时生效」语义（实现时可按固定顺序求交集）。

示例：`{replace::id;until-slash;digit}` 同时消费一段子串，该子串须**既**止于 `/` 之前，**又**全为数字（如 `13455`），而不是先按 `/` 切一段再对下一段做 digit。

### 3.2 action

| action | 含义 |
|--------|------|
| `replace` | wildcard 段：参与路由索引；Canonical 中变为 `:name` / `*name`（仅输出格式，消费语义由 rule 决定）；Captured 值单独返回 |
| `keep` | 匹配并捕获，Canonical 保留完整 `{keep;rule1[;rule2...]}`；Converted 中填入本次匹配子串 |

### 3.3 rule（消费字符串的规则）

**rule** 描述从当前位置起，**如何消费**后续输入字符串。一个表达式可挂 **多个 rule**，它们同时约束同一次消费（见 §3.1）。

| rule | 消费约束 |
|------|---------|
| `until-slash` | 边界：本次消费止于下一个 `/` 之前（不含 `/`） |
| `until-blank` | 边界：本次消费止于下一个空白字符之前（不含空白） |
| `rest` | 边界：本次消费尽余下全部字符（catch-all；与其他边界 rule 并用时以语义兼容为准） |
| `digit` | 字符：本次消费的子串须为连续 `[0-9]+` |
| `hexdigit` | 字符：本次消费的子串须为连续 `[0-9a-fA-F]+` |

示例：

- `{replace::user-id;until-slash}`：单 rule，消费至 `/` 前，等价 httprouter `:user-id`
- `{keep;digit}`：单 rule，消费一段数字
- `{replace::id;until-slash;digit}`：两维同时消费——边界（至 `/`）与字符类（digit）叠加在同一段上

未知 rule 或未指定任何 rule：Register 时返回 error。

### 3.4 示例

**路径类（replace-only，Converted 可缓存）**

```
注册: /abc/{replace::user-id;until-slash}
输入: /abc/13455

Canonical:  /abc/:user-id
Converted:  /abc/:user-id          （与 Canonical 相同，注册期缓存）
Captures:   user-id = 13455
```

```
注册: /abc/{replace:*path;rest}
输入: /abc/a/b/c

Canonical:  /abc/*path
Converted:  /abc/*path
Captures:   path = a/b/c
```

**文本类（含 keep，Converted 不可缓存）**

```
注册: error code {keep;digit}, transacton-id is {replace;hexdigit}
输入: error code 123456, transaction-id is 123456abcd

Canonical:  error code {keep;digit}, transacton-id is {replace;hexdigit}
Converted:  error code 123456, transaction-id is     （keep 填值；replace 段不出现在输出串）
Captures:   "" = 123456
            "" = 123456abcd
```

> 注：示例注册串中 `transacton-id` 为原文拼写；是否按字面匹配由注册 pattern 决定。

---

## 4. Canonical 与 Converted

### 4.1 Canonical（编译期，Register 时计算）

对每个 segment：

| segment 类型 | Canonical 转换 |
|-------------|----------------|
| 字面量 | 原样保留 |
| `{replace::name;rules...}` | `:name`（rules 不参与 Canonical 字面，仅影响匹配） |
| `{replace:*name;rules...}` | `*name` |
| `{keep;rules...}` | 保留 `{keep;rules...}` 原文（含完整 rules） |

### 4.2 Converted（输出串）

| 模式 | 计算时机 | 是否缓存 |
|------|---------|---------|
| 仅含 `replace` | Register 时等于 Canonical | **可缓存**（`cachedConverted`） |
| 含 `keep` | 每次 Lookup 成功时按输入计算 | **不可缓存** |

含 `keep` 时 Converted 生成规则：

- 字面量 → 原样写入
- `{keep;rules...}` → 写入本次匹配到的子串
- `{replace;...}` → **不写入** Converted（值仅出现在 Captures）

replace-only 时：`Converted == Canonical == node.CachedConverted()`。

### 4.3 Node（注册句柄）

与 `httpmux` 一致：**注册期信息挂在 Node 上**，Lookup 返回匹配到的 leaf `*Node[T]`，而非临时 `Match` 结构体。

```go
type Node[T any] struct {
    // radix 索引内部字段（Radix 后端）或索引条目（Scan 后端）
    value           T
    raw             string // 注册原文
    canonical       string // 编译期模板
    hasKeep         bool
    cachedConverted string // 仅当 !hasKeep
    registered      bool
}

func (n *Node[T]) Value() T
func (n *Node[T]) Raw() string
func (n *Node[T]) Canonical() string
func (n *Node[T]) HasKeep() bool
func (n *Node[T]) CachedConverted() string // HasKeep 时为 undefined，勿用
```

Radix 后端下 Node 即 radix 索引的 leaf；Scan 后端下 Node 为索引条目，类型统一，匹配后端不同。

---

## 5. 匹配结果

```go
type Capture struct {
    Key   string // 可为空串
    Value string
}

type Captures []Capture

func (cs Captures) ByName(name string) string
```

**注册期 vs 运行期分离**（对齐 `httpmux`）：

| 数据 | 归属 | 说明 |
|------|------|------|
| `Value`、`Raw`、`Canonical`、`HasKeep`、`CachedConverted` | `*Node[T]` | Register 时确定，跨 Lookup 复用 |
| `Captures` | Lookup 返回 `*Captures` | 每次匹配提取；非 nil 时调用方须 `PutCaptures` |
| `Converted` | Lookup 返回值 | replace-only：`node.CachedConverted()`；含 keep：现场拼装 |

`Captures` 顺序与 pattern 中表达式出现顺序一致。

---

## 6. API

```go
package patternmux

func New[T any]() *Mux[T]

func (m *Mux[T]) Register(pattern string, value T) error

// Lookup 返回匹配到的注册 Node、捕获段、Converted 输出串。
// captures 非 nil 时，调用方须 PutCaptures(captures) 归还 pool。
func (m *Mux[T]) Lookup(input string) (node *Node[T], captures *Captures, converted string, ok bool)

func PutCaptures(cs *Captures)
```

调用示例：

```go
node, caps, converted, ok := mux.Lookup(input)
if !ok { /* no match */ }
if caps != nil {
    defer patternmux.PutCaptures(caps)
}
v := node.Value()
canonical := node.Canonical()
// converted：replace-only 时等于 node.CachedConverted()
```

### Register

1. parse pattern → AST（`Literal` | `Expr`）
2. 校验表达式（action / name / rules，至少一个 rule）
3. 计算 `Canonical`、`HasKeep`、`cachedConverted`，写入 `Node`
4. 插入匹配索引
5. 冲突处理：**相同 Raw 或相同挂载点重复注册 → 返回 error**（与 `httpmux` 一致，不允许静默覆盖）

### Lookup

1. 在索引中查找匹配的 pattern，得到 leaf `*Node[T]`
2. 提取 `Captures`（pool 分配）
3. 若 `!node.HasKeep()`：`converted = node.CachedConverted()`；否则现场拼装
4. 返回 `node, captures, converted, true`

---

## 7. 多 pattern 命中策略

**已确认：最长 literal 前缀优先。**

当多个 pattern 同时匹配同一输入时，选择 **literal 前缀累计长度最大** 的 pattern。长度相同时，以 **Register 顺序靠后** 的为准（后注册优先）。

> literal 前缀长度：从 pattern 起点开始，连续 literal segment 的字符数之和（不含 `{expr}` 占位段）。

---

## 8. 架构

推荐 **统一 AST + 按 rule 选匹配后端**，分阶段交付。

```
Register(pattern)
    → Parser → AST
    → Compiler → Node meta + MatchBackend
    → Index.Insert

Lookup(input)
    → Index.Search → *Node[T]
    → Matcher.Run(input, AST) → *Captures
    → Build Converted（HasKeep 时）
    → node, captures, converted
```

### 8.1 MatchBackend

| Backend | 典型 rule 组合 | 匹配实现 | 版本 |
|---------|---------------|---------|------|
| **Radix** | `until-slash`、`rest`（可与其他 rule 叠加） | radix 索引 | v1 |
| **Scan** | `digit`、`hexdigit` 及多 rule 组合 + keep/replace 混排 | 线性段扫描 / 编译状态机 | v2 |

v1 交付：

- 完整 Parser + Compiler（Canonical / HasKeep / cache 判定）
- Radix 后端匹配 + Lookup
- Scan 后端：**Register 可 parse，Lookup 暂不支持（v2）**

v2 交付：

- Scan 后端完整匹配
- keep / digit / hexdigit 的 Converted 拼装

### 8.2 与 httpmux 复用

Radix 后端的索引逻辑与 `httpmux/tree.go` 同族，差异：

- 无 HTTP method 维度
- 无「路径」专用抽象；pattern / input 均为普通字符串，消费语义由 rule 决定
- 表达式 `{replace::name;until-slash}` 在 Register 时 lowering 为索引 key，运行时索引不解析 `{}`
- `until-slash` / `rest` 的索引行为由 rule lowering 决定，而非 name 中的 `:`` / `*` 前缀
- Lookup 签名对齐：`httpmux` 为 `(*Node[T], *Params, tsr)`；`patternmux` 为 `(*Node[T], *Captures, converted, ok)`

---

## 9. 待定默认值（待确认）

以下尚未逐条确认，设计默认如下：

| 项 | 默认 |
|----|------|
| Register 冲突（相同 Canonical 不同 Value） | **不允许**，返回 error |
| v1 范围 | Radix 后端 + Parser 骨架；Scan 后端 v2 |

---

## 10. 测试

| 类别 | 内容 |
|------|------|
| Parser | 各 action/name/rules 组合；非法语法、未指定 rule error |
| Compiler | Canonical、HasKeep、cachedConverted 判定 |
| Golden | 本文 §3.4 三个示例 |
| 冲突 | 重复 Register error |
| 优先级 | 最长 literal 前缀 + 同长后注册优先 |
| Benchmark | replace-only 与 `until-slash` rule 的 pattern 与 `httpmux` 同量级（后续 `internal/radixperf` 扩展） |

---

## 11. 文件布局（计划）

```
patternmux/
  design.md          # 本文
  patternmux.go      # Mux[T], Register, Lookup
  ast.go             # AST 类型
  parse.go           # Parser
  compile.go         # Canonical / HasKeep / cache
  match_radix.go     # Radix 后端（v1）
  match_scan.go      # Scan 后端（v2）
  *_test.go
```

---

## 12. 版本计划

| 阶段 | 交付 |
|------|------|
| **v1** | Parser、Compiler、Radix 后端、`Mux[T]` API、replace-only Converted 缓存 |
| **v2** | Scan 后端、keep Converted 拼装、digit/hexdigit |
| **v3** | 与 `httpmux` / `httpsvr` 集成评估 |
