package rstest

import (
	"github.com/stvp/tempredis"
	"time"
	"github.com/gomodule/redigo/redis"
	"github.com/rgalanakis/redsync"
)

// Servers is a slice of tempredis reservers.
// Create a Servers slice of the length equal to the number of servers,
// then use Start() to fill it with servers. Stop() stops the servers.
// Pool(n) returns a slice of redis.Pool instances, one for each server,
// up to n.
type Servers []*tempredis.Server

// Start starts the tempredis servers and fills in the empty slice.
func (ts Servers) Start() {
	for i := 0; i < len(ts); i++ {
		server, err := tempredis.Start(tempredis.Config{})
		if err != nil {
			panic(err)
		}
		ts[i] = server
	}
}

// Pools returns a slice of redis.Pool instances, one for each server, up to n.
func (ts Servers) Pools(n int) []*redis.Pool {
	var pools []*redis.Pool
	for _, server := range ts {
		func(server *tempredis.Server) {
			pools = append(pools, &redis.Pool{
				MaxIdle:     3,
				IdleTimeout: 240 * time.Second,
				Dial: redsync.UnixDialer(server.Socket()),
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
func (ts Servers) Stop() {
	for _, server := range ts {
		server.Term()
	}
}