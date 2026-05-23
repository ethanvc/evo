# HTTP 路由性能对比

对比五种路由实现的路径分发性能：

- **ServeMux**：Go 标准库 `net/http.ServeMux`（Go 1.22+ 支持 `{param}` 路径参数）
- **HTTPRouter**：[julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) `@master`（commit `4840180`，2024-01-30），基于 radix tree
- **HttpMux**：本仓库 `httpmux`，基于 httprouter master 改造的泛型 radix tree
- **PatternMux**：本仓库 `patternmux`，`{replace::name;until-slash}` 语法，Register 时 lowering 为 `:name` radix tree
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
| 静态路由    | ServeMux / HTTPRouter / Gin / PatternMux：`ServeHTTP` 或等价；HttpMux / PatternMux 静态：`Lookup` + `PutCaptures` |

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
| 静态路由          | 100      | 20         | 17      | **11**     | 31     |
| 带参路由          | 150      | 35         | 36      | **28**     | 39     |
| 嵌套参数          | 168      | 39         | 41      | **35**     | 46     |
| 根路由           | —        | —          | —       | —          | —      |
| 短静态           | —        | —          | —       | —          | —      |
| 300 路由 / 静态   | 106      | 19         | 17      | **11**     | 30     |
| 300 路由 / 参数   | 150      | 33         | 33      | **29**     | 38     |
| 300 路由 / 最后一条 | 151      | 37         | 37      | **34**     | 42     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin    |
| ---- | -------- | ---------- | ------- | ---------- | ------ |
| 静态路由 | 167      | 2.1        | 1.8     | **1.2**    | 3.7    |
| 带参路由 | 167      | 6.9        | 5.3     | **4.8**    | 10     |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin   |
| ---- | -------- | ---------- | ------- | ---------- | ----- |
| 静态路由 | 0        | 0          | 0       | 0          | 0     |
| 带参路由 | 1 (16B)  | 0          | 0       | **0**      | 0     |

## Lookup 补充对比 (ns/op)

| 实现         | ns/op | allocs/op | 说明                          |
| ---------- | ----- | --------- | --------------------------- |
| HTTPRouter | 81    | 2 (56B)   | 未归还 params pool（库无导出 API）  |
| HttpMux    | 32    | **0**     | `Lookup` + `PutParams`      |
| PatternMux | **28**| **0**     | `Lookup` + `PutCaptures`    |

HTTPRouter 的 `Lookup` 按值返回 `Params` 且 benchmark 无法归还 pool，会显示 2 次 alloc。**带参 `ServeHTTP` 主表才是公平对比**，radix 方案均为 0 allocs。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 慢约 **3–5 倍**；带参路由有 1 次堆分配（16B）。
2. **PatternMux 静态路由最快**：纯 `Lookup` 约 **11 ns/op**，并发约 **1.2 ns/op**；与 HttpMux 同族 radix，无 method 维度开销更小。
3. **PatternMux 带参路由略快于 HttpMux/HTTPRouter**：单线程 **~28–35 ns / 0 allocs**；`PutCaptures` 复用 pool，与 Gin 同为 0 alloc。
4. **HttpMux 与 HTTPRouter master 同量级**：带参 `ServeHTTP` 均为 **~33–40 ns / 0 allocs**。
5. **`PutCaptures` / `PutParams` 是关键**：`Lookup` 返回指针后必须归还 pool，否则 slice 逃逸、性能退化。
6. **选型建议**：
   - 通用 pattern 语法、路径查找 → **PatternMux**
   - HTTP method + path → **HttpMux**
   - handler 分发、中间件框架 → **Gin**
   - 路由简单、依赖最少 → **ServeMux**

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发。
