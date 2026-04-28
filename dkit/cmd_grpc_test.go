package dkit

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func TestQuerySvrListSendsHeadersToReflection(t *testing.T) {
	addr, cleanup := startReflectionServerRequiringHeader(t, "authorization", "Bearer token")
	defer cleanup()

	err := GrpcMain(&GrpcMainReq{
		Host:    addr,
		Query:   "list-svr",
		Headers: []string{"Authorization: Bearer token"},
	})
	require.NoError(t, err)
}

func TestQuerySvrListFailsWithoutRequiredReflectionHeader(t *testing.T) {
	addr, cleanup := startReflectionServerRequiringHeader(t, "authorization", "Bearer token")
	defer cleanup()

	err := GrpcMain(&GrpcMainReq{Host: addr, Query: "list-svr"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing authorization")
}

func startReflectionServerRequiringHeader(t *testing.T, key, value string) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer(grpc.StreamInterceptor(requireStreamHeader(key, value)))
	reflection.Register(server)

	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		server.Stop()
		_ = lis.Close()
	}
}

func requireStreamHeader(key, value string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(stream.Context())
		if !ok || !contains(md.Get(key), value) {
			return status.Errorf(codes.Unauthenticated, "missing %s", key)
		}
		return handler(srv, stream)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
