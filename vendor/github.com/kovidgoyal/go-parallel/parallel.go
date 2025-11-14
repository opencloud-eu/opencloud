package parallel

import (
	"fmt"
	"iter"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

var _ = fmt.Print

type PanicError struct {
	frames      []runtime.Frame
	panic_value any
}

const indent_lead = "  "

func format_frame_line(frame runtime.Frame) string {
	return fmt.Sprintf("\r\n%s%s%s:%d", indent_lead, frame.Function, frame.File, frame.Line)
}

func (e *PanicError) walk(level int, yield func(string) bool) bool {
	s := "Panic"
	cause := fmt.Sprintf("%v", e.panic_value)
	if _, ok := e.panic_value.(*PanicError); ok {
		cause = "sub-panic (see below)"
	}
	if level > 0 {
		s = "\r\n--> Sub-panic"
	}
	if !yield(fmt.Sprintf("%s caused by: %s\r\nStack trace (most recent call first):", s, cause)) {
		return false
	}
	for _, f := range e.frames {
		if !yield(format_frame_line(f)) {
			return false
		}
	}
	if sp, ok := e.panic_value.(*PanicError); ok {
		return sp.walk(level+1, yield)
	}
	return true
}

func (e *PanicError) lines() iter.Seq[string] {
	return func(yield func(string) bool) {
		e.walk(0, yield)
	}
}

func (e *PanicError) Error() string {
	return strings.Join(slices.Collect(e.lines()), "")
}

func (e *PanicError) Unwrap() error {
	if ans, ok := e.panic_value.(*PanicError); ok {
		return ans
	}
	return nil
}

// Format a stack trace on panic and return it as an error
func Format_stacktrace_on_panic(r any, skip_frames int) (err *PanicError) {
	pcs := make([]uintptr, 512)
	n := runtime.Callers(2+skip_frames, pcs)
	var ans []runtime.Frame
	frames := runtime.CallersFrames(pcs[:n])
	found_first_frame := false
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		if !found_first_frame {
			if strings.HasPrefix(frame.Function, "runtime.") {
				continue
			}
			found_first_frame = true
		}
		ans = append(ans, frame)
	}
	return &PanicError{frames: ans, panic_value: r}
}

// Run the specified function in parallel over chunks from the specified range.
// If the function panics, it is turned into a regular error. If multiple function calls panic,
// any one of the panics will be returned.
func Run_in_parallel_over_range(num_procs int, f func(int, int), start, limit int) (err error) {
	num_items := limit - start
	if num_procs <= 0 {
		num_procs = runtime.GOMAXPROCS(0)
	}
	num_procs = max(1, min(num_procs, num_items))
	if num_procs < 2 {
		defer func() {
			if r := recover(); r != nil {
				err = Format_stacktrace_on_panic(r, 1)
			}
		}()
		f(start, limit)
		return
	}
	err_once := sync.Once{}
	chunk_sz := max(1, num_items/num_procs)
	var wg sync.WaitGroup
	for start < limit {
		end := min(start+chunk_sz, limit)
		wg.Add(1)
		go func(start, end int) {
			defer func() {
				if r := recover(); r != nil {
					perr := Format_stacktrace_on_panic(r, 1)
					err_once.Do(func() { err = perr })
				}
				wg.Done()
			}()
			f(start, end)
		}(start, end)
		start = end
	}
	wg.Wait()
	return
}

// Run the specified function in parallel over chunks from the specified range.
// If the function panics, it is turned into a regular error. If the function
// returns an error it is returned. If multiple function calls panic or return errors,
// any one of them will be returned.
func Run_in_parallel_over_range_with_error(num_procs int, f func(int, int) error, start, limit int) (err error) {
	num_items := limit - start
	if num_procs <= 0 {
		num_procs = runtime.GOMAXPROCS(0)
	}
	num_procs = max(1, min(num_procs, num_items))
	if num_procs < 2 {
		defer func() {
			if r := recover(); r != nil {
				err = Format_stacktrace_on_panic(r, 1)
			}
		}()
		err = f(start, limit)
		return
	}
	chunk_sz := max(1, num_items/num_procs)
	var wg sync.WaitGroup
	err_once := sync.Once{}
	for start < limit {
		end := min(start+chunk_sz, limit)
		wg.Add(1)
		go func(start, end int) {
			defer func() {
				if r := recover(); r != nil {
					perr := Format_stacktrace_on_panic(r, 1)
					err_once.Do(func() { err = perr })
				}
				wg.Done()
			}()
			if cerr := f(start, end); cerr != nil {
				err_once.Do(func() { err = cerr })
			}
		}(start, end)
		start = end
	}
	wg.Wait()
	return
}

// Run the specified function in parallel until one of them returns true.
// The functions are passed a keep_going variable that they should periodically check and return false if it is false.
// If any of the functions panic, the panic is turned into a regular error and returned.
func Run_in_parallel_to_first_result(num_procs int, f func(start, limit int, keep_going *atomic.Bool) bool, start, limit int) (err error) {
	var keep_going atomic.Bool
	keep_going.Store(true)
	num_items := limit - start
	if num_procs <= 0 {
		num_procs = runtime.GOMAXPROCS(0)
	}
	num_procs = max(1, min(num_procs, num_items))
	if num_procs < 2 {
		defer func() {
			if r := recover(); r != nil {
				err = Format_stacktrace_on_panic(r, 1)
			}
		}()
		f(start, limit, &keep_going)
		return
	}
	chunk_sz := max(1, num_items/num_procs)
	var wg sync.WaitGroup
	var err_once sync.Once
	ch := make(chan bool, num_items/chunk_sz+1)
	for start < limit {
		end := min(start+chunk_sz, limit)
		wg.Add(1)
		go func(start, end int) {
			defer func() {
				if r := recover(); r != nil {
					perr := Format_stacktrace_on_panic(r, 1)
					err_once.Do(func() { err = perr })
				}
				wg.Done()
			}()
			ch <- f(start, end, &keep_going)
		}(start, end)
		start = end
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for x := range ch {
		if x {
			break
		}
	}
	keep_going.Store(false)
	wg.Wait()
	return
}
