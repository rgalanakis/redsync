package redsync_test

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
	"github.com/rgalanakis/redsync/rstest"
)

func expensiveOperation() {}

func ExampleNew() {
	var conn redis.Conn
	conn = redigomock.NewConn()
	pool := &redis.Pool{Dial: rstest.ConnDialer(conn)}
	pools := []*redis.Pool{pool}
	mutex := redsync.New(pools).NewMutex("example-new", redsync.NonBlocking())
	fmt.Println(mutex)
	// Output:
	// redsync.Mutex{name: example-new, tries: 1, expiry: 8s, poolcnt: 1}
}

func ExampleMutex_Lock() {
	conn := redigomock.NewConn()
	rstest.AddLockExpects(conn, "example-mutex-lock", "OK")

	pools := rstest.PoolsForConn(conn, 1)
	mutex := redsync.New(pools).NewMutex("example-mutex-lock", redsync.NonBlocking())
	err := mutex.Lock()
	if err == redsync.ErrFailed {
		fmt.Println("Failed to acquire lock.")
	} else if err != nil {
		fmt.Println("Lock acquisition had unexpected error")
	} else {
		fmt.Println("Acquired lock")
		defer mutex.Unlock()
		expensiveOperation()
	}
	// Output:
	// Acquired lock
}

func ExampleMutex_WithLock() {
	conn := redigomock.NewConn()
	rstest.AddLockExpects(conn, "example-mutex-with-lock", nil, "OK", "err")

	pools := rstest.PoolsForConn(conn, 1)
	rs := redsync.New(pools)

	result := "no calls"

	mutex1 := rs.NewMutex("example-mutex-with-lock", redsync.NonBlocking())
	called1, err1 := mutex1.WithLock(func() {
		panic("Lock will not be acquired because Redis returns nil")
	})

	mutex2 := rs.NewMutex("example-mutex-with-lock", redsync.NonBlocking())
	called2, err2 := mutex2.WithLock(func() {
		result = "mutex2 called"
	})

	mutex3 := rs.NewMutex("example-mutex-with-lock", redsync.NonBlocking())
	called3, err3 := mutex3.WithLock(func() {
		panic("Lock will not be acquired because Redis returns error")
	})

	fmt.Printf("Mutex1: called: %v,\terror: %v\n", called1, err1)
	fmt.Printf("Mutex2: called: %v,\terror: %v\n", called2, err2)
	fmt.Printf("Mutex3: called: %v,\terror: %v\n", called3, err3)
	fmt.Printf("Result: %v\n", result)
	// Output:
	// Mutex1: called: false,	error: <nil>
	// Mutex2: called: true,	error: <nil>
	// Mutex3: called: false,	error: <nil>
	// Result: mutex2 called
}
