package base

import (
	"errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"testing"
)

func Test_statusError_Error(t *testing.T) {
	err := errors.Join(errors.New("hello"), New(codes.Internal, "").Err())
	require.Equal(t, "hello\ncode=Internal", err.Error())
}
