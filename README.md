# evohttp
Example: auto parse url param and body to request struct:
```golang
createUser := func(c context.Context, req *CreateUserReq)(*CreateUserResp, error){
	...
}
svr := NewServer()
svr.POST("/api/users/:user_id", NewStdHandlerF(createUser))
```

# evolog
Extend slog's function.
1. trace id support.
2. filed mask support. user can mark struct filed and then log library will mask sensitive data for you.

Examples: mask sensitive field.
```golang
type Abc struct {
    Name string `evolog:"ignore"`
}
abc := &Abc{Name: "test"}
slog.InfoContext(c, "MaskSensitiveField", slog.Any("abc", abc))
```

# base
Some base utils commonly used:
1. type safe SyncMap.
2. error code based Status inspired from grpc. So you can just process limited and unified error codes.

# evores
Library to obtain cpu/memory and others resources in process perspective.

This Library will consider cgroup limit.


