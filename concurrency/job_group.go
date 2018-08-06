package concurrency

import "sync"

type JobGroup struct {
	ch   JobsData
	wg   sync.WaitGroup
	err  error
	lock sync.Mutex
}

func (g *JobGroup) Dispatch(fn func() error) {
	g.wg.Add(1)

	g.ch <- func() {
		var err error

		defer func() {
			if err != nil {
				g.lock.Lock()
				g.err = err
				g.lock.Unlock()
			}
		}()
		defer g.wg.Done()

		err = fn()
	}
}

func (g *JobGroup) Finish() error {
	g.wg.Wait()
	return g.err
}
