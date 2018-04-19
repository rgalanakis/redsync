// Package redsync provides a Redis-based distributed mutual exclusion lock implementation as described
// in the post http://redis.io/topics/distlock.
//
// See examples for suggestions on how to use the lock.
//
// Testing with locks
//
// This package uses a combination of testing against real redis servers using tempredis,
// and in-memory mocking using redigomock.
// Clients of redsync are expected to test against redigomock, rather than having to run real redis.
// There are helpers available for testing with locks in the redsync/rstest package.
// The Mutex examples include usages of rstest, in particular rstest.AddLockExpects.
// Please refer to them for examples of how to use mocks when testing redsync locks.
package redsync
