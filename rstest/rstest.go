package rstest

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
)

func NewMockConn() *redigomock.Conn {
	return redigomock.NewConn()
}

func AddLockExpects(conn *redigomock.Conn, name string, expects ...interface{}) *redigomock.Cmd {
	cmd := conn.Command("SET", name, redigomock.NewAnyData(), "NX", "PX", redigomock.NewAnyInt())
	for _, e := range expects {
		cmd = cmd.Expect(e)
	}
	return cmd
}

// ConnDialer returns fake as its connection.
func ConnDialer(fake redis.Conn) redsync.Dialer {
	return func() (redis.Conn, error) {
		return fake, nil
	}
}

func PoolsForConn(conn redis.Conn, n int) (pools []*redis.Pool) {
	for i := 0; i < n; i++ {
		pools = append(pools, &redis.Pool{Dial: ConnDialer(conn)})
	}
	return pools
}