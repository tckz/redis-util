package main

/*
 * 入力ファイルを並列分割処理するためオフセットの計算が必要になる都合、
 * 改行コードは「必ずLF」であることを前提としている。
 */

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	redisutil "github.com/tckz/redis-util"
)

var version string

func main() {

	showVersion := flag.Bool("version", false, "Show version")
	withoutKey := flag.Bool("without-key", false, "Whether output with key or not")
	out := flag.String("out", "out-", "path/to/prefix-of-file-")
	outSplit := flag.Uint("out-split", 5, "Number of output files")
	compress := flag.String("compress", "none", "{gzip|none=without compression}")
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

	if *outSplit <= 0 {
		log.Fatalf("*** --out-split must be >= 1")
	}

	if *worker <= 0 {
		log.Fatalf("*** --worker must be >= 1")
	}

	chOut := make(chan string, *outSplit)
	chLine := make(chan string, *worker)
	chFile := make(chan uint64)
	from := time.Now()

	wgOut := redisutil.StartWriters(*outSplit, *out, *compress, chOut)

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

	var lineCount int64
	// ファイル入力が全部終わったら、入力行chを閉じる
	go func() {
		for i := 0; i < len(files); i++ {
			lc := <-chFile
			lineCount = lineCount + int64(lc)
		}
		close(chLine)
	}()

	ctx := context.Background()
	chResult := make(chan redisutil.Result, *worker)
	for i := uint(0); i < *worker; i++ {
		// 入力行を受け取ってredisからgetする
		index := i
		go func() {
			chResult <- hgetall(ctx, index, nodes, chLine, chOut, *withoutKey)
		}()
	}

	// 全ての受信goルーチンが終わったら終了
	totalResult := redisutil.NewResult()
	for i := uint(0); i < *worker; i++ {
		result := <-chResult
		totalResult = totalResult.Combine(result)
	}

	close(chOut)
	wgOut.Wait()

	elapsed := time.Since(from)
	fmt.Fprintf(os.Stderr, "Lines: %d, Got: %d, Bad: %d, Elapsed: %s, Errors: %v\n",
		lineCount, totalResult.Lines, totalResult.BadCount, elapsed, totalResult.Errors)
}

func hgetall(ctx context.Context, i uint, nodes []string, chLine <-chan string, chOut chan<- string, withoutKey bool) redisutil.Result {
	client := redisutil.NewRedisClient(nodes)
	defer client.Close()

	var lc uint64
	from := time.Now()

	result := redisutil.NewResult()

	for key := range chLine {
		lc++
		if lc%100000 == 0 {
			fmt.Fprintf(os.Stderr, "[%02d]hgetall: %d\n", i, lc)
		}

		rec, err := client.HGetAll(ctx, key).Result()
		if err != nil {
			result.AddError(err.Error())
			continue
		}

		if len(rec) == 0 {
			result.AddError("Key does not exist")
		} else {
			b, err := json.Marshal(rec)
			if err != nil {
				panic(err)
			}

			s := string(b)
			if withoutKey {
				chOut <- s
			} else {
				chOut <- key + "\t" + s
			}
		}
	}

	elapsed := time.Since(from)
	fmt.Fprintf(os.Stderr, "[%02d]hgetall: %d, Elapsed: %s\n", i, lc, elapsed)

	result.Lines = lc

	return result
}
