package rstest_test

import (
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync/rstest"
	"github.com/rgalanakis/redsync"
	"errors"
	"fmt"
)

func ExampleAddLockExpects() {
	conn := redigomock.NewConn()
	rstest.AddLockExpects(conn, "example-lock-expects", "OK", "OK", nil, "e").
		ExpectError(errors.New("uh-oh"))

	pools := rstest.PoolsForConn(conn, 1)
	mutex := redsync.New(pools...).NewMutex("example-lock-expects", redsync.NonBlocking())

	fmt.Println(mutex.Lock())
	fmt.Println(mutex.Lock())
	fmt.Println(mutex.Lock())
	fmt.Println(mutex.Lock())
	fmt.Println(mutex.Lock())
	// Output:
	// <nil>
	// <nil>
	// redsync: failed to acquire lock
	// redsync: failed to acquire lock
	// uh-oh

}
