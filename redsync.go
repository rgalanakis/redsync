package redsync

import (
	"github.com/gomodule/redigo/redis"
	"time"
)

// Redsync provides a simple method for creating distributed mutexes using multiple Redis connection pools.
type Redsync struct {
	pools []*redis.Pool
}

// New creates and returns a new Redsync instance from given Redis connection pools.
func New(pools []*redis.Pool) *Redsync {
	return &Redsync{
		pools: pools,
	}
}

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

func Blocking() MutexOpts {
	return MutexOpts{
		Expiry: 8 * time.Second,
		Tries:  32,
		Delay:  500 * time.Millisecond,
		Factor: 0.01,
	}
}

func NonBlocking() MutexOpts {
	return MutexOpts{
		Expiry: 8 * time.Second,
		Tries:  1,
		Delay:  10 * time.Millisecond,
		Factor: 0.01,
	}
}

// NewMutex returns a new distributed mutex with given Name.
func (r *Redsync) NewMutex(name string, opts MutexOpts) *Mutex {
	return &Mutex{
		name:   name,
		expiry: opts.Expiry,
		tries:  opts.Tries,
		delay:  opts.Delay,
		factor: opts.Factor,
		quorum: len(r.pools)/2 + 1,
		pools:  r.pools,
	}
}
