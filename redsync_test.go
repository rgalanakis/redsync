package redsync_test

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rafaeljusto/redigomock"
	"github.com/rgalanakis/redsync"
	"github.com/rgalanakis/redsync/rstest"
	"github.com/stvp/tempredis"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestLocker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redsync Suite")
}

var _ = Describe("redsync", func() {
	tr := make(TempServers, 8)

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
		rs := redsync.New(pools...)
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
			rs := redsync.New(pools...)

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

		It("requires a quorum to get the lock", func() {
			pools := tr.Pools(4)
			for mask := 0; mask < 1<<uint(len(pools)); mask++ {
				opts := redsync.Blocking()
				opts.Tries = 1
				mutex := redsync.New(pools...).NewMutex("test-mutex-partial-"+strconv.Itoa(mask), opts)

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
			pools := rstest.PoolsForConn(redigomock.NewConn(), 4)
			mutex := redsync.New(pools...).NewMutex("test-errors", redsync.NonBlocking())
			Expect(mutex.Lock()).To(Not(Succeed()))
			Expect(mutex.Lock().Error()).To(ContainSubstring("not registered in redigomock library"))
		})

		It("can use rstest to set up lock mocks", func() {
			name := "test-lockmock"
			conn := redigomock.NewConn()
			rstest.AddLockExpects(conn, name, "OK", nil)

			pools := rstest.PoolsForConn(conn, 1)
			mutex1 := redsync.New(pools...).NewMutex(name, redsync.NonBlocking())
			Expect(mutex1.Lock()).To(Succeed())

			mutex2 := redsync.New(pools...).NewMutex(name, redsync.NonBlocking())
			Expect(mutex2.Lock()).To(Equal(redsync.ErrFailed))
		})

		It("will conditionally execute a function on lock acquisition", func() {
			name := "test-withlock"
			conn := redigomock.NewConn()
			rstest.AddLockExpects(conn, name, nil, "OK", nil).ExpectError(errors.New("failed"))
			pools := rstest.PoolsForConn(conn, 1)

			rs := redsync.New(pools...)
			mutex1 := rs.NewMutex(name, redsync.NonBlocking())
			mutex2 := rs.NewMutex(name, redsync.NonBlocking())
			mutex3 := rs.NewMutex(name, redsync.NonBlocking())
			mutex4 := rs.NewMutex(name, redsync.NonBlocking())
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

		It("supports a lock to avoid race conditions in the connection library", func() {
			var wg sync.WaitGroup
			conn := redigomock.NewConn()
			name := "test-racelock"
			pools := rstest.PoolsForConn(conn, 1)

			rs := redsync.New(pools...)
			opts := redsync.NonBlocking()
			opts.MemMutex = &sync.Mutex{}
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rmux := rs.NewMutex(name, opts)
					rmux.WithLock(func() {})
				}()
			}
			wg.Wait()
		})
	})

	Describe("TCPDialier", func() {
		It("connects to a host", func() {
			_, err := redsync.TcpDialer("127.0.0.1:6379")()
			Expect(err).To(MatchError(ContainSubstring("connection refused")))
		})
	})

})

type TempServers []*tempredis.Server

// Start starts the tempredis servers and fills in the empty slice.
func (ts TempServers) Start() {
	for i := 0; i < len(ts); i++ {
		server, err := tempredis.Start(tempredis.Config{})
		if err != nil {
			panic(err)
		}
		ts[i] = server
	}
}

// Pools returns a slice of redis.Pool instances, one for each server, up to n.
func (ts TempServers) Pools(n int) []*redis.Pool {
	var pools []*redis.Pool
	for _, server := range ts {
		func(server *tempredis.Server) {
			pools = append(pools, &redis.Pool{
				MaxIdle:     3,
				IdleTimeout: 240 * time.Second,
				Dial:        redsync.UnixDialer(server.Socket()),
				TestOnBorrow: func(c redis.Conn, t time.Time) error {
					_, err := c.Do("PING")
					return err
				},
			})
		}(server)
		if len(pools) == n {
			break
		}
	}
	return pools
}

// Stop stops the testredis servers.
func (ts TempServers) Stop() {
	for _, server := range ts {
		server.Term()
	}
}

// TempServers is a slice of tempredis reservers.
// Create a TempServers slice of the length equal to the number of servers,
// then use Start() to fill it with servers. Stop() stops the servers.
// Pool(n) returns a slice of redis.Pool instances, one for each server,
// up to n.
//func ExampleServers() {
//	tr := make(TempServers, 2)
//	fmt.Println("Created", len(tr), "temp redis servers")
//	fmt.Println("TempServers are nil?", tr[0] == nil)
//	tr.Start()
//	fmt.Println("Started servers")
//	fmt.Println("TempServers are nil?", tr[0] == nil)
//
//	pools := tr.Pools(2)
//	fmt.Println("Created a slice of", len(pools), "[]*redis.Pool, each to a different server")
//	tr.Stop()
//	fmt.Println("TempServers terminated")
//	// Output:
//	// Created 2 temp redis servers
//	// TempServers are nil? true
//	// Started servers
//	// TempServers are nil? false
//	// Created a slice of 2 []*redis.Pool, each to a different server
//	// TempServers terminated
//}
