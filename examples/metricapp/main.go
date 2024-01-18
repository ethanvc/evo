package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethanvc/evo/base"
	"github.com/ethanvc/evo/evohttp"
	"github.com/ethanvc/evo/examples/metricapp/rediskey"
	"github.com/ethanvc/evo/plog"
	"github.com/ethanvc/evo/rediscli"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"net"
	"net/http"
	"time"
)

func main() {
	fx.New(
		fx.Provide(NewGormDb),
		fx.Provide(newRedisClient),
		fx.Provide(newUserController),
		fx.Provide(newHttpServer),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

func newHttpServer(lc fx.Lifecycle, user *userController) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	evoSvr := evohttp.NewServer()
	userGroup := evoSvr.SubBuilder("/api/users")
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

func (u *UserDto) TableName() string {
	return "user_tab"
}

type userController struct {
	redisCli *rediscli.RedisClient
	db       *gorm.DB
}

func newUserController(
	redisCli *rediscli.RedisClient,
	db *gorm.DB,
) *userController {
	return &userController{
		redisCli: redisCli,
		db:       db,
	}
}

func (controller *userController) QueryUser(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	// here no need to record the error, because import event will be recorded in place.
	resp, err = controller.queryUserFromCache(c, req)
	if err == nil {
		return resp, nil
	}
	resp, err = controller.queryUserFromDb(c, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (controller *userController) queryUserFromCache(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	// for request resolved network, have to record time cost and response content.
	c = plog.WithLogContext(c, nil)
	defer func() { plog.RequestLog(c, err, req, resp) }()
	c, cancel := context.WithTimeoutCause(c, time.Millisecond*100,
		base.New(codes.DeadlineExceeded, "GetFromRedisTimeout").Err())
	defer cancel()
	err = controller.redisCli.Get(c, rediskey.UserCacheKey(req.Uid), resp)
	switch err {
	case redis.Nil:
		return nil, base.New(codes.NotFound, "UserNotFoundInCache").Err()
	case nil:
		return resp, nil
	default:
		return nil, errors.Join(base.New(codes.Internal, "UnknownRedisGetErr").Err(), err)
	}
}

func (controller *userController) queryUserFromDb(c context.Context, req *QueryUserReq) (resp *UserDto, err error) {
	c = plog.WithLogContext(c, nil)
	defer func() { plog.RequestLog(c, err, req, resp) }()
	if req.Uid == 0 {
		return nil, base.New(codes.InvalidArgument, "UidZero").Err()
	}
	filter := &UserDto{
		Uid: req.Uid,
	}
	err = controller.db.Where(filter).First(&resp).Error
	switch err {
	case gorm.ErrRecordNotFound:
		return nil, base.New(codes.NotFound, "UserNotFoundInDb").Err()
	case nil:
		return
	default:
		return nil, errors.Join(base.New(codes.Internal, "UserDbError").Err(), err)
	}
}

func (controller *userController) saveUserToCache(c context.Context, user *UserDto) (err error) {
	c = plog.WithLogContext(c, nil)
	defer func() { plog.RequestLog(c, err, user, nil) }()
	err = controller.redisCli.Set(c, rediskey.UserCacheKey(user.Uid), user, 0)
	switch err {
	case nil:
		return
	default:
		return errors.Join(base.New(codes.Unknown, "SaveUserToRedisErr").Err(), err)
	}
}

func NewGormDb() (*gorm.DB, error) {
	dsn := "root:@tcp(127.0.0.1:3306)/test_db?charset=utf8mb4"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
