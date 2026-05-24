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
  - `Node` 公开 accessor：`Value` / `GetPattern` / `GetPatternWithExpr` / `HasKeep`（`Canonical`、`CachedConverted` 是内部字段，仅 Mux 自用）
  - `Captures`（key 可为空串；Lookup 运行时返回，可 pool 复用）
  - `Converted`（输出串；replace-only 由内部缓存返回，含 keep 则每次 Lookup 计算）
- 不限于 HTTP path；分隔符不限于 `/`

### 非目标

- HTTP method 维度（由 `httpmux` 负责）
- 正则级通用规则引擎（不做「每个 pattern 编译成一个 regex」）
- 关心 pattern 是否「像路径」；本包只有「字面量 + rule 消费」，无 path 抽象

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
| **Node[T]** | 注册句柄，持有挂载值 `T` 与若干 Pattern 视图（公开：`GetPattern` / `GetPatternWithExpr` / `HasKeep`；内部：`canonical`、`cachedConverted`）；Lookup 返回匹配到的 leaf。 |
| **Captures** | Lookup 期提取的 `(key, value)` 列表，key 可为空；pool 分配，调用方须 `PutCaptures` 归还。 |
| **WildcardSpec** | 编译期由 rule 推导出的通配符消费规格：`(boundary, class, keep, name)`。所有 rule（含 `digit` / `hexdigit` / `until-blank` / `rest` / `keep`）都内化为 spec 字段，由统一 radix tree 在 Lookup 时按 spec 消费输入。 |
| **Mux[T]** | 泛型入口：`Register(pattern, value)` 建索引，`Lookup(input)` 返回 Node、Captures、Converted。 |

---

## 3. Pattern 语法

模式串由 **字面量** 与 **`{表达式}`** 交替组成。

### 3.1 表达式

**语法规则**（一条规则描述全部结构）：

1. 整体由 `{` 和 `}` 包围。
2. 大括号内部由 `;` 分段，记作 `segment₁ ; segment₂ ; … ; segmentₙ`。
3. **`segment₁` 必须是 `action`**，形式 `action[:name]`：
   - `action` ∈ {`replace`, `keep`}，必填。
   - `name` 是 action 的可选后缀，以 `:` 与 action 分隔；对所有 action 通用。
4. `segment₂..segmentₙ` 是 `rule`（消费规则），**至少 1 个**；多个 rule 同时作用于本表达式消费的同一段子串（详见 §3.3）。

形式化模板：

```
{action[:name];rule1[;rule2;...]}
```

示例：

| 表达式 | action | name | rules（消费规则，同时生效） |
|--------|--------|------|----------------|
| `{replace::user-id;until-slash}` | replace | `:user-id` | `[until-slash]` |
| `{replace:*path;rest}` | replace | `*path` | `[rest]` |
| `{replace;hexdigit}` | replace | — | `[hexdigit]` |
| `{keep:err-code;digit}` | keep | `err-code` | `[digit]` |
| `{keep;digit}` | keep | — | `[digit]` |
| `{keep;digit;hexdigit}` | keep | — | `[digit, hexdigit]`（多条消费规则组合） |

字段说明：

| 字段 | 必填 | 说明 |
|------|------|------|
| `action` | 是 | 见 §3.2。 |
| `name` | 否 | **表达式的标识符**，主要用于 `Node.GetPattern()` 输出中替换该表达式；对 `replace` 还会出现在 `Canonical` 里（典型 `:ident` / `*ident`）。`name` 本身不附带段分隔或 catch-all 语义——消费行为完全由 rule 决定；`replace::id` 与 `replace:*id` 的差异仅在 `name` 字面（进而影响 `Canonical` / `GetPattern` 字面），匹配语义统一由 rule 控制。`name` 缺省时，`GetPattern` 用 `PlaceholderName`（默认 `noname`）顶替。 |
| `rule` | 至少 1 个 | 见 §3.3。 |

**违例（Register 阶段报错）**：

| 形态 | 错误 |
|---|---|
| `{` 与 `}` 不匹配 | `ErrInvalidSyntax` |
| `{}` 或 `segment₁` 为空（缺 action） | `ErrInvalidSyntax` |
| `segment₁` 不是已知 action | `ErrInvalidSyntax` |
| `action:` 后 name 为空（如 `{replace:;...}`、`{keep:;...}`） | `ErrInvalidSyntax` |
| 只有 action 没有 rule（如 `{replace::id}`） | `ErrMissingRule` |
| rule 段为空（如 `{replace::id;}`） | `ErrInvalidSyntax` |
| rule 不在已知集合 | `ErrUnknownRule` |

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
| `replace` | wildcard 段：参与路由索引；`Canonical` 中变为 `name` 字面（典型 `:ident` / `*ident`）或在 unnamed 时省略；Captured 值单独返回 |
| `keep` | wildcard 段：参与路由索引；`Canonical` 保留完整 `{keep[:name];rule1[;rule2...]}`（含可选 name）；`Converted` 中填入本次匹配子串 |

