# HTTP 路由性能对比

对比四种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) `@master`（commit `4840180`，2024-01-30），基于 radix tree
- **HttpMux**：本仓库 `httpmux`，基于 httprouter master 改造的泛型 radix tree，benchmark 调用 `Lookup`
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
| `ParamRoute_Lookup`    | 公平对比：HTTPRouter / HttpMux 均调用 `Lookup` |

## 单线程结果 (ns/op，越低越好)

| 场景            | ServeMux | HTTPRouter | HttpMux | Gin    |
| ------------- | -------- | ---------- | ------- | ------ |
| 静态路由          | 120      | 23         | **19**  | 36     |
| 带参路由          | 169      | **40**     | 101     | 47     |
| 嵌套参数          | 207      | **49**     | 110     | 56     |
| 根路由           | 47       | 15         | **12**  | 27     |
| 短静态           | 70       | 19         | **15**  | 31     |
| 300 路由 / 静态   | 136      | 25         | **22**  | 38     |
| 300 路由 / 参数   | 190      | **42**     | 101     | 48     |
| 300 路由 / 最后一条 | 187      | **47**     | 107     | 52     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin    |
| ---- | -------- | ---------- | ------- | ------ |
| 静态路由 | 181      | 6.2        | **5.3** | **4.1** |
| 带参路由 | 183      | **7.3**    | 45      | 8.3    |

## 内存分配 (allocs/op)

| 场景                        | ServeMux | HTTPRouter (ServeHTTP) | HttpMux (Lookup) | Gin   |
| ------------------------- | -------- | ---------------------- | ---------------- | ----- |
| 静态路由                      | 0        | 0                      | 0                | 0     |
| 带参路由                      | 1 (16B)  | **0**                  | 2 (56B)          | **0** |
| 带参路由 Lookup 公平对比（见下表） | —        | 2 (56B)                | 2 (56B)          | —     |

## Lookup 公平对比 (ns/op)

同一路由表、同样调用 `Lookup`，排除 HTTP 栈和测试方式差异：

| 实现         | ns/op | allocs/op |
| ---------- | ----- | --------- |
| HTTPRouter | 102   | 2 (56B)   |
| HttpMux    | 101   | 2 (56B)   |

**结论：HttpMux 与 HTTPRouter master 的 Lookup 性能基本一致，泛型不是瓶颈。**

## 分析：为什么带参路由看起来 HttpMux 慢 2.5 倍？

之前对比 HTTPRouter **v1.3.0 tag** 时，HttpMux 带参路由约 110 ns vs HTTPRouter 约 51 ns，容易误判为泛型或 tree 实现退化。换成 **master** 并重跑后，原因更清晰：

### 1. 测试路径不同（主因）

| 实现         | 测试 API    | 带参路由 ns/op | allocs |
| ---------- | --------- | ----------- | ------ |
| HTTPRouter | `ServeHTTP` | **40**      | **0**  |
| HttpMux    | `Lookup`    | 101         | 2      |
| HTTPRouter | `Lookup`    | 102         | 2      |

`ServeHTTP` 在 handler 回调内传递 `Params`，slice 始终留在 `sync.Pool` 里，benchmark 看不到堆分配。`Lookup` 必须 `return handle, *ps, tsr`，`*ps` 按值返回导致 slice header 逃逸（2 次 alloc，56B）。**同一代码库的 Lookup 路径开销相同。**

### 2. v1.3.0 vs master 的差异（次要）

|                | v1.3.0 (2019)              | master / HttpMux (2024)     |
| -------------- | -------------------------- | --------------------------- |
| `getValue` 签名  | `(handle, p Params, tsr)`  | `(handle, ps *Params, tsr)` + pool |
| 节点字段           | 每节点 `maxParams uint8`      | 无 per-node maxParams         |
| 带参 ServeHTTP 参考 | ~51 ns / 1 alloc           | ~40 ns / 0 alloc            |

HttpMux 继承的是 master 的 pool+callback 模型，与 v1.3.0 的 per-node `maxParams` 优化不是同一条演化线。但 **Lookup 公平对比下两者性能相同**，说明 HttpMux 的泛型改造没有引入额外开销。

### 3. 并发带参路由 HttpMux 偏慢

并发 `Parallel_ParamRoute` 下 HttpMux（45 ns）仍慢于 HTTPRouter ServeHTTP（7.3 ns），部分因为 Lookup 返回值的逃逸在并行 benchmark 里被放大；HTTPRouter 的 ServeHTTP 路径仍无 alloc。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 方案慢约 **3–5 倍**，并发差距更大。带参路由每次匹配有 1 次堆分配（16B）。
2. **HttpMux 静态路由最快**：纯静态路径约 **19 ns/op**，与 HTTPRouter master 同量级且略快；路由规模从 16 增到 300 时耗时几乎不变。
3. **HttpMux 带参 Lookup 与 HTTPRouter master 等价**：公平对比均为 **~101 ns / 2 allocs**，泛型不是性能问题。
4. **HTTPRouter / Gin 的 ServeHTTP 带参路径更快**：约 **40–48 ns / 0 allocs**，因为 `Params` 不逃逸出路由层。若 HttpMux 也需要 0 alloc，应提供类似 ServeHTTP 的回调 API，或在调用方用完 params 后 `Put` 回 pool。
5. **选型建议**：
   - 只需路由查找、静态路径极致性能 → **HttpMux**
   - 需要完整 HTTP handler 分发、带参路由 0 alloc → **HTTPRouter** 或 **Gin**
   - 需要中间件、参数绑定等框架能力 → **Gin**
   - 路由简单、依赖最少 → **ServeMux** 可用，高 QPS 有明显差距

> 以上结论基于空 handler 的路由匹配开销。主表横向对比时 HTTPRouter/Gin 测 `ServeHTTP`、HttpMux 测 `Lookup`，带参场景需结合「Lookup 公平对比」一节解读。实际业务中 handler 逻辑通常远大于路由分发。
