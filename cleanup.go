package redisutil

type CleanupFunc func()

type Cleanups struct {
	funcs []CleanupFunc
}

func (c *Cleanups) Add(f CleanupFunc) {
	c.funcs = append(c.funcs, f)
}

func (c *Cleanups) Do() {
	for _, f := range c.funcs {
		defer f()
	}
}
