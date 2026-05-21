# HTTP 路由性能对比

对比三种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter)，基于 radix tree
- **Gin**：[gin-gonic/gin](https://github.com/gin-gonic/gin)，基于 httprouter 的 Web 框架

## 运行方式

```bash
cd internal/radixperf
go test -benchmem -bench ./...
```

## 测试环境


| 项       | 值                                        |
| ------- | ---------------------------------------- |
| 机器      | Apple M2 Pro                             |
| OS/Arch | darwin / arm64                           |
| Go      | 1.26                                     |
| 测试方式    | `httptest` 直接调用 `ServeHTTP`，handler 为空实现 |


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


| 场景            | ServeMux | HTTPRouter | Gin    |
| ------------- | -------- | ---------- | ------ |
| 静态路由          | 120      | **23**     | 36     |
| 带参路由          | 172      | 49         | **48** |
| 嵌套参数          | 213      | **57**     | 57     |
| 根路由           | 46       | **15**     | 28     |
| 短静态           | 70       | **19**     | 31     |
| 300 路由 / 静态   | 137      | **25**     | 37     |
| 300 路由 / 参数   | 190      | 51         | **49** |
| 300 路由 / 最后一条 | 194      | 58         | **53** |


## 并发结果 (ns/op)


| 场景   | ServeMux | HTTPRouter | Gin     |
| ---- | -------- | ---------- | ------- |
| 静态路由 | 181      | 5.9        | **4.6** |
| 带参路由 | 187      | 18         | **9.3** |


## 内存分配 (allocs/op)


| 场景   | ServeMux | HTTPRouter | Gin   |
| ---- | -------- | ---------- | ----- |
| 静态路由 | 0        | 0          | 0     |
| 带参路由 | 1 (16B)  | 1 (32B)    | **0** |


## 结论

1. **ServeMux 最慢**：单线程比 radix tree 方案慢约 **3–5 倍**，并发场景差距更大（约 **30 倍**）。带参路由每次匹配还有 1 次堆分配（16B）。
2. **HTTPRouter 静态路由最快**：纯静态路径匹配约 **23 ns/op**，并发静态路由约 **5.9 ns/op**。路由数量从 16 增到 300 时，静态路由耗时几乎不变（120 → 137 vs 23 → 25），说明 radix tree 对路由规模不敏感。
3. **Gin 带参路由有优势**：单线程带参路由与 HTTPRouter 接近（~48 ns），并发带参路由明显更快（**9.3 vs 17.6 ns/op**），且 **零内存分配**（context pool 复用）。
4. **选型建议**：
  - 只需路由分发、追求极致静态性能 → **HTTPRouter**
  - 需要中间件、参数绑定等框架能力 → **Gin**（路由性能接近 HTTPRouter，并发更好）
  - 路由简单、依赖最少 → **ServeMux** 可用，但高 QPS 场景有明显性能差距

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发，路由选型对整体延迟的影响需结合具体场景评估。