> `name` 与 action 正交：所有 action 都可挂或不挂 name，只是出场位置不同——`replace` 的 name 同时出现在 `Canonical` 与 `GetPattern`，`keep` 的 name 出现在 `Canonical`（包在 `{...}` 原文里）与 `GetPattern`。两者**都不影响匹配语义**。

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

**命名 keep**（`keep` 也可挂 name；只影响 `Canonical` / `GetPattern` 字面，匹配语义不变）

```
注册: error code {keep:err-code;digit}, tx {replace::id;hexdigit}
输入: error code 12, tx dead

Canonical:    error code {keep:err-code;digit}, tx :id
GetPattern:   error code err-code, tx :id
Converted:    error code 12, tx                （keep 填值；replace 不出现）
Captures:     err-code = 12
              :id      = dead
```

> 注：示例注册串中 `transacton-id` 为原文拼写；是否按字面匹配由注册 pattern 决定。

---

## 4. Canonical 与 Converted

`Compile` 产出 `compiledPattern`，其中四个字符串字段从不同角度描述同一条注册 pattern。**记忆口诀**：

| 字段 | 口诀 | 说明 |
|------|------|------|
| **Raw** | 你写了什么 | Register 原文；表达式 `{...}` 原封不动。对应 `Node.GetPatternWithExpr()`。 |
| **Canonical** | Mux 认为「这是哪条路由」（内部） | Register 期去重 key；replace-only 时兼任输出模板。不暴露 getter。 |
| **Pattern** | 给人看的标签（监控/日志） | 每个表达式替换成 `name`；unnamed → `PlaceholderName`（`noname`）。对应 `Node.GetPattern()`。 |
| **CachedConverted** | replace-only 的 Lookup 输出缓存（内部） | `!HasKeep` 时等于 Canonical；含 keep 时为空，Lookup 现场拼装。用户拿 `Lookup` 返回的 `converted` 即可。 |

```
Register 输入
    │
    ▼
  Raw ────────────── 原文存档（byRaw 去重、GetPatternWithExpr）
    │
    ▼ Compile
    ├── Canonical ── 内部去重 key（byCanonical）；replace-only 时 = CachedConverted
    ├── Pattern ──── 人读路由 ID（GetPattern）
    └── CachedConverted
          ├─ !HasKeep → = Canonical（Lookup 直接返回）
          └─  HasKeep → ""（Lookup 现场 assembleConverted）
```

### 4.1 Canonical（编译期，Register 时计算）

对每个 segment：

| segment 类型 | Canonical 转换 |
|-------------|----------------|
| 字面量 | 原样保留 |
| `{replace:name;rules...}` | `name` 字面（典型 `:ident` / `*ident`）；rules 不参与 Canonical |
| `{replace;rules...}` | 不贡献（unnamed replace 在 Canonical 中省略） |
| `{keep[:name];rules...}` | 保留 `{keep[:name];rules...}` 原文（含可选 name 和完整 rules） |

### 4.2 Converted（输出串）

| 模式 | 计算时机 | 是否缓存 |
|------|---------|---------|
| 仅含 `replace` | Register 时等于 Canonical | **可缓存**（`cachedConverted`） |
| 含 `keep` | 每次 Lookup 成功时按输入计算 | **不可缓存** |

含 `keep` 时 Converted 生成规则：

- 字面量 → 原样写入
- `{keep;rules...}` → 写入本次匹配到的子串
- `{replace;...}` → **不写入** Converted（值仅出现在 Captures）

replace-only 时：`Converted == Canonical`，并被缓存到内部 `cachedConverted` 字段，`Lookup` 命中时直接返回该缓存。

### 4.3 Node（注册句柄）

与 `httpmux` 一致：**注册期信息挂在 Node 上**，Lookup 返回匹配到的 leaf `*Node[T]`，而非临时 `Match` 结构体。

