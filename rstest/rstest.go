package rstest

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
)

// AddLockExpects is a helper for adding redigomock.Conn expectations for locking.
// name is the string name of the mutex or a redigomock.FuzzyMatcher,
// and expects are each arguments passed to redigomock.Cmd#Expect.
// For example, if you wanted to acquire the "my-mutex" four times,
// and succeed the first two times, then fail due to contention,
// you could use AddLockExpects(mockConn, "my-mutex", "OK", "OK", nil).
func AddLockExpects(conn *redigomock.Conn, name interface{}, expects ...interface{}) *redigomock.Cmd {
	cmd := conn.Command("SET", name, redigomock.NewAnyData(), "NX", "PX", redigomock.NewAnyInt())
	for _, e := range expects {
		cmd = cmd.Expect(e)
	}
	return cmd
}

// ConnDialer returns fake as its connection.
// This is generally useful for fake connections- use TcpDialer or UnixDialer for real connections.
func ConnDialer(fake redis.Conn) redsync.Dialer {
	return func() (redis.Conn, error) {
		return fake, nil
	}
}

// PoolsForConn returns a slice of n redis.Pool instances,
// all of which return the same connection.
// See package specs for usage.
func PoolsForConn(conn redis.Conn, n int) (pools []*redis.Pool) {
	for i := 0; i < n; i++ {
		pools = append(pools, &redis.Pool{Dial: ConnDialer(conn)})
	}
	return pools
}
