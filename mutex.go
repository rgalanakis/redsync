package redsync

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"fmt"
	"github.com/gomodule/redigo/redis"
	"sync"
)

// Mutex is a distributed mutual exclusion lock.
// Note that a redsync.Mutex is not goroutine-safe.
// Each goroutine should create its own Mutex instance for locking.
// Note that Redsync instances are threadsafe, so they can be reused across goroutines.
type Mutex struct {
	name   string
	expiry time.Duration

	tries int
	delay time.Duration

	factor float64

	quorum int

	memMutex *sync.Mutex

	value string
	until time.Time

	pools []*redis.Pool
}

// String returns a string representation of the mutex.
func (m *Mutex) String() string {
	return fmt.Sprintf("redsync.Mutex{name: %s, tries: %d, expiry: %s, poolcnt: %d}",
		m.name, m.tries, m.expiry.String(), len(m.pools))
}

// Name returns the mutex name.
func (m *Mutex) Name() string {
	return m.name
}

// Value returns the mutex value.
func (m *Mutex) Value() string {
	return m.value
}

// Lock acquires a lock on the mutex with the receiver's Name.
// If Lock returns nil, the lock is acquired. Callers should make sure Unlock is called,
// usually via defer m.Unlock().
// If Lock returns ErrFailed, the lock could not be acquired because it was held by another mutex.
// Callers may wish to call Lock() again to retry.
// If Lock returns any other error, the lock may not be acquire-able do to an unexpected error,
// like if redis is not running.
func (m *Mutex) Lock() error {
	value, err := m.genValue()
	if err != nil {
		return err
	}

	for i := 0; i < m.tries; i++ {
		if i != 0 {
			time.Sleep(m.delay)
		}

		start := time.Now()

		acquired, err := m.acquireAll(value)
		if err != nil {
			m.releaseAll(value)
			return err
		}

		until := time.Now().Add(m.expiry - time.Now().Sub(start) - time.Duration(int64(float64(m.expiry)*m.factor)) + 2*time.Millisecond)
		if acquired >= m.quorum && time.Now().Before(until) {
			m.value = value
			m.until = until
			return nil
		}
		m.releaseAll(value)
	}

	return ErrFailed
}

// Unlock unlocks m and returns the status of unlock.
func (m *Mutex) Unlock() bool {
	released := m.releaseAll(m.value)
	return released >= m.quorum
}

// WithLock invokes f if the lock was successfully invoked. See Lock for more info.
// The boolean return value is true if the lock was acquired and f was invoked,
// false if not.
// The error is only non-nil if an unexpected error occurred.
// In other words, if Lock() returns ErrFailed, WithLock returns an error of nil.
func (m *Mutex) WithLock(f func()) (bool, error) {
	err := m.Lock()
	if err == ErrFailed {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer m.Unlock()
	f()
	return true, nil
}

func (m *Mutex) genValue() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (m *Mutex) acquireAll(value string) (int, error) {
	n := 0
	for _, pool := range m.pools {
		ok, err := m.acquire(pool, value)
		if ok {
			n++
		}
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (m *Mutex) acquire(pool *redis.Pool, value string) (bool, error) {
	if m.memMutex != nil {
		m.memMutex.Lock()
		defer m.memMutex.Unlock()
	}
	conn := pool.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("SET", m.name, value, "NX", "PX", int(m.expiry/time.Millisecond)))
	//fmt.Println("acquire", reply, "err", err)
	if reply == "OK" {
		return true, nil
	}
	if err == redis.ErrNil {
		return false, nil
	}
	return false, err
}

func (m *Mutex) releaseAll(value string) int {
	n := 0
	for _, pool := range m.pools {
		if m.release(pool, value) {
			n++
		}
	}
	return n
}

var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (m *Mutex) release(pool *redis.Pool, value string) bool {
	if m.memMutex != nil {
		m.memMutex.Lock()
		defer m.memMutex.Unlock()
	}
	conn := pool.Get()
	defer conn.Close()
	status, err := deleteScript.Do(conn, m.name, value)
	return err == nil && status != 0
}