```go
type Node[T any] struct {
    // 注册期元信息（所有节点共用，仅 leaf 上有意义）
    value           T
    raw             string // 注册原文，含 `{...}` 表达式
    canonical       string // 去重 / 路由模板 ID（内部使用）
    pattern         string // 表达式替换为 name 后的形态（unnamed → PlaceholderName）
    hasKeep         bool
    cachedConverted string // replace-only 时的预计算 Converted（内部使用）
    registered      bool

    // 树内部：static 节点用 prefix / indices / children，
    // wildcard 节点用 spec 描述如何消费输入。
}

// 公共 API（仅这 4 个）：
func (n *Node[T]) Value() T                   // 注册值
func (n *Node[T]) GetPatternWithExpr() string // 注册原文（调试用）
func (n *Node[T]) GetPattern() string         // 表达式 → name；unnamed → PlaceholderName（监控/日志）
func (n *Node[T]) HasKeep() bool              // 输出是否动态
```

> `canonical` 与 `cachedConverted` 是**内部字段**：前者只在 `Mux.Register` 中用作去重 key（用户感知到的是 `ErrDuplicateCanonical`）；后者只在 `Mux.Lookup` 内部用作 replace-only 路径的输出缓存（用户感知到的是 `Lookup` 返回的 `converted`）。两者都不暴露 getter。

**例 1**：unnamed keep + 命名 replace

pattern = `error code {keep;digit}, tx {replace::id;hexdigit}`

| 字段 / 输出 | 值 | 公开吗 |
|---|---|---|
| `GetPatternWithExpr()` | `error code {keep;digit}, tx {replace::id;hexdigit}` | ✓ |
| `GetPattern()` | `error code noname, tx :id` | ✓ |
| `HasKeep()` | `true` | ✓ |
| 内部 `canonical` | `error code {keep;digit}, tx :id` | ✗ |
| 内部 `cachedConverted` | （空，含 keep 不缓存） | ✗ |
| Lookup 返回 `converted`（运行期，输入 `error code 42, tx deadbeef`） | `error code 42, tx ` | ✓ |

**例 2**：命名 keep + 命名 replace（`name` 与 action 正交）

pattern = `error code {keep:err-code;digit}, tx {replace::id;hexdigit}`

| 字段 / 输出 | 值 |
|---|---|
| `GetPatternWithExpr()` | `error code {keep:err-code;digit}, tx {replace::id;hexdigit}` |
| `GetPattern()` | `error code err-code, tx :id` |
| 内部 `canonical` | `error code {keep:err-code;digit}, tx :id` |

无论 pattern 中含哪些 rule，`Node[T]` 都是同一棵统一 radix tree 的节点；不再区分 Radix / Scan 后端。

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
| `Value`、`GetPatternWithExpr`、`GetPattern`、`HasKeep` | `*Node[T]`（公开） | Register 时确定，跨 Lookup 复用 |
| `canonical`、`cachedConverted` | `*Node[T]`（内部字段） | 仅 Mux 自用：去重、replace-only 路径输出缓存 |
| `Captures` | Lookup 返回 `*Captures` | 每次匹配提取；非 nil 时调用方须 `PutCaptures` |
| `Converted` | Lookup 返回值 | replace-only：内部 `cachedConverted`；含 keep：现场拼装 |

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
routeID := node.GetPattern() // 例如 "/api/v1/users/:id"，用于监控/日志
// converted：业务输出串。replace-only 时由 Mux 缓存返回，含 keep 时现场拼装
```

### Register

1. parse pattern → AST（`Literal` | `Expr`）
2. 校验表达式（action / name / rules，至少一个 rule）
3. 计算 `canonical`（内部去重 key）、`hasKeep`、`cachedConverted`，写入 `Node`
4. 插入匹配索引
5. 冲突处理：**相同 Raw 或相同 `canonical` 重复注册 → 返回 error**（与 `httpmux` 一致，不允许静默覆盖）

### Lookup

1. 在索引中查找匹配的 pattern，得到 leaf `*Node[T]`
2. 提取 `Captures`（pool 分配）
3. 若 `!node.HasKeep()`：`converted` 取内部 `cachedConverted`；否则现场拼装
4. 返回 `node, captures, converted, true`

---

## 7. 多 pattern 命中策略

**已确认：最长 literal 前缀优先。**

当多个 pattern 同时匹配同一输入时，选择 **literal 前缀累计长度最大** 的 pattern。长度相同时，以 **Register 顺序靠后** 的为准（后注册优先）。

> literal 前缀长度：从 pattern 起点开始，连续 literal segment 的字符数之和（不含 `{expr}` 占位段）。

---

## 8. 架构

**统一 radix tree**：所有 pattern 不论 rule 组合，都注册进同一棵 tree。Lookup 时按 tree 结构走静态前缀，遇到 wildcard 节点再按 spec 消费输入；不再做 Radix / Scan 分流。

```
Register(pattern)
    → Parser → AST（Literal | Expr）
    → Compiler → Raw / Canonical / HasKeep / CachedConverted / LiteralPrefix
    → tree.addPattern（segments 直接驱动插入；Expr → wildcardSpec 节点）

