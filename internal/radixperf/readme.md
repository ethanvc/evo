# HTTP 路由性能对比

对比四种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) `@master`（commit `4840180`，2024-01-30），基于 radix tree
- **HttpMux**：本仓库 `httpmux`，基于 httprouter master 改造的泛型 radix tree
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
| 带参路由    | 四者均通过 `httptest` 调用 `ServeHTTP`，handler 为空实现                    |
| 静态路由    | ServeMux / HTTPRouter / Gin：`ServeHTTP`；HttpMux：`Lookup` + `PutParams` |

HttpMux 带参场景通过 benchmark 内 `httpMuxBenchHandler` 适配为 `http.Handler`：`Lookup` 拿到 `*Params` 后调用 `PutParams` 归还全局 pool，生命周期与 HTTPRouter `ServeHTTP` 一致。

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

| 场景            | ServeMux | HTTPRouter | HttpMux | Gin    |
| ------------- | -------- | ---------- | ------- | ------ |
| 静态路由          | 121      | 23         | **19**  | 36     |
| 带参路由          | 175      | **41**     | 41      | 48     |
| 嵌套参数          | 215      | **50**     | 51      | 57     |
| 根路由           | 47       | 15         | **12**  | 28     |
| 短静态           | 71       | 20         | **15**  | 31     |
| 300 路由 / 静态   | 137      | 25         | **22**  | 38     |
| 300 路由 / 参数   | 192      | 42         | 42      | 48     |
| 300 路由 / 最后一条 | 195      | 48         | 48      | 54     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin    |
| ---- | -------- | ---------- | ------- | ------ |
| 静态路由 | 179      | 2.4        | 5.2     | **4.5** |
| 带参路由 | 183      | 9.8        | **4.4** | 9.0    |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin   |
| ---- | -------- | ---------- | ------- | ----- |
| 静态路由 | 0        | 0          | 0       | 0     |
| 带参路由 | 1 (16B)  | **0**      | **0**   | **0** |

## Lookup 补充对比 (ns/op)

| 实现         | ns/op | allocs/op | 说明                          |
| ---------- | ----- | --------- | --------------------------- |
| HTTPRouter | 105   | 2 (56B)   | 未归还 params pool（库无导出 API）  |
| HttpMux    | 41    | **0**     | `Lookup` + `PutParams`      |

HTTPRouter 的 `Lookup` 按值返回 `Params` 且 benchmark 无法归还 pool，会显示 2 次 alloc；HttpMux 通过 `PutParams` 归还后达到 0 alloc。**带参 `ServeHTTP` 主表才是公平对比**，两者均为 ~41 ns / 0 allocs。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 慢约 **3–5 倍**；带参路由有 1 次堆分配（16B）。
2. **HttpMux 与 HTTPRouter master 同量级**：带参 `ServeHTTP` 均为 **~41–51 ns / 0 allocs**，泛型没有引入 measurable 开销。
3. **HttpMux 静态路由略快**：纯 `Lookup` 约 **19 ns**，300 条路由规模下几乎不变（137 → 22 vs 121 → 19）。
4. **并发带参 HttpMux 略优**：**~4.4 ns**，HTTPRouter ~9.8 ns，Gin ~9.0 ns（均 0 alloc）。
5. **`PutParams` 是关键**：`Lookup` 返回 `*Params` 后必须调用 `PutParams` 归还全局 pool，否则 slice 逃逸、pool 泄漏，性能会退化到 ~100 ns / 2 allocs。
6. **选型建议**：
   - 静态路径查找 → **HttpMux**（`Lookup`）
   - handler 分发、带参 0 alloc → **HttpMux**（`Lookup` + `PutParams`）或 **HTTPRouter**
   - 中间件、参数绑定等框架能力 → **Gin**
   - 路由简单、依赖最少 → **ServeMux**

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发。
