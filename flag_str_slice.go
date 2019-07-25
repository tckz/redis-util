package redisutil

import (
	"strings"
)

type StrSlice []string

func (s *StrSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *StrSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}
