package redisutil

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type SplitReader struct {
	MinBlockSize int64
}

type SplitPoint struct {
	BeginOffset int64
	EndOffset   int64
}

func DecorateWriter(compression CompressionType, w io.Writer) (io.Writer, CleanupFunc, error) {
	switch compression {
	case CompressionNone:
		return w, func() {}, nil
	case CompressionGzip:
		gzw := gzip.NewWriter(w)
		return gzw, func() { gzw.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("unknown compression type: %s", compression)
	}
}

func DecorateReader(fn string, r io.Reader) (io.Reader, CleanupFunc, error) {
	if strings.HasSuffix(fn, ".gz") {
		if gzr, err := gzip.NewReader(r); err != nil {
			return nil, nil, err
		} else {
			return gzr, func() { gzr.Close() }, nil
		}
	} else if strings.HasSuffix(fn, ".bz2") {
		return bzip2.NewReader(r), func() {}, nil
	}
	return r, func() {}, nil
}

func (s *SplitReader) CalcSplitPoint(splitCount uint, size int64) ([]SplitPoint, error) {
	if splitCount <= 0 {
		return nil, fmt.Errorf("splitCount must > 0")
	}

	if size <= 0 {
		return nil, fmt.Errorf("size must > 0")
	}

	blockSize := size / int64(splitCount)
	minBlockSize := s.MinBlockSize
	if minBlockSize == 0 {
		minBlockSize = 1024
	}
	if blockSize <= minBlockSize {
		blockSize = minBlockSize
	}

	ret := make([]SplitPoint, 0, splitCount)
	for i, beginOffset := uint(0), int64(0); i < splitCount && beginOffset < size; i++ {
		endOffset := beginOffset + blockSize - 1
		if endOffset >= (size-1) || i == splitCount-1 {
			endOffset = size - 1
		}

		ret = append(ret, SplitPoint{
			BeginOffset: beginOffset,
			EndOffset:   endOffset,
		})

		beginOffset = endOffset + 1
	}

	return ret, nil
}

// LoadFile 指定ファイルを分割並列入力し、行をchに飛ばす
func (s *SplitReader) LoadFile(i uint, splitCount uint, file string, chResult chan<- uint, chLine chan<- string, logStep uint) {
	fi, err := os.Stat(file)
	if err != nil {
		panic(err)
	}
	fileSize := fi.Size()

	s.LoadSeeker(i, splitCount, fileSize,
		func() (io.ReadSeeker, error) {
			return os.Open(file)
		},
		chResult, chLine, logStep)
}

func (s *SplitReader) LoadSeeker(i uint, splitCount uint, fileSize int64, genSeeker func() (io.ReadSeeker, error), chResult chan<- uint, chLine chan<- string, logStep uint) {

	splitPoints, err := s.CalcSplitPoint(splitCount, fileSize)
	if err != nil {
		panic(err)
	}

	chSplit := make(chan uint, len(splitPoints))
	// 分割された1ブロック分の処理
	f := func(splitIndex int, beginOffset int64, endOffset int64) {
		fmt.Fprintf(os.Stderr, "[%02d-%02d]LoadSeeker: %d to %d\n",
			i, splitIndex, beginOffset, endOffset)

		fp, err := genSeeker()
		if err != nil {
			panic(err)
		}
		defer func() {
			if c, ok := fp.(io.Closer); ok {
				c.Close()
			}
		}()

		_, err = fp.Seek(beginOffset, io.SeekStart)
		if err != nil {
			panic(err)
		}

		reader := bufio.NewReader(fp)
		lc := uint(0)
		first := true
		currentPos := beginOffset
		for {
			// 現在地がこのブロックの担当分を超えていたら終了
			if currentPos > (endOffset + 1) {
				break
			}
			text, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}

			// 2番目以降の分割ブロックは行の途中から始まる可能性が高い
			// 最初の行は中途半端なのでskip（前のブロックが処理）
			if first && currentPos == beginOffset && beginOffset != 0 {
				first = false
			} else {
				lc++
				if lc%logStep == 0 {
					fmt.Fprintf(os.Stderr, "[%02d-%02d]LoadFile: %d\n", i, splitIndex, lc)
				}
				chLine <- strings.TrimRight(text, "\r\n")
			}

			// 現在地を求める
			currentPos = currentPos + int64(len(text))
		}

		// 分割ブロック分の入力終了を通知
		chSplit <- lc
	}

	wg := &sync.WaitGroup{}
	splittedCount := len(splitPoints)
	for i, e := range splitPoints {
		index := i
		point := e
		wg.Add(1)
		go func() {
			defer wg.Done()
			f(index, point.BeginOffset, point.EndOffset)
		}()
	}

	var lineCount uint
	for i := 0; i < splittedCount; i++ {
		lc := <-chSplit
		lineCount = lineCount + lc
	}
	wg.Wait()

	// 1ファイル分の完了を通知
	chResult <- lineCount
}
