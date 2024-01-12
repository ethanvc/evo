package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evohttp"
	"github.com/ethanvc/evo/plog"
	"github.com/ethanvc/evo/rediscli"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"net"
	"net/http"
	"time"
)

func main() {
	fx.New(
		fx.Provide(newRedisClient),
		fx.Provide(newUserController),
		fx.Provide(newHttpServer),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

func newHttpServer(lc fx.Lifecycle, user *userController) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	evoSvr := evohttp.NewServer()
	userGroup := evoSvr.SubBuilder("/api/user")
	userGroup.POST("/get", evohttp.NewStdHandlerF(user.QueryUser))
	srv.Handler = evoSvr
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func newRedisClient() *rediscli.RedisClient {
	opt := &redis.UniversalOptions{}
	cli := redis.NewUniversalClient(opt)
	return rediscli.NewRedisClient(cli, nil)
}

// https://learn.microsoft.com/en-gb/aspnet/web-api/overview/data/using-web-api-with-entity-framework/part-5
// DTO对象，用于传输和对外输出实体属性

type QueryUserReq struct {
	Uid int64
}

type UserDto struct {
	Uid int64
}

type userController struct {
	redisCli *rediscli.RedisClient
}

func newUserController(redisCli *rediscli.RedisClient) *userController {
	return &userController{
		redisCli: redisCli,
	}
}

func (controller *userController) QueryUser(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	// here no need to record the error, because import event will be recorded in place.
	resp, err = controller.queryUserFromCache(c, req)
	if err == nil {
		return resp, nil
	}
	return &UserDto{
		Uid: 1,
	}, nil
}

func (controller *userController) queryUserFromCache(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	// for request resolved network, have to record time cost and response content.
	c = plog.WithLogContext(c, nil)
	defer func() { plog.RequestLog(c, err, req, resp) }()
	c, cancel := context.WithTimeoutCause(c, time.Millisecond*100,
		base.New(codes.DeadlineExceeded, "GetFromRedisTimeout").Err())
	defer cancel()
	err = controller.redisCli.Get(c, fmt.Sprintf("a_%d", req.Uid), resp)
	switch err {
	case redis.Nil:
		return nil, base.New(codes.NotFound, "UserNotFoundInCache").Err()
	case nil:
		return resp, nil
	default:
		return nil, errors.Join(base.New(codes.Internal, "UnknownRedisGetErr").Err(), err)
	}
}
