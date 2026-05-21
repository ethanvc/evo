# HTTP 路由性能对比

对比四种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter)，基于 radix tree
- **HttpMux**：本仓库 `httpmux`，基于 httprouter 改造的泛型 radix tree，benchmark 调用 `Lookup`
- **Gin**：[gin-gonic/gin](https://github.com/gin-gonic/gin)，基于 httprouter 的 Web 框架

## 运行方式

```bash
cd internal/radixperf
go test -benchmem -bench ./...
```

## 测试环境

| 项       | 值                                                                 |
| ------- | ----------------------------------------------------------------- |
| 机器      | Apple M2 Pro                                                      |
| OS/Arch | darwin / arm64                                                    |
| Go      | 1.26                                                              |
| 测试方式    | ServeMux / HTTPRouter / Gin：`httptest` 调用 `ServeHTTP`，handler 为空实现 |
|         | HttpMux：直接调用 `Lookup`，不经过 HTTP 栈                                  |

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

## 单线程结果 (ns/op，越低越好)

| 场景            | ServeMux | HTTPRouter | HttpMux | Gin    |
| ------------- | -------- | ---------- | ------- | ------ |
| 静态路由          | 123      | 24         | **20**  | 37     |
| 带参路由          | 174      | 51         | 110     | **50** |
| 嵌套参数          | 212      | **58**     | 118     | 58     |
| 根路由           | 48       | 15         | **12**  | 28     |
| 短静态           | 71       | 20         | **15**  | 32     |
| 300 路由 / 静态   | 139      | 25         | **22**  | 38     |
| 300 路由 / 参数   | 195      | 52         | 110     | **49** |
| 300 路由 / 最后一条 | 193      | **58**     | 116     | 54     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin     |
| ---- | -------- | ---------- | ------- | ------- |
| 静态路由 | 180      | 6.3        | **2.0** | 4.2     |
| 带参路由 | 181      | 17         | 47      | **11**  |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux  | Gin   |
| ---- | -------- | ---------- | -------- | ----- |
| 静态路由 | 0        | 0          | 0        | 0     |
| 带参路由 | 1 (16B)  | 1 (32B)    | 2 (56B)  | **0** |

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 方案慢约 **3–5 倍**，并发场景差距更大（约 **30–90 倍**）。带参路由每次匹配还有 1 次堆分配（16B）。
2. **HttpMux 静态路由最快**：纯静态路径匹配约 **20 ns/op**，并发静态路由约 **2.0 ns/op**，优于 HTTPRouter 和 Gin。路由数量从 16 增到 300 时，静态路由耗时几乎不变（123 → 139 vs 20 → 22），说明 radix tree 对路由规模不敏感。
3. **HTTPRouter / Gin 带参路由更快**：单线程带参路由 HttpMux 约 **110 ns/op**（`Lookup` 返回 `Params` 有 2 次分配），HTTPRouter / Gin 约 **50–58 ns/op**。并发带参路由 Gin 最快（**11 ns/op**），HTTPRouter 次之（**17 ns/op**）。
4. **选型建议**：
  - 只需路由查找、追求极致静态性能 → **HttpMux**
  - 需要完整 HTTP handler 分发、带参路由性能优先 → **HTTPRouter** 或 **Gin**
  - 需要中间件、参数绑定等框架能力 → **Gin**
  - 路由简单、依赖最少 → **ServeMux** 可用，但高 QPS 场景有明显性能差距

> 以上结论基于空 handler 的路由匹配开销。HttpMux 测的是 `Lookup`，其他三者测的是 `ServeHTTP`，横向对比时需考虑这一差异。实际业务中 handler 逻辑通常远大于路由分发，路由选型对整体延迟的影响需结合具体场景评估。
