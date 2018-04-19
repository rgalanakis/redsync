package rstest

import (
	"github.com/stvp/tempredis"
	"time"
	"github.com/gomodule/redigo/redis"
)

type Tempredis struct {
	servers []*tempredis.Server
}

func NewTempredis(cnt int) *Tempredis {
	return &Tempredis{servers: make([]*tempredis.Server, cnt)}
}

func (ts *Tempredis) Start() {
	for i := 0; i < len(ts.servers); i++ {
		server, err := tempredis.Start(tempredis.Config{})
		if err != nil {
			panic(err)
		}
		ts.servers[i] = server
	}
}

func (ts *Tempredis) Pools(n int) []*redis.Pool {
	var pools []*redis.Pool
	for _, server := range ts.servers {
		func(server *tempredis.Server) {
			pools = append(pools, &redis.Pool{
				MaxIdle:     3,
				IdleTimeout: 240 * time.Second,
				Dial: func() (redis.Conn, error) {
					return redis.Dial("unix", server.Socket())
				},
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

func (ts *Tempredis) Stop() {
	for _, server := range ts.servers {
		server.Term()
	}
}