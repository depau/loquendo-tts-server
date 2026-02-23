package threads

import (
	"runtime"

	"golang.org/x/sys/windows"
)

type req struct {
	fn   func() (uintptr, uintptr, error)
	done chan resp
}

type resp struct {
	r1 uintptr
	r2 uintptr
	e  error
}

// ThreadExecutor runs all submitted calls on one dedicated OS thread.
type ThreadExecutor struct {
	ch chan req
}

func NewExecutor(buffer int) *ThreadExecutor {
	e := &ThreadExecutor{ch: make(chan req, buffer)}
	go e.loop()
	return e
}

func (e *ThreadExecutor) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for r := range e.ch {
		r1, r2, err := r.fn()
		r.done <- resp{r1: r1, r2: r2, e: err}
	}
}

func (e *ThreadExecutor) Close() { close(e.ch) }

// CallProc schedules proc.Call(args...) on the executor thread and blocks.
func (e *ThreadExecutor) CallProc(proc *windows.Proc, args ...uintptr) (uintptr, uintptr, error) {
	done := make(chan resp, 1)

	e.ch <- req{
		fn: func() (uintptr, uintptr, error) {
			return proc.Call(args...)
		},
		done: done,
	}

	r := <-done
	return r.r1, r.r2, r.e
}
