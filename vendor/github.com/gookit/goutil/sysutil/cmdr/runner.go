package cmdr

import (
	"fmt"
	"strings"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/cliutil/cmdline"
	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil/textutil"
	"github.com/gookit/goutil/x/ccolor"
)

// Task struct
type Task struct {
	err   error
	index int

	// ID for task
	ID  string
	Cmd *Cmd

	// BeforeRun hook
	BeforeRun func(t *Task)
	PrevCond  func(prev *Task) bool
}

// NewTask instance
func NewTask(cmd *Cmd) *Task {
	return &Task{
		Cmd: cmd,
	}
}

// get task id by cmd.Name
func (t *Task) ensureID(idx int) {
	t.index = idx
	if t.ID != "" {
		return
	}

	id := t.Cmd.IDString()
	if t.Cmd.Name == "" {
		id += mathutil.String(idx)
	}
	t.ID = id
}

var rpl = textutil.NewVarReplacer("$").DisableFlatten()

// RunWith command
func (t *Task) RunWith(ctx maputil.Data) error {
	cmdVars := ctx.StringMap("cmdVars")

	if len(cmdVars) > 0 {
		// rpl := strutil.NewReplacer(cmdVars)
		for i, val := range t.Cmd.Args {
			if strings.ContainsRune(val, '$') {
				t.Cmd.Args[i] = rpl.RenderSimple(val, cmdVars)
			}
		}
	}

	return t.Run()
}

// Run command
func (t *Task) Run() error {
	if t.BeforeRun != nil {
		t.BeforeRun(t)
	}

	t.err = t.Cmd.Run()
	return t.err
}

// Err get
func (t *Task) Err() error {
	return t.err
}

// Index get
func (t *Task) Index() int {
	return t.index
}

// Cmdline get
func (t *Task) Cmdline() string {
	return t.Cmd.Cmdline()
}

// IsSuccess of task
func (t *Task) IsSuccess() bool {
	return t.err == nil
}

// RunnerHookFn func
type RunnerHookFn func(r *Runner, t *Task) bool

// Runner use for batch run multi task commands
type Runner struct {
	prev *Task
	// task name to index
	idMap map[string]int
	tasks []*Task
	// Errs on run tasks, key is Task.ID
	Errs errorx.ErrMap

	// TODO Concurrent run

	// Workdir common workdir
	Workdir string
	// EnvMap will append to task.Cmd on run
	EnvMap map[string]string

	// Params for add custom params
	Params maputil.Map

	// DryRun dry run all commands
	DryRun bool
	// OutToStd stdout and stderr
	OutToStd bool
	// IgnoreErr continue on error
	IgnoreErr bool
	// BeforeRun hooks on each task. return false to skip current task.
	BeforeRun func(r *Runner, t *Task) bool
	// AfterRun hook on each task. return false to stop running.
	AfterRun func(r *Runner, t *Task) bool
}

// NewRunner instance with config func
func NewRunner(fns ...func(rr *Runner)) *Runner {
	rr := &Runner{
		idMap:  make(map[string]int, 0),
		tasks:  make([]*Task, 0),
		Errs:   make(errorx.ErrMap),
		Params: make(maputil.Map),
	}

	rr.OutToStd = true
	for _, fn := range fns {
		fn(rr)
	}
	return rr
}

// WithOutToStd set
func (r *Runner) WithOutToStd() *Runner {
	r.OutToStd = true
	return r
}

// Add multitask at once
func (r *Runner) Add(tasks ...*Task) *Runner {
	for _, task := range tasks {
		r.AddTask(task)
	}
	return r
}

// AddTask add one task
func (r *Runner) AddTask(task *Task) *Runner {
	if task.Cmd == nil {
		panic("task command cannot be empty")
	}

	idx := len(r.tasks)
	task.ensureID(idx)

	// TODO check id repeat
	r.idMap[task.ID] = idx
	r.tasks = append(r.tasks, task)
	return r
}

// AddCmd commands
func (r *Runner) AddCmd(cmds ...*Cmd) *Runner {
	for _, cmd := range cmds {
		r.AddTask(&Task{Cmd: cmd})
	}
	return r
}

// GitCmd quick a git command task
func (r *Runner) GitCmd(subCmd string, args ...string) *Runner {
	return r.AddTask(&Task{
		Cmd: NewGitCmd(subCmd, args...),
	})
}

// CmdWithArgs a command task
func (r *Runner) CmdWithArgs(cmdName string, args ...string) *Runner {
	return r.AddTask(&Task{
		Cmd: NewCmd(cmdName, args...),
	})
}

// CmdWithAnys a command task
func (r *Runner) CmdWithAnys(cmdName string, args ...any) *Runner {
	return r.AddTask(&Task{
		Cmd: NewCmd(cmdName, arrutil.SliceToStrings(args)...),
	})
}

// AddCmdline as a command task
func (r *Runner) AddCmdline(line string) *Runner {
	bin, args := cmdline.NewParser(line).BinAndArgs()

	return r.AddTask(&Task{
		Cmd: NewCmd(bin, args...),
	})
}

// Run all tasks
func (r *Runner) Run() error {
	// do run tasks
	for i, task := range r.tasks {
		if r.BeforeRun != nil && !r.BeforeRun(r, task) {
			continue
		}

		if r.prev != nil && task.PrevCond != nil && !task.PrevCond(r.prev) {
			continue
		}

		if r.DryRun {
			ccolor.Infof("DRY-RUN: task#%d execute completed\n\n", i+1)
			continue
		}

		if !r.RunTask(task) {
			break
		}
		fmt.Println() // with newline.
	}

	if len(r.Errs) == 0 {
		return nil
	}
	return r.Errs
}

// StepRun one command
func (r *Runner) StepRun() error {
	return nil // TODO
}

// RunTask command
func (r *Runner) RunTask(task *Task) (goon bool) {
	if len(r.EnvMap) > 0 {
		task.Cmd.AppendEnv(r.EnvMap)
	}

	if r.OutToStd && !task.Cmd.HasStdout() {
		task.Cmd.ToOSStdoutStderr()
	}

	// common workdir
	if r.Workdir != "" && task.Cmd.Dir == "" {
		task.Cmd.WithWorkDir(r.Workdir)
	}

	// do running
	if err := task.RunWith(r.Params); err != nil {
		r.Errs[task.ID] = err
		ccolor.Errorf("Task#%d run error: %s\n", task.Index()+1, err)

		// not ignore error, stop.
		if !r.IgnoreErr {
			return false
		}
	}

	if r.AfterRun != nil && !r.AfterRun(r, task) {
		return false
	}

	// store prev
	r.prev = task
	return true
}

// Len of tasks
func (r *Runner) Len() int {
	return len(r.tasks)
}

// Reset instance
func (r *Runner) Reset() *Runner {
	r.prev = nil
	r.tasks = make([]*Task, 0)
	r.idMap = make(map[string]int, 0)
	return r
}

// TaskIDs get
func (r *Runner) TaskIDs() []string {
	ss := make([]string, 0, len(r.idMap))
	for id := range r.idMap {
		ss = append(ss, id)
	}
	return ss
}

// Prev task instance after running
func (r *Runner) Prev() *Task {
	return r.prev
}

// Task get by id name
func (r *Runner) Task(id string) (*Task, error) {
	if idx, ok := r.idMap[id]; ok {
		return r.tasks[idx], nil
	}
	return nil, fmt.Errorf("task %q is not exists", id)
}
