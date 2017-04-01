package main

type jobsData chan func()

func (ch jobsData) group() *jobGroup {
	return &jobGroup{ch: ch}
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
