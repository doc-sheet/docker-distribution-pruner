package main

import "sync"

type jobGroup struct {
	ch   jobsData
	wg   sync.WaitGroup
	err  error
	lock sync.Mutex
}

func (g *jobGroup) dispatch(fn func() error) {
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

func (g *jobGroup) finish() error {
	g.wg.Wait()
	return g.err
}
