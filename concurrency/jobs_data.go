package concurrency

type JobsData chan func()

func (ch JobsData) Group() *JobGroup {
	return &JobGroup{ch: ch}
}

func (ch JobsData) Run(max int) {
	for max > 0 {
		go func() {
			for job := range ch {
				job()
			}
		}()
		max--
	}
}

func (ch JobsData) Close() {
	close(ch)
}
