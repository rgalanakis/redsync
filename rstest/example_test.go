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
	mutex := redsync.New(pools).NewMutex("example-lock-expects", redsync.NonBlocking())

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

func ExampleServers() {
	tr := make(rstest.Servers, 2)
	fmt.Println("Created", len(tr), "temp redis servers")
	fmt.Println("Servers are nil?", tr[0] == nil)
	tr.Start()
	fmt.Println("Started servers")
	fmt.Println("Servers are nil?", tr[0] == nil)
	tr.Stop()
	fmt.Println("Servers terminated")
	// Output:
	// Created 2 temp redis servers
	// Servers are nil? true
	// Started servers
	// Servers are nil? false
	// Servers terminated
}