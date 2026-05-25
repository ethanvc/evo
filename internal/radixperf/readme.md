# HTTP 路由性能对比

对比五种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) `@master`（commit `4840180`，2024-01-30），基于 radix tree
- **HttpMux**：本仓库 `httpmux`，基于 httprouter master 改造的泛型 radix tree
- **PatternMux**：本仓库 `patternmux`，`{replace::name;until-slash}` 语法，所有 rule 统一在一棵 radix tree 内，wildcard 节点按 spec 消费输入
- **Gin**：[gin-gonic/gin](https://github.com/gin-gonic/gin)，基于 httprouter 的 Web 框架

## 运行方式

```bash
cd internal/radixperf
go test -run=^$ -bench=. -benchmem -count=5
```

## 测试环境

| 项       | 值                                                                 |
| ------- | ----------------------------------------------------------------- |
| 机器      | Apple M1 Pro                                                      |
| OS/Arch | darwin / arm64                                                    |
| Go      | 1.26.1                                                            |
| 带参路由    | 四者均通过 `httptest` 调用 `ServeHTTP`，handler 为空实现                    |
| 静态路由    | ServeMux / HTTPRouter / Gin / PatternMux：`ServeHTTP` 或等价；HttpMux / PatternMux 静态：`Lookup` + `PutCaptures` |

本页数值取 `-count=5` 的中位数，单位为 `ns/op`，除非表格另有说明。

HttpMux 带参场景通过 benchmark 内 `httpMuxBenchHandler` 适配为 `http.Handler`：`Lookup` 拿到 `*Params` 后调用 `PutParams` 归还全局 pool。

PatternMux 将 `:id` 路径转为 `/path/{replace::id;until-slash}` 注册；**无 HTTP method 维度**，同 path 不同 method 的重复注册会跳过（路由条数略少于其他实现）。

## 测试场景

| Benchmark              | 说明                                    |
| ---------------------- | ------------------------------------- |
| `StaticRoute`          | 静态路由 `GET /api/v1/users`              |
| `ParamRoute`           | 单参数路由 `GET /api/v1/users/12345`       |
| `ParamNestedRoute`     | 嵌套参数 `GET /api/v1/users/12345/orders` |
| `RootRoute`            | 根路由 `GET /`                           |
| `ShortStaticRoute`     | 短静态路由 `GET /health`                   |
| `ManyRoutes_Static`    | 300 条路由后查静态路由                         |
| `ManyRoutes_Param`     | 300 条路由后查参数路由                         |
| `ManyRoutes_Last`      | 300 条路由后查最后一条参数路由                     |
| `Parallel_StaticRoute` | 并发静态路由                                |
| `Parallel_ParamRoute`  | 并发参数路由                                |
| `ParamRoute_Lookup`    | 补充对比：直接调用 `Lookup`（见下方说明）            |

## 单线程结果 (ns/op，越低越好)

| 场景            | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin    |
| ------------- | -------- | ---------- | ------- | ---------- | ------ |
| 静态路由          | 101.7    | 20.2       | 16.5    | **13.9**   | 34.6   |
| 根路由           | 39.8     | 13.6       | 9.8     | **3.8**    | 25.8   |
| 短静态路由         | 58.8     | 17.1       | 12.3    | **8.5**    | 30.1   |
| 带参路由          | 148.6    | 35.3       | **33.5** | 36.9       | 43.9   |
| 嵌套参数          | 190.4    | 43.4       | **40.4** | 50.2       | 51.5   |
| 300 路由 / 静态   | 118.6    | 22.1       | 18.6    | **14.2**   | 34.2   |
| 300 路由 / 参数   | 173.3    | 36.8       | **33.6** | 34.9       | 43.6   |
| 300 路由 / 最后一条 | 176.8    | 41.4       | **38.5** | 39.8       | 45.9   |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin    |
| ---- | -------- | ---------- | ------- | ---------- | ------ |
| 静态路由 | 178.6    | 4.9        | 4.2     | **1.8**    | 6.6    |
| 带参路由 | 175.7    | **5.2**    | 5.5     | 7.1        | 11.0   |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin   |
| ---- | -------- | ---------- | ------- | ---------- | ----- |
| 静态路由 | 0        | 0          | 0       | 0          | 0     |
| 带参路由 | 1 (16B)  | 0          | 0       | **0**      | 0     |

## Lookup 补充对比 (ns/op)

| 实现         | ns/op | allocs/op | 说明                          |
| ---------- | ----- | --------- | --------------------------- |
| HTTPRouter | 90.5  | 2 (56B)   | 未归还 params pool（库无导出 API）  |
| HttpMux    | **33.0** | 0         | `Lookup` + `PutParams`      |
| PatternMux | 35.5  | 0         | `Lookup` + `PutCaptures`    |

## 本次 PatternMux 改动回归检查

本次改动把 `Converted` 的 replace-only 输出统一为 `Pattern`，并移除了 `Canonical` / `CachedConverted` 路径。为避免不同机器或系统负载干扰，额外在同一台机器上用临时 `HEAD` worktree 跑了改动前基线，仅比较 PatternMux 子基准：

```bash
go test -run=^$ -bench='/PatternMux$' -benchmem -count=5
```

| 场景 | 改动前 | 当前 | 变化 |
| ---- | ------ | ---- | ---- |
| 静态路由 | 14.08 | 14.03 | -0.4% |
| 带参路由 | 35.43 | 35.70 | +0.8% |
| 嵌套参数 | 50.56 | 50.58 | +0.0% |
| 根路由 | 3.811 | 3.805 | -0.2% |
| 短静态路由 | 8.525 | 8.570 | +0.5% |
| 300 路由 / 静态 | 14.27 | 14.25 | -0.1% |
| 300 路由 / 参数 | 35.00 | 35.03 | +0.1% |
| 300 路由 / 最后一条 | 39.79 | 39.70 | -0.2% |
| 并发静态路由 | 1.834 | 1.798 | -2.0% |
| Lookup 带参 | 34.78 | 34.92 | +0.4% |
| 并发带参路由 | 6.476 | 6.062 | -6.4% |

结论：没有观察到性能退化。所有 PatternMux 场景仍为 **0 B/op、0 allocs/op**；单线程差异基本落在 ±1% 内，并发项波动较大但当前结果不慢于基线。

HTTPRouter 的 `Lookup` 按值返回 `Params` 且 benchmark 无法归还 pool，会显示 2 次 alloc。**带参 `ServeHTTP` 主表才是公平对比**，radix 方案均为 0 allocs。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 慢约 **3–5 倍**；带参路由有 1 次堆分配（16B）。
2. **PatternMux 静态路由最快**：单线程 **13.9 ns/op**、根路由 **3.8 ns/op**、并发 **1.8 ns/op**；统一 radix tree 没有 method 维度，静态前缀查找开销最低。
3. **PatternMux 带参路由与 HTTPRouter / HttpMux 同量级**：单线程 **~35–50 ns / 0 allocs**；含 `until-slash` / `digit` / `hexdigit` / `until-blank` / `keep` 等 rule 都共用同一棵 tree。嵌套参数因每层 wildcard 都要做 capture 截断回溯，比 HttpMux 慢约 24%，与 Gin 接近。
4. **HttpMux 与 HTTPRouter master 同量级**：带参 `ServeHTTP` 均为 **~33–43 ns / 0 allocs**。
5. **`PutCaptures` / `PutParams` 是关键**：`Lookup` 返回指针后必须归还 pool，否则 slice 逃逸、性能退化。
6. **选型建议**：
   - 通用 pattern 语法（含 digit / hexdigit / blank / keep 等 rule）→ **PatternMux**
   - 纯 HTTP method + path → **HttpMux**
   - handler 分发、中间件框架 → **Gin**
   - 路由简单、依赖最少 → **ServeMux**

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发。
