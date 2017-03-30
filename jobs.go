package main

import "sync"

type jobsData chan func()

var jobsRunner jobsData = make(jobsData)
var parallelWalkRunner jobsData = make(jobsData)

func (ch jobsData) group() *jobsGroup {
	return &jobsGroup{ch: ch}
}

func (ch jobsData) run(max int) {
	for max > 0 {
		go func() {
			for job := range ch {
				job()
			}
		}()
		max--
	}
}

func (ch jobsData) close() {
	close(ch)
}

type jobsGroup struct {
	ch   jobsData
	wg   sync.WaitGroup
	err  error
	lock sync.Mutex
}

func (g *jobsGroup) Dispatch(fn func() error) {
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

func (g *jobsGroup) Finish() error {
	g.wg.Wait()
	return g.err
}
