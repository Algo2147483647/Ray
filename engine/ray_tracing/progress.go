package ray_tracing

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const progressInterval = 100 * time.Millisecond

type progressReporter struct {
	label     string
	unit      string
	total     int64
	completed atomic.Int64
	done      chan struct{}
	finished  chan struct{}
	closeOnce sync.Once
}

func newProgressReporter(label, unit string, total int64) *progressReporter {
	reporter := &progressReporter{
		label: label, unit: unit, total: total,
		done: make(chan struct{}), finished: make(chan struct{}),
	}
	go reporter.run()
	return reporter
}

func (r *progressReporter) Add(count int64) {
	if r != nil && count > 0 {
		r.completed.Add(count)
	}
}

func (r *progressReporter) Close() {
	if r == nil {
		return
	}
	r.closeOnce.Do(func() { close(r.done) })
	<-r.finished
}

func (r *progressReporter) run() {
	defer close(r.finished)
	start := time.Now()
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()
	print := func(current int64, complete bool) {
		percent := 100.0
		if r.total > 0 {
			percent = 100 * float64(current) / float64(r.total)
		}
		if complete {
			fmt.Printf("\r%s complete: %d/%d %s (100%%) Time: %v\n", r.label, r.total, r.total, r.unit, time.Since(start).Round(time.Second))
			return
		}
		fmt.Printf("\r%s: %d/%d %s (%.2f%%) Time: %v", r.label, current, r.total, r.unit, percent, time.Since(start).Round(time.Second))
	}
	for {
		select {
		case <-r.done:
			print(r.total, true)
			return
		case <-ticker.C:
			print(r.completed.Load(), false)
		}
	}
}
