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
| 静态路由          | 119      | 23         | 20      | **18**     | 36     |
| 根路由           | 46       | 15         | 11      | **5**      | 27     |
| 短静态路由         | 69       | 19         | 14      | **11**     | 31     |
| 带参路由          | 170      | **41**     | 42      | 46         | 47     |
| 嵌套参数          | 208      | **50**     | 51      | 64         | 57     |
| 300 路由 / 静态   | 135      | 25         | 22      | **18**     | 37     |
| 300 路由 / 参数   | 189      | **42**     | 44      | 48         | 49     |
| 300 路由 / 最后一条 | 187      | **48**     | 49      | 52         | 53     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin    |
| ---- | -------- | ---------- | ------- | ---------- | ------ |
| 静态路由 | 181      | 2.3        | 5.4     | **1.8**    | 3.9    |
| 带参路由 | 181      | **4.5**    | 7.2     | 5.0        | 11.9   |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | PatternMux | Gin   |
| ---- | -------- | ---------- | ------- | ---------- | ----- |
| 静态路由 | 0        | 0          | 0       | 0          | 0     |
| 带参路由 | 1 (16B)  | 0          | 0       | **0**      | 0     |

## Lookup 补充对比 (ns/op)

| 实现         | ns/op | allocs/op | 说明                          |
| ---------- | ----- | --------- | --------------------------- |
| HTTPRouter | 102   | 2 (56B)   | 未归还 params pool（库无导出 API）  |
| HttpMux    | **41**| 0         | `Lookup` + `PutParams`      |
| PatternMux | 45    | 0         | `Lookup` + `PutCaptures`    |

HTTPRouter 的 `Lookup` 按值返回 `Params` 且 benchmark 无法归还 pool，会显示 2 次 alloc。**带参 `ServeHTTP` 主表才是公平对比**，radix 方案均为 0 allocs。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 慢约 **3–5 倍**；带参路由有 1 次堆分配（16B）。
2. **PatternMux 静态路由最快**：单线程 **18 ns/op**、根路由 **5 ns/op**、并发 **1.8 ns/op**；统一 radix tree 没有 method 维度，静态前缀查找开销最低。
3. **PatternMux 带参路由与 HTTPRouter / HttpMux 同量级**：单线程 **~46–64 ns / 0 allocs**，并发 **5 ns / 0 allocs**；含 `until-slash` / `digit` / `hexdigit` / `until-blank` / `keep` 等 rule 都共用同一棵 tree。嵌套参数因每层 wildcard 都要做 capture 截断回溯，比 HTTPRouter 慢约 25%，仍领先 Gin。
4. **HttpMux 与 HTTPRouter master 同量级**：带参 `ServeHTTP` 均为 **~41–50 ns / 0 allocs**。
5. **`PutCaptures` / `PutParams` 是关键**：`Lookup` 返回指针后必须归还 pool，否则 slice 逃逸、性能退化。
6. **选型建议**：
   - 通用 pattern 语法（含 digit / hexdigit / blank / keep 等 rule）→ **PatternMux**
   - 纯 HTTP method + path → **HttpMux**
   - handler 分发、中间件框架 → **Gin**
   - 路由简单、依赖最少 → **ServeMux**

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发。
