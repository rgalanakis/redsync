package redsync_test

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rgalanakis/redsync"
	"github.com/rgalanakis/redsync/rstest"
	"strconv"
	"testing"
	"time"
)

func TestLocker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redsync Suite")
}

var _ = Describe("redsync", func() {
	tr := rstest.NewTempredis(8)

	BeforeSuite(func() {
		tr.Start()
	})

	AfterSuite(func() {
		tr.Stop()
	})

	getPoolValues := func(pools []*redis.Pool, name string) (values []string) {
		for _, pool := range pools {
			conn := pool.Get()
			value, err := redis.String(conn.Do("GET", name))
			conn.Close()
			if err != nil && err != redis.ErrNil {
				panic(err)
			}
			values = append(values, value)
		}
		return values
	}

	getPoolExpiries := func(pools []*redis.Pool, name string) (expiries []int) {
		for _, pool := range pools {
			conn := pool.Get()
			expiry, err := redis.Int(conn.Do("PTTL", name))
			conn.Close()
			if err != nil && err != redis.ErrNil {
				panic(err)
			}
			expiries = append(expiries, expiry)
		}
		return expiries
	}

	clogPools := func(pools []*redis.Pool, mask int, mutex *redsync.Mutex) int {
		n := 0
		for i, pool := range pools {
			if mask&(1<<uint(i)) == 0 {
				n++
				continue
			}
			conn := pool.Get()
			_, err := conn.Do("SET", mutex.Name(), "foobar")
			conn.Close()
			if err != nil {
				panic(err)
			}
		}
		return n
	}

	newTestMutexes := func(pools []*redis.Pool, name string, n int) (mutexes []*redsync.Mutex) {
		rs := redsync.New(pools)
		for i := 0; i < n; i++ {
			mutexes = append(mutexes, rs.NewMutex(name, redsync.Blocking()))
		}
		return mutexes
	}

	assertAcquired := func(pools []*redis.Pool, mutex *redsync.Mutex) {
		n := 0
		values := getPoolValues(pools, mutex.Name())
		for _, value := range values {
			if value == mutex.Value() {
				n++
			}
		}
		quorum := redsync.Quorum(len(pools))
		if n < quorum {
			Fail(fmt.Sprintf("Expected n >= %d, got %d", quorum, n))
		}
	}

	Describe("Redsync", func() {

		It("can create a Mutex", func() {
			pools := tr.Pools(8)
			rs := redsync.New(pools)

			mutex := rs.NewMutex("test-redsync", redsync.Blocking())
			Expect(mutex.Lock()).To(Succeed())
		})
	})

	Describe("Mutex", func() {
		It("can acquire a lock", func() {
			pools := tr.Pools(8)
			mutexes := newTestMutexes(pools, "test-mutex", 8)
			orderCh := make(chan int)
			for i, mutex := range mutexes {
				go func(i int, mutex *redsync.Mutex) {
					Expect(mutex.Lock()).To(Succeed())
					defer mutex.Unlock()
					assertAcquired(pools, mutex)

					orderCh <- i
				}(i, mutex)
			}
			for range mutexes {
				<-orderCh
			}
		})

		It("can extend a lock", func() {
			pools := tr.Pools(8)
			mutexes := newTestMutexes(pools, "test-mutex-extend", 1)
			mutex := mutexes[0]

			Expect(mutex.Lock()).To(Succeed())
			defer mutex.Unlock()

			time.Sleep(1 * time.Second)

			expiries := getPoolExpiries(pools, mutex.Name())
			Expect(mutex.Extend()).To(BeTrue())

			expiries2 := getPoolExpiries(pools, mutex.Name())

			for i, expiry := range expiries {
				if expiry >= expiries2[i] {
					Fail(fmt.Sprintf("Expected expiries[%d] > expiry, got %d %d", i, expiries2[i], expiry))
				}
			}
		})

		It("requires a quorum to get the lock", func() {
			pools := tr.Pools(4)
			for mask := 0; mask < 1<<uint(len(pools)); mask++ {
				opts := redsync.Blocking()
				opts.Tries = 1
				mutex := redsync.New(pools).NewMutex("test-mutex-partial-"+strconv.Itoa(mask), opts)

				n := clogPools(pools, mask, mutex)

				if n >= len(pools)/2+1 {
					Expect(mutex.Lock()).To(Succeed())
					assertAcquired(pools, mutex)
				} else {
					Expect(mutex.Lock()).To(Equal(redsync.ErrFailed))
				}
			}
		})

		It("errors if all servers reply with an unexpected error", func() {
			pools := rstest.PoolsForConn(rstest.NewMockConn(), 4)
			mutex := redsync.New(pools).NewMutex("test-errors", redsync.NonBlocking())
			Expect(mutex.Lock()).To(Not(Succeed()))
			Expect(mutex.Lock().Error()).To(ContainSubstring("not registered in redigomock library"))
		})

		It("can use rstest to set up lock mocks", func() {
			name := "test-lockmock"
			conn := rstest.NewMockConn()
			rstest.AddLockExpects(conn, name, "OK", nil)

			pools := rstest.PoolsForConn(conn, 1)
			mutex1 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			Expect(mutex1.Lock()).To(Succeed())

			mutex2 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			Expect(mutex2.Lock()).To(Equal(redsync.ErrFailed))
		})

		It("will conditionally execute a function on lock acquisition", func() {
			name := "test-withlock"
			conn := rstest.NewMockConn()
			rstest.AddLockExpects(conn, name, nil, "OK", nil).ExpectError(errors.New("failed"))
			pools := rstest.PoolsForConn(conn, 1)

			mutex1 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			mutex2 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			mutex3 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			mutex4 := redsync.New(pools).NewMutex(name, redsync.NonBlocking())
			var res1, res2, res3, res4 bool

			locked1, err1 := mutex1.WithLock(func() {
				res1 = true
			})
			locked2, err2 := mutex2.WithLock(func() {
				res2 = true
			})
			locked3, err3 := mutex3.WithLock(func() {
				res3 = true
			})
			locked4, err4 := mutex4.WithLock(func() {
				res4 = true
			})
			Expect(err1).To(Not(HaveOccurred()))
			Expect(locked1).To(BeFalse())
			Expect(res1).To(BeFalse())

			Expect(err2).To(Not(HaveOccurred()))
			Expect(locked2).To(BeTrue())
			Expect(res2).To(BeTrue())

			Expect(err3).To(Not(HaveOccurred()))
			Expect(locked3).To(BeFalse())
			Expect(res3).To(BeFalse())

			Expect(err4).To(HaveOccurred())
			Expect(locked4).To(BeFalse())
			Expect(res4).To(BeFalse())
		})
	})

})
