package main

/*
 * 入力ファイルを並列分割処理するためオフセットの計算が必要になる都合、
 * 改行コードは「必ずLF」であることを前提としている。
 */

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	redisutil "github.com/tckz/redis-util"
)

var version string

func main() {

	showVersion := flag.Bool("version", false, "Show version")
	worker := flag.Uint("worker", 32, "Number of receiving goroutines")
	inSplit := flag.Uint("in-split", 8, "Number of goroutines for reading file")
	var nodes redisutil.StrSlice
	flag.Var(&nodes, "node", "Redis server host and port(ex. 127.0.0.1:6379)")
	flag.Parse()
	files := flag.Args()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version)
		return
	}

	if len(files) == 0 {
		log.Fatalf("*** Files to load must be specified")
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
	chFile := make(chan uint)
	from := time.Now()

	for i, file := range files {
		// ファイルを分割並列入力して、入力行をチャンネルに投げる
		sr := redisutil.SplitReader{MinBlockSize: 1024 * 4}
		go sr.LoadFile(uint(i), *inSplit, file, chFile, chLine, 100000)
	}

	var lineCount int64
	// ファイル入力が全部終わったら、入力行chを閉じる
	go func() {
		for i := 0; i < len(files); i++ {
			lc := <-chFile
			lineCount = lineCount + int64(lc)
		}
		close(chLine)
	}()

	chResult := make(chan redisutil.Result, *worker)
	for i := uint(0); i < *worker; i++ {
		go hmset(i, nodes, chResult, chLine)
	}

	// 全ての受信goルーチンが終わったら終了
	totalResult := redisutil.NewResult()
	for i := uint(0); i < *worker; i++ {
		result := <-chResult
		totalResult = totalResult.Combine(result)
	}

	elapsed := time.Since(from)
	fmt.Fprintf(os.Stderr, "Lines: %d, Got: %d, Bad: %d, Elapsed: %s, Errors: %v\n",
		lineCount, totalResult.Lines, totalResult.BadCount, elapsed, totalResult.Errors)
}

func hmset(i uint, nodes []string, chResult chan<- redisutil.Result, chLine <-chan string) {
	client := redisutil.NewRedisClient(nodes)
	defer client.Close()

	var lc uint64
	from := time.Now()

	result := redisutil.NewResult()

	for line := range chLine {
		lc++
		if lc%100000 == 0 {
			fmt.Fprintf(os.Stderr, "[%02d]hmset: %d\n", i, lc)
		}

		// {key}    {json}    {unixtime msec}
		token := strings.SplitN(line, "\t", 3)
		if len(token) != 3 {
			result.AddError("Number of tokens != 3")
			continue
		}

		var m map[string]interface{}
		err := json.Unmarshal([]byte(token[1]), &m)
		if err != nil {
			result.AddError("Invalid json")
			continue
		}

		_, err = client.HMSet(token[0], m).Result()
		if err != nil {
			result.AddError(err.Error())
			continue
		}

		if ms := token[2]; ms != "-1" {
			i, err := strconv.ParseInt(ms, 10, 64)
			if err != nil {
				result.AddError(err.Error())
				continue
			}

			d := time.Millisecond * time.Duration(i)
			_, err = client.PExpire(token[0], d).Result()
			if err != nil {
				result.AddError(err.Error())
				continue
			}
		}
	}

	elapsed := time.Since(from)
	fmt.Fprintf(os.Stderr, "[%02d]hmset: %d, Elapsed: %s\n", i, lc, elapsed)

	result.Lines = lc

	// 入力行が尽きたら受信件数をchResultに返して終了
	chResult <- result
}
