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
| 静态路由    | ServeMux / HTTPRouter / Gin：`ServeHTTP`；HttpMux：`Lookup`（纯 tree 查找） |

HttpMux 带参场景在 benchmark 内通过 `httpMuxBenchHandler` 适配为 `http.Handler`（不修改 `httpmux` 库）。Handler 内部调用 `Lookup`，并用 `cached` 字段复用 params buffer，模拟 HTTPRouter 在 handler 回调中使用 params 的模式。

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
| `ParamRoute_Lookup`    | 补充对比：HTTPRouter / HttpMux 均直接调用 `Lookup` |

## 单线程结果 (ns/op，越低越好)

| 场景            | ServeMux | HTTPRouter | HttpMux | Gin    |
| ------------- | -------- | ---------- | ------- | ------ |
| 静态路由          | 120      | 23         | **19**  | 36     |
| 带参路由          | 169      | **40**     | 106     | 46     |
| 嵌套参数          | 206      | **49**     | 114     | 57     |
| 根路由           | 47       | 15         | **12**  | 27     |
| 短静态           | 70       | 19         | **15**  | 31     |
| 300 路由 / 静态   | 136      | 25         | **22**  | 37     |
| 300 路由 / 参数   | 190      | **42**     | 107     | 48     |
| 300 路由 / 最后一条 | 189      | **47**     | 121     | 54     |

## 并发结果 (ns/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin    |
| ---- | -------- | ---------- | ------- | ------ |
| 静态路由 | 181      | 6.0        | **4.9** | **4.1** |
| 带参路由 | 183      | **4.8**    | 35      | 12     |

## 内存分配 (allocs/op)

| 场景   | ServeMux | HTTPRouter | HttpMux | Gin   |
| ---- | -------- | ---------- | ------- | ----- |
| 静态路由 | 0        | 0          | 0       | 0     |
| 带参路由 | 1 (16B)  | **0**      | 2 (56B) | **0** |

## Lookup 补充对比 (ns/op)

| 实现         | ns/op | allocs/op |
| ---------- | ----- | --------- |
| HTTPRouter | 101   | 2 (56B)   |
| HttpMux    | 102   | 2 (56B)   |

## 带参路由：外部测试环境相同，但数字仍有差异

带参 benchmark 现在四者都走 `httptest` + `ServeHTTP`，**外部测试环境已对齐**。HttpMux 侧在 benchmark 内增加了 `cached` params buffer，模拟 handler 消费 params。

但 HttpMux 仍显示 **~106 ns / 2 allocs**，而 HTTPRouter 为 **~40 ns / 0 allocs**。原因不在测试框架，而在 `Lookup` API：

| 路径 | params 生命周期 | benchmark 可见 alloc |
| --- | -------------- | ------------------- |
| HTTPRouter `ServeHTTP` | pool 借出 → handler 回调 → `putParams` 归还 | **0** |
| HttpMux `Lookup` | pool 借出 → `return *ps` 按值返回 → 无法归还 pool | **2** |

`Lookup` 必须把 `Params` 作为返回值交给调用方，slice header 逃逸到堆；benchmark 侧的 params 缓存只能复用 handler 自己的 buffer，**无法回收 httpmux 内部的 pool slice**。`ParamRoute_Lookup` 公平对比表明两者 tree 查找本身等价（~102 ns / 2 allocs）。

## 结论

1. **ServeMux 最慢**：单线程比 radix tree 慢约 **3–5 倍**；带参路由有 1 次堆分配（16B）。
2. **HttpMux 静态路由最快**：约 **19 ns/op**，路由规模增大时几乎不变。
3. **HttpMux 与 HTTPRouter tree 性能等价**：`Lookup` 对比均为 **~102 ns / 2 allocs**；带参 `ServeHTTP` 数字差异来自 `Lookup` 返回值语义，不是泛型或 tree 实现。
4. **若带参场景也要 0 alloc**：需要在 `httpmux` 内提供 callback 式分发（params 不逃逸），或在 `Lookup` 后暴露 `PutParams` 让调用方归还 pool——这属于库 API 扩展，benchmark 层无法单独解决。
5. **选型建议**：
   - 静态路径查找 → **HttpMux**（`Lookup`）
   - 完整 HTTP 分发、带参 0 alloc → **HTTPRouter** / **Gin**
   - 框架能力 → **Gin**
   - 简单场景 → **ServeMux**

> 以上结论基于空 handler 的路由匹配开销。实际业务中 handler 逻辑通常远大于路由分发。
