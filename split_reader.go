package redisutil

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

// LoadFile 指定ファイルを分割並列入力し、行をchに飛ばす
// TODO: bufio.Reader
func LoadFile(i uint, splitCount uint, file string, chResult chan<- uint, chLine chan<- string, logStep uint) {

	fi, err := os.Stat(file)
	if err != nil {
		panic(err)
	}
	fileSize := fi.Size()
	blockSize := fileSize / int64(splitCount)
	if blockSize <= 0 {
		log.Fatalf("*** Too small file to split(fileSize:%d, splitCount:%d",
			fileSize, splitCount)
	}

	chSplit := make(chan uint)
	// 分割された1ブロック分の処理
	f := func(splitIndex uint, offset int64, endOffset int64) {
		fmt.Fprintf(os.Stderr, "[%02d-%02d]LoadFile: %s %d to %d\n",
			i, splitIndex, file, offset, endOffset)

		fp, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		defer fp.Close()

		_, err = fp.Seek(offset, 0)
		if err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(fp)
		lc := uint(0)
		first := true
		currentPos := offset
		for scanner.Scan() {
			// 現在地がこのブロックの担当分を超えていたら終了
			if currentPos > endOffset {
				break
			}
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			text := scanner.Text()
			// 現在地を求める
			// 改行がLFであること前提・・
			currentPos = currentPos + int64(len(text)) + 1

			// 2番目以降の分割ブロックは行の途中から始まる可能性が高い
			// 最初の行は中途半端なのでskip（前のブロックが処理してくれる）
			if first && offset != 0 {
				first = false
				continue
			}

			lc++
			if lc%logStep == 0 {
				fmt.Fprintf(os.Stderr, "[%02d-%02d]LoadFile: %d\n", i, splitIndex, lc)
			}
			chLine <- text
		}

		// 分割ブロック分の入力終了を通知
		chSplit <- lc
	}

	// 分割数を元に1ブロックあたりの開始位置/終了位置を求めて
	// それぞれgoルーチンで並列入力
	for i := uint(0); i < splitCount; i++ {
		endOffset := int64(i+1) * blockSize
		if endOffset >= fileSize {
			endOffset = fileSize - 1
		}
		go f(i, int64(i)*blockSize, endOffset)
	}

	// 全ての分割ブロックの処理が終わるまで待つ
	var lineCount uint
	for i := uint(0); i < splitCount; i++ {
		lc := <-chSplit
		lineCount = lineCount + lc
	}

	// 1ファイル分の完了を通知
	chResult <- lineCount
}
