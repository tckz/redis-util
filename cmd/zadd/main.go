package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	redisutil "github.com/tckz/redis-util"
)

var version string

func main() {

	showVersion := flag.Bool("version", false, "Show version")
	worker := flag.Uint("worker", 32, "Number of worker goroutines")
	inSplit := flag.Uint("in-split", 8, "Number of goroutines for reading file")
	randomKeys := flag.Uint("random", 0, "Number of pairs that consist of member and score to generate")
	randomPrefix := flag.String("random-prefix", "rand-", "Prefix of random generated member")
	key := flag.String("key", "", "Key of ZSET")
	var nodes redisutil.StrSlice
	flag.Var(&nodes, "node", "Redis server host and port(ex. 127.0.0.1:6379)")
	flag.Parse()
	files := flag.Args()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version)
		return
	}

	if *randomKeys > 0 {
		if *key == "" {
			log.Fatalf("*** key must be specified")
		}
	} else {
		if len(files) == 0 {
			log.Fatalf("*** Files to load must be specified")
		}
	}

	if len(nodes) == 0 {
		nodes = []string{"127.0.0.1:6379"}
	}

	if *inSplit <= 0 {
		log.Fatalf("*** --in-split must be >= 1")
	}

	if *worker <= 0 {
		log.Fatalf("*** --worker must be >= 1")
	}

	chLine := make(chan string, *worker)
	from := time.Now()

	var lineCount int64
	if *randomKeys > 0 {
		go func() {
			for i := uint(0); i < *randomKeys; i++ {
				member := uuid.Must(uuid.NewRandom()).String()
				// 整数部分でなんとなく大小がわかるように
				score := rand.Float64() * 1000000
				chLine <- fmt.Sprintf("%s\t%f\t%s%s", *key, score, *randomPrefix, member)
			}
			close(chLine)
		}()
	} else {
		chFile := make(chan uint64)
		for i, file := range files {
			// ファイルを分割並列入力して、入力行をチャンネルに投げる
			sr := redisutil.SplitReader{MinBlockSize: 1024 * 4}
			index := i
			fn := file
			go func() {
				lc := sr.LoadFile(uint(index), *inSplit, fn, chLine, 100000)
				chFile <- lc
			}()
		}

		// ファイル入力が全部終わったら、入力行chを閉じる
		go func() {
			for i := 0; i < len(files); i++ {
				lc := <-chFile
				lineCount = lineCount + int64(lc)
			}
			close(chLine)
		}()
	}

	chResult := make(chan redisutil.Result, *worker)
	for i := uint(0); i < *worker; i++ {
		index := i
		go func() {
			chResult <- zadd(index, nodes, chLine)
		}()
	}

	// 全ての受信goルーチンが終わったら終了
	totalResult := redisutil.NewResult()
	for i := uint(0); i < *worker; i++ {
		result := <-chResult
		totalResult = totalResult.Combine(result)
	}

	fmt.Fprintf(os.Stderr, "Lines: %d, Got: %d, Bad: %d, Elapsed: %s, Errors: %v\n",
		lineCount, totalResult.Lines, totalResult.BadCount, time.Since(from), totalResult.Errors)
}

func zadd(i uint, nodes []string, chLine <-chan string) redisutil.Result {
	client := redisutil.NewRedisClient(nodes)
	defer client.Close()

	var lc uint64
	from := time.Now()

	result := redisutil.NewResult()

	members := make([]redis.Z, 0, 1024)
	for line := range chLine {
		lc++
		if lc%100000 == 0 {
			fmt.Fprintf(os.Stderr, "[%02d]zadd: %d\n", i, lc)
		}

		// {key}    {score}	{member}...
		tokens := strings.SplitN(line, "\t", -1)
		tokenCount := len(tokens)
		if tokenCount < 3 || (tokenCount&1) == 0 {
			result.AddError(fmt.Sprintf("Number of tokens = %d", tokenCount))
			continue
		}

		if cap(members) < tokenCount-1 {
			members = make([]redis.Z, 0, tokenCount-1)
		}

		members = members[:0]
		for i := 1; i < tokenCount; i += 2 {
			score, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				result.AddError(err.Error())
				continue
			}
			members = append(members, redis.Z{
				Score:  score,
				Member: tokens[i+1],
			})
		}
		_, err := client.ZAdd(tokens[0], members...).Result()
		if err != nil {
			result.AddError(err.Error())
			continue
		}
	}

	fmt.Fprintf(os.Stderr, "[%02d]zadd: %d, Elapsed: %s\n", i, lc, time.Since(from))

	result.Lines = lc

	return result
}
