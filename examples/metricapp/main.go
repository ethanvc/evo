package main

import (
	"context"
	"fmt"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/plog"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"log/slog"
)

func main() {
	fx.New().Run()
}

// https://learn.microsoft.com/en-gb/aspnet/web-api/overview/data/using-web-api-with-entity-framework/part-5
// DTO对象，用于传输和对外输出实体属性

type QueryUserReq struct {
	Uid int64
}

type UserDto struct {
}

type userController struct {
	redisCli redis.UniversalClient
}

func (controller *userController) QueryUser(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	// query redis cache, case can downgrade
	switch base.Code(err) {
	case codes.OK:
		plog.ReportEvent(c, "UserFoundInCache")
		return
	case codes.NotFound:
		plog.ReportEvent(c, "UserNotFoundInCache")
	default:
		plog.ReportErrEvent(c, "UserCacheUnknownErr")
		slog.ErrorContext(c, "UserCacheUnknownErr")
	}
	return
}

func (controller *userController) queryUserFromCache(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	cmd := controller.redisCli.Get(c, fmt.Sprintf("a_%d", req.Uid))
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return
}