Lookup(input)
    → tree.matchInput → *Node[T] + *Captures
    → 含 keep 时按 leaf.segments + captures 拼装 Converted
    → node, captures, converted
```

### 8.1 树结构

每个节点要么是 **static**（携带 `prefix string` 的字面量节点），要么是 **wildcard**（携带 `spec wildcardSpec` 的通配段节点）。任一节点同时可挂：

- `children []*Node[T]` + `indices string`：按首字节索引的静态子节点
- `wildcards []*Node[T]`：通配子节点列表

匹配时先查静态（最长字面量前缀优先，自然契合 §7），未命中再按 `wildcards` 逐个尝试；通配子树失败时通过 `Captures` 截断回溯。

### 8.2 wildcardSpec

由 Compile 期把 `Expr` 的 rule 列表归约成：

| 字段 | 取值 | 决定 |
|------|------|------|
| `boundary` | `none` / `slash` / `blank` | 消费何时停（无、`/`、空白） |
| `class` | `any` / `digit` / `hex` | 消费的字符类 |
| `keep` | `bool` | 仅影响 Converted 拼装，不参与路由 |
| `name` | `string` | Capture key |

`consume(input)` 用一个循环同时检查 boundary 与 class，长度 > 0 才视为命中——单 rule（如 `until-slash`）与多 rule（如 `until-slash;digit`）走同一段代码。

### 8.3 多 wildcard 共位

同一父节点下若出现多个不同 spec 的 wildcard（典型：`/u/{:id;digit}` 与 `/u/{:name;until-slash}`），按 **后注册优先** 的顺序排列（`descendOrAddWildcard` 把新 wildcard 头插到列表），逐一尝试，第一个完整命中（含子树）的胜出，落空时回溯 `Captures`。这与 §6 的 tie-break 一致。

### 8.4 Converted 拼装与并发

- replace-only：Register 期算好 `cachedConverted`，Lookup 直接返回，**无 alloc**
- 含 `keep`：leaf 节点持有 `segments []Segment`，Lookup 用 `sync.Pool` 借出 byte buffer 现场拼装；返回的 `string` 独立持有 → **多 goroutine Lookup 并发安全**

### 8.5 与 httpmux 复用

整体结构与 `httpmux/tree.go` 同族（radix + 通配子节点 + 优先级），差异：

- 无 HTTP method 维度
- 无「路径」专用抽象；pattern / input 均为普通字符串，消费语义由 spec 决定
- 通配节点用 `wildcardSpec` 而非固定的 param/catchAll 二选一，可承载 `digit` / `hexdigit` / `until-blank` 等非路径 rule
- 同一父节点允许并存多 wildcard 子节点；httpmux 仅允许一个
- Lookup 签名对齐：`httpmux` 为 `(*Node[T], *Params, tsr)`；`patternmux` 为 `(*Node[T], *Captures, converted, ok)`

---

## 9. 默认值

| 项 | 默认 |
|----|------|
| Register 冲突（相同 Raw 或相同 Canonical） | **不允许**，返回 error |
| 表达式消费长度 | 必须 > 0；为空段 Lookup miss |
| Lookup 是否需吃完 input | **是**；trailing 字符无对应段则 miss |

---

## 10. 测试

| 类别 | 内容 |
|------|------|
| Parser | 各 action/name/rules 组合；非法语法、未指定 rule error |
| Compiler | Canonical、HasKeep、CachedConverted、LiteralPrefix |
| Golden | 本文 §3.4 三个示例 |
| Tree | until-slash / rest / until-blank / digit / hexdigit 命中；static 优先于同位 wildcard |
| 多 rule 交叉 | until-slash + digit 取边界交集 |
| 多 wildcard 共位 | 不同 spec 的 wildcard 共存于同父节点，按后注册优先回溯 |
| 冲突 | 重复 Raw / Canonical → error |
| 并发 | 多 goroutine 并发 Lookup 配合 `-race` |
| Benchmark | 与 `httpmux` / `httprouter` / `gin` 同量级（`internal/radixperf` 维护） |

---

## 11. 文件布局

```
patternmux/
  design.md          # 本文
  ast.go             # AST 类型
  parse.go           # Parser
  compile.go         # Canonical / HasKeep / CachedConverted / LiteralPrefix
  tree.go            # 统一 radix tree：static + wildcardSpec 节点、消费循环
  patternmux.go      # Mux[T]、Register、Lookup、Converted 拼装 + buffer pool
  captures.go        # Captures 与 pool
  node.go            # Node[T] 公开访问器
  errors.go          # 包级 error
  *_test.go
```

---
