package redsync

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	getPoolValues := func(pools []*redis.Pool, name string) []string {
		var values []string
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

	getPoolExpiries := func(pools []*redis.Pool, name string) []int {
		var expiries []int
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

	clogPools := func(pools []*redis.Pool, mask int, mutex *Mutex) int {
		n := 0
		for i, pool := range pools {
			if mask&(1<<uint(i)) == 0 {
				n++
				continue
			}
			conn := pool.Get()
			_, err := conn.Do("SET", mutex.name, "foobar")
			conn.Close()
			if err != nil {
				panic(err)
			}
		}
		return n
	}

	newTestMutexes := func(pools []*redis.Pool, name string, n int) []*Mutex {
		var mutexes []*Mutex
		rs := New(pools)
		for i := 0; i < n; i++ {
			mutexes = append(mutexes, rs.NewMutex(name, Blocking()))
		}
		return mutexes
	}

	assertAcquired := func(pools []*redis.Pool, mutex *Mutex) {
		n := 0
		values := getPoolValues(pools, mutex.name)
		for _, value := range values {
			if value == mutex.value {
				n++
			}
		}
		if n < mutex.quorum {
			Fail(fmt.Sprintf("Expected n >= %d, got %d", mutex.quorum, n))
		}
	}

	Describe("Redsync", func() {

		It("uses its pool to acquire a lock", func() {
			pools := tr.Pools(8)
			rs := New(pools)

			mutex := rs.NewMutex("test-redsync", Blocking())
			err := mutex.Lock()
			if err != nil {

			}
		})
	})

	Describe("Mutex", func() {
		It("can acquire a lock", func() {
			pools := tr.Pools(8)
			mutexes := newTestMutexes(pools, "test-mutex", 8)
			orderCh := make(chan int)
			for i, mutex := range mutexes {
				go func(i int, mutex *Mutex) {
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

			expiries := getPoolExpiries(pools, mutex.name)
			Expect(mutex.Extend()).To(BeTrue())

			expiries2 := getPoolExpiries(pools, mutex.name)

			for i, expiry := range expiries {
				if expiry >= expiries2[i] {
					Fail(fmt.Sprintf("Expected expiries[%d] > expiry, got %d %d", i, expiries2[i], expiry))
				}
			}
		})

		It("requires a quorum to get the lock", func() {
			pools := tr.Pools(4)
			for mask := 0; mask < 1<<uint(len(pools)); mask++ {
				mutexes := newTestMutexes(pools, "test-mutex-partial-"+strconv.Itoa(mask), 1)
				mutex := mutexes[0]
				mutex.tries = 1

				n := clogPools(pools, mask, mutex)

				if n >= len(pools)/2+1 {
					Expect(mutex.Lock()).To(Succeed())
					assertAcquired(pools, mutex)
				} else {
					Expect(mutex.Lock()).To(Equal(ErrFailed))
				}
			}
		})
	})

})
