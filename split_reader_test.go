package redisutil

import (
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestLoadSeeker(t *testing.T) {
	assert := assert.New(t)

	r := &SplitReader{
		MinBlockSize: 1,
	}

	s := `1234
5678
9abc
defg
hijk
lmno
pqrs
tuvw
xyz!
`

	for split := 1; split < len(s)*2; split++ {

		chLine := make(chan string, 16)
		var lines []string

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range chLine {
				lines = append(lines, s)
			}
		}()

		result := r.FromSeeker(0, uint(split), int64(len(s)),
			func() (io.ReadSeeker, error) {
				sr := strings.NewReader(s)
				return sr, nil
			},
			chLine, 1000)

		close(chLine)
		wg.Wait()

		sort.Strings(lines)
		assert.Equal(`1234
5678
9abc
defg
hijk
lmno
pqrs
tuvw
xyz!`, strings.Join(lines, "\n"), "splitCount=%d", split)

		assert.Equal(uint64(9), result, "splitCount=%d", split)
	}

}

func TestCalcSplitPoint(t *testing.T) {
	assert := assert.New(t)

	r := &SplitReader{
		MinBlockSize: 1,
	}

	cs := []struct {
		splitCount uint
		size       int64
		result     []SplitPoint
		err        error
	}{
		{ // 0
			size: 100, splitCount: 1,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 99},
			},
			err: nil,
		},
		{ // 1
			size: 100, splitCount: 2,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 49},
				{BeginOffset: 50, EndOffset: 99},
			},
			err: nil,
		},
		{ // 2
			size: 100, splitCount: 8,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 11},
				{BeginOffset: 12, EndOffset: 23},
				{BeginOffset: 24, EndOffset: 35},
				{BeginOffset: 36, EndOffset: 47},
				{BeginOffset: 48, EndOffset: 59},
				{BeginOffset: 60, EndOffset: 71},
				{BeginOffset: 72, EndOffset: 83},
				{BeginOffset: 84, EndOffset: 99},
			},
			err: nil,
		},
		{ // 3
			size: 100, splitCount: 0,
			result: nil,
			err:    errors.New("splitCount must > 0"),
		},
		{ // 4
			size: 3, splitCount: 3,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 0},
				{BeginOffset: 1, EndOffset: 1},
				{BeginOffset: 2, EndOffset: 2},
			},
			err: nil,
		},
		{ // 5
			size: 3, splitCount: 5,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 0},
				{BeginOffset: 1, EndOffset: 1},
				{BeginOffset: 2, EndOffset: 2},
			},
			err: nil,
		},
	}
	for i, e := range cs {
		ret, err := r.CalcSplitPoint(e.splitCount, e.size)
		if diff := cmp.Diff(e.result, ret); diff != "" {
			t.Errorf("[%d]%s", i, diff)
		}
		if e.err == nil {
			assert.Nil(err, "[%d]should be nil", i)
			if e.result != nil && len(e.result) > 0 {
				assert.Equal(e.size-1, e.result[len(e.result)-1].EndOffset, "[%d]last EndOffset should equal %d", i, e.size-1)
			}
		} else {
			assert.EqualError(err, e.err.Error())
		}
	}
}

func TestCalcSplitPoint2(t *testing.T) {
	assert := assert.New(t)

	r := &SplitReader{}

	cs := []struct {
		splitCount uint
		size       int64
		result     []SplitPoint
		err        error
	}{
		{ // 0
			size: 512, splitCount: 3,
			result: []SplitPoint{
				{BeginOffset: 0, EndOffset: 511},
			},
			err: nil,
		},
		{ // 1
			size: 0, splitCount: 3,
			result: nil,
			err:    errors.New("size must > 0"),
		},
	}
	for i, e := range cs {
		ret, err := r.CalcSplitPoint(e.splitCount, e.size)
		if diff := cmp.Diff(e.result, ret); diff != "" {
			t.Errorf("[%d]%s", i, diff)
		}
		if e.err == nil {
			assert.Nil(err, "[%d]should be nil", i)
			if e.result != nil && len(e.result) > 0 {
				assert.Equal(e.size-1, e.result[len(e.result)-1].EndOffset, "[%d]last EndOffset should equal %d", i, e.size-1)
			}
		} else {
			assert.EqualError(err, e.err.Error())
		}
	}
}
