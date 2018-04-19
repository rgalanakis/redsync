package rstest

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
)

func NewMockConn() *redigomock.Conn {
	return redigomock.NewConn()
}

func AddLockExpects(conn *redigomock.Conn, name string, expects ...interface{}) {
	cmd := conn.Command("SET", name, redigomock.NewAnyData(), "NX", "PX", redigomock.NewAnyInt())
	for _, e := range expects {
		cmd = cmd.Expect(e)
	}
}

// MockDialer returns mock as its connection.
func MockDialer(mock *redigomock.Conn) redsync.Dialer {
	return func() (redis.Conn, error) {
		return mock, nil
	}
}

func MockPools(conn *redigomock.Conn, n int) (pools []*redis.Pool) {
	for i := 0; i < n; i++ {
		pools = append(pools, &redis.Pool{Dial: MockDialer(conn)})
	}
	return pools
}