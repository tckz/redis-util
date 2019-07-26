package redisutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func StartWriters(outSplit uint, out string, compress string, chLine <-chan string) *sync.WaitGroup {
	wgOut := &sync.WaitGroup{}
	for i := uint(0); i < outSplit; i++ {
		outFn := fmt.Sprintf("%s%03d", out, i)
		d := filepath.Dir(outFn)
		if err := os.MkdirAll(d, os.ModePerm); err != nil {
			panic(err)
		}

		ct := GetCompressionType(compress)

		outFn = outFn + ct.Ext
		cleanups := &Cleanups{}
		f, err := os.Create(outFn)
		if err != nil {
			panic(err)
		}
		cleanups.Add(func() { f.Close() })

		w, cleanup, err := DecorateWriter(ct.Type, f)
		if err != nil {
			cleanups.Do()
			panic(err)
		}
		cleanups.Add(cleanup)

		wgOut.Add(1)
		go func() {
			defer wgOut.Done()
			defer cleanups.Do()

			for e := range chLine {
				fmt.Fprintln(w, e)
			}
		}()
	}

	return wgOut
}
