package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
)

/*func main(){
	pool := newPool()
	conn := pool.Get()
	defer conn.Close()
	err := ping(conn)
	if err != nil {
		fmt.Println(err)
	}
}*/

func newPool() *redis.Pool{
	return &redis.Pool{
		// Maximum number of idle connections in the pool.
		MaxIdle: 80,
		// max number of connections
		MaxActive: 12000,
		// Dial is an application supplied function for creating and
		// configuring a connection.
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "0.0.0.0:6379")
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

func ping(c redis.Conn) error{
	s, err := redis.String(c.Do("PING"))
	if err != nil{
		return err
	}

	fmt.Printf("PING respond = %s\n", s)
	//output: PONG

	return nil
}

