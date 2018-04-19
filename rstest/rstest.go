package rstest

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

type MockConn = redigomock.Conn

// MockDialer returns mock as its connection.
func MockDialer(mock *MockConn) func() (redis.Conn, error) {
	return func() (redis.Conn, error) {
		return mock, nil
	}
}

func MockPools(conn *MockConn, n int) (pools []*redis.Pool) {
	for i := 0; i < n; i++ {
		pools = append(pools, &redis.Pool{Dial: MockDialer(conn)})
	}
	return pools
}