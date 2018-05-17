package rstest

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
	"sync"
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

// ThreadsafeConn can be used to wrap a redis.Conn that is not threadsafe.
// This is usually a redigomock.Conn.
type ThreadsafeConn struct {
	Conn redis.Conn
	lock sync.Mutex
}

func (t *ThreadsafeConn) Close() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Close()
}

func (t *ThreadsafeConn) Err() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Err()
}

func (t *ThreadsafeConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Do(commandName, args...)
}

func (t *ThreadsafeConn) Send(commandName string, args ...interface{}) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Send(commandName, args...)
}

func (t *ThreadsafeConn) Flush() error {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Flush()
}

func (t *ThreadsafeConn) Receive() (reply interface{}, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.Conn.Receive()
}
