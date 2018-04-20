package redsync

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"time"
)

// Dialer functions return an item with the redis.Conn interface, or an error.
// It fulfills the interface for the Dial argument to a redis.Pool.
type Dialer func() (redis.Conn, error)

// TcpDialer connects to an address string, like "localhost:6379".
func TcpDialer(addr string) Dialer {
	return func() (redis.Conn, error) {
		return redis.Dial("tcp", addr)
	}
}

// UnixDialer connects to an address string, like "/var/folders/6j/xyz/T/abc/redis.sock".
func UnixDialer(addr string) Dialer {
	return func() (redis.Conn, error) {
		fmt.Printf(addr)
		return redis.Dial("unix", addr)
	}
}

// Redsync is a factory for redsync.Mutex.
// It wraps a number of redis.Pool instances, each of which can have multiple connections.
// Use NewMutex to create a mutex.
type Redsync struct {
	pools []*redis.Pool
}

// New creates and returns a new Redsync instance from given Redis connection pools.
func New(pools ...*redis.Pool) *Redsync {
	return &Redsync{
		pools: pools,
	}
}

// MutexOpts are the options for mutex construction.
// In general, calls should use redsync.Blocking() or redsync.NonBlocking()
// and customize the result, but they can also create a MutexOpts themselves.
type MutexOpts struct {
	// Expiry is the amount of time before the lock expires.
	// Useful to make sure the lock is expired even if the lock is never released,
	// like if a process dies while the lock is held.
	Expiry time.Duration
	// Tries is the number of times a lock acquisition is attempted.
	Tries int
	// Delay is the amount of time to wait between retries.
	Delay time.Duration
	// Factor is the clock drift Factor.
	Factor float64
}

// Blocking returns the default MutexOpts for a blocking mutex.
// A blocking mutex will not return from Lock until the lock is acquired,
// or Delay has elapsed (500ms by default).
func Blocking() MutexOpts {
	return MutexOpts{
		Expiry: 8 * time.Second,
		Tries:  32,
		Delay:  500 * time.Millisecond,
		Factor: 0.01,
	}
}

// NonBlocking returns the default MutexOpts for a non-blocking mutex.
// A non-blocking mutex gives up the first time if it cannot acquire a mutex,
// rather than retrying and spinning.
func NonBlocking() MutexOpts {
	return MutexOpts{
		Expiry: 8 * time.Second,
		Tries:  1,
		Delay:  10 * time.Millisecond,
		Factor: 0.01,
	}
}

// NewMutex returns a new distributed mutex with given name and options.
func (r *Redsync) NewMutex(name string, opts MutexOpts) *Mutex {
	return &Mutex{
		name:   name,
		expiry: opts.Expiry,
		tries:  opts.Tries,
		delay:  opts.Delay,
		factor: opts.Factor,
		quorum: Quorum(len(r.pools)),
		pools:  r.pools,
	}
}

// Quorum returns the number of servers that must agree in order to acquire a lock (n/2 + 1).
func Quorum(n int) int {
	return n/2 + 1
}
