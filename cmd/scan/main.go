package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	redisutil "github.com/tckz/redis-util"
)

var (
	version string
)

func main() {
	optCount := flag.Int64("count", 1000, "Scan count at once")
	optVersion := flag.Bool("version", false, "Show version")
	optCursor := flag.Uint64("cursor", 0, "Beginning of cursor")
	optMatch := flag.String("match", "", "match")
	var nodes redisutil.StrSlice
	flag.Var(&nodes, "node", "Redis server host and port(ex. 127.0.0.1:6379)")
	flag.Parse()

	if *optVersion {
		fmt.Printf("%s\n", version)
		return
	}

	if len(nodes) == 0 {
		nodes = []string{"127.0.0.1:6379"}
	}

	cl := redisutil.NewRedisClient(nodes)
	defer cl.Close()

	from := time.Now()
	ctx := context.Background()
	nextCursor := *optCursor
	var count uint64
	for {
		keys, cursor, err := cl.Scan(ctx, nextCursor, *optMatch, *optCount).Result()
		if err != nil {
			log.Printf("*** Scan: %v", err)
			return
		}
		for _, k := range keys {
			count++
			if count%10000 == 0 {
				log.Printf("count=%d, nextCursor=%d\n", count, nextCursor)
			}
			fmt.Printf("%s\n", k)
		}
		if cursor == 0 {
			break
		}
		nextCursor = cursor
	}
	log.Printf("Elapsed: %s, total=%d, lastCursor=%d\n", time.Since(from), count, nextCursor)
}
