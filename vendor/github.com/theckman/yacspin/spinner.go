// Package yacspin provides Yet Another CLi Spinner for Go, taking inspiration
// (and some utility code) from the https://github.com/briandowns/spinner
// project. Specifically this project borrows the default character sets, and
// color mappings to github.com/fatih/color colors, from that project.
//
// This spinner should support all major operating systems, and is tested
// against Linux, MacOS, and Windows.
//
// This spinner also supports an alternate mode of operation when the TERM
// environment variable is set to "dumb". This is discovered automatically when
// constructing the spinner.
//
// Within the yacspin package there are some default spinners stored in the
// yacspin.CharSets variable, and you can also provide your own. There is also a
// list of known colors in the yacspin.ValidColors variable, if you'd like to
// see what's supported. If you've used github.com/fatih/color before, they
// should look familiar.
//
//		cfg := yacspin.Config{
//			Frequency:     100 * time.Millisecond,
//			CharSet:       yacspin.CharSets[59],
//			Suffix:        " backing up database to S3",
//			Message:       "exporting data",
//			StopCharacter: "✓",
//			StopColors:    []string{"fgGreen"},
//		}
//
//		spinner, err := yacspin.New(cfg)
//		// handle the error
//
//		spinner.Start()
//
//		// doing some work
//		time.Sleep(2 * time.Second)
//
//		spinner.Message("uploading data")
//
//		// upload...
//		time.Sleep(2 * time.Second)
//
//		spinner.Stop()
//
// Check out the Config struct to see all of the possible configuration options
// supported by the Spinner.
package yacspin

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
)

type character struct {
	Value string
	Size  int
}

func setToCharSlice(ss []string) ([]character, int) {
	if len(ss) == 0 {
		return nil, 0
	}

	var maxWidth int
	c := make([]character, len(ss))

	for i, s := range ss {
		n := runewidth.StringWidth(s)
		if n > maxWidth {
			maxWidth = n
		}

		c[i] = character{
			Value: s,
			Size:  n,
		}
	}

	return c, maxWidth
}

// TerminalMode is a type to represent the bit flag controlling the terminal
// mode of the spinner, accepted as a field on the Config struct. See the
// comments on the exported constants for more info.
type TerminalMode uint32

const (
	// AutomaticMode configures the constructor function to try and determine if
	// the application using yacspin is being executed within a interactive
	// (teletype [TTY]) session.
	AutomaticMode TerminalMode = 1 << iota

	// ForceTTYMode configures the spinner to operate as if it's running within
	// a TTY session.
	ForceTTYMode

	// ForceNoTTYMode configures the spinner to operate as if it's not running
	// within a TTY session. This mode causes the spinner to only animate when
	// data is being updated. Each animation is rendered on a new line. You can
	// trigger an animation by calling the Message() method, including with the
	// last value it was called with.
	ForceNoTTYMode

	// ForceDumbTerminalMode configures the spinner to operate as if it's
	// running within a dumb terminal. This means the spinner will not use ANSI
	// escape sequences to print colors or to erase each line. Line erasure to
	// animate the spinner is accomplished by overwriting the line with space
	// characters.
	ForceDumbTerminalMode

	// ForceSmartTerminalMode configures the spinner to operate as if it's
	// running within a terminal that supports ANSI escape sequences (VT100).
	// This includes printing of stylized text, and more better line erasure to
	// animate the spinner.
	ForceSmartTerminalMode
)

func termModeAuto(t TerminalMode) bool       { return t&AutomaticMode > 0 }
func termModeForceTTY(t TerminalMode) bool   { return t&ForceTTYMode > 0 }
func termModeForceNoTTY(t TerminalMode) bool { return t&ForceNoTTYMode > 0 }
func termModeForceDumb(t TerminalMode) bool  { return t&ForceDumbTerminalMode > 0 }
func termModeForceSmart(t TerminalMode) bool { return t&ForceSmartTerminalMode > 0 }

// Config is the configuration structure for the Spinner type, which you provide
// to the New() function. Some of the fields can be updated after the *Spinner
// is constructed, others can only be set when calling the constructor. Please
// read the comments for those details.
type Config struct {
	// Frequency specifies how often to animate the spinner. Optimal value
	// depends on the character set you use.
	Frequency time.Duration

	// Writer is the place where we are outputting the spinner, and can't be
	// changed after the *Spinner has been constructed. If omitted (nil), this
	// defaults to os.Stdout.
	Writer io.Writer

	// ShowCursor specifies that the cursor should be shown by the spinner while
	// animating. If it is not shown, the cursor will be restored when the
	// spinner stops. This can't be changed after the *Spinner has been
	// constructed.
	//
	// Please note, if you do not set this to true and the program crashes or is
	// killed, you may need to reset your terminal for the cursor to appear
	// again.
	ShowCursor bool

	// HideCursor describes whether the cursor should be hidden by the spinner
	// while animating. If it is hidden, it will be restored when the spinner
	// stops. This can't be changed after the *Spinner has been constructed.
	//
	// Please note, if the program crashes or is killed you may need to reset
	// your terminal for the cursor to appear again.
	//
	// Deprecated: use ShowCursor instead.
	HideCursor bool

	// SpinnerAtEnd configures the spinner to render the animation at the end of
	// the line instead of the beginning. The default behavior is to render the
	// animated spinner at the beginning of the line.
	SpinnerAtEnd bool

	// ColorAll describes whether to color everything (all) or just the spinner
	// character(s). This cannot be changed after the *Spinner has been
	// constructed.
	ColorAll bool

	// Colors are the colors used for the different printed messages. This
	// respects the ColorAll field.
	Colors []string

	// CharSet is the list of characters to iterate through to draw the spinner.
	CharSet []string

	// Prefix is the string printed immediately before the spinner.
	//
	// If SpinnerAtEnd is set to true, it's recommended that this string start
	// with a space character (` `).
	Prefix string

	// Suffix is the string printed immediately after the spinner and before the
	// message.
	//
	// If SpinnerAtEnd is set to false, it's recommended that this string starts
	// with an space character (` `).
	Suffix string

	// SuffixAutoColon configures whether the spinner adds a colon after the
	// suffix automatically. If there is a message, a colon followed by a space
	// is added to the suffix. Otherwise, if there is no message, or the suffix
	// is only space characters, the colon is omitted.
	//
	// If SpinnerAtEnd is set to true, this option is ignored.
	SuffixAutoColon bool

	// Message is the message string printed by the spinner. If SpinnerAtEnd is
	// set to false and SuffixAutoColon is set to true, the printed line will
	// look like:
	//
	//    <prefix><spinner><suffix>: <message>
	//
	// If SpinnerAtEnd is set to true, the printed line will instead look like
	// this:
	//
	//    <message><prefix><spinner><suffix>
	//
	// In this case, it may be preferred to set the Prefix to empty space (` `).
	Message string

	// StopMessage is the message used when Stop() is called.
	StopMessage string

	// StopCharacter is spinner character used when Stop() is called.
	// Recommended character is ✓, and can be more than just one character.
	StopCharacter string

	// StopColors are the colors used for the Stop() printed line. This respects
	// the ColorAll field.
	StopColors []string

	// StopFailMessage is the message used when StopFail() is called.
	StopFailMessage string

	// StopFailCharacter is the spinner character used when StopFail() is
	// called. Recommended character is ✗, and can be more than just one
	// character.
	StopFailCharacter string

	// StopFailColors are the colors used for the StopFail() printed line. This
	// respects the ColorAll field.
	StopFailColors []string

	// TerminalMode is a bitflag field to control how the internal TTY / "dumb
	// terminal" detection works, to allow consumers to override the internal
	// behaviors. To set this value, it's recommended to use the TerminalMode
	// constants exported by this package.
	//
	// If not set, the New() function implicitly sets it to AutomaticMode. The
	// New() function also returns an error if you have conflicting flags, such
	// as setting ForceTTYMode and ForceNoTTYMode, or if you set AutomaticMode
	// and any other flags set.
	//
	// When in AutomaticMode, the New() function attempts to determine if the
	// current application is running within an interactive (teletype [TTY])
	// session. If it does not appear to be within a TTY, it sets this field
	// value to ForceNoTTYMode | ForceDumbTerminalMode.
	//
	// If this does appear to be a TTY, the ForceTTYMode bitflag will bet set.
	// Similarly, if it's a TTY and the TERM environment variable isn't set to
	// "dumb" the ForceSmartTerminalMode bitflag will also be set.
	//
	// If the deprecated NoTTY Config struct field is set to true, and this
	// field is AutomaticMode, the New() function sets field to the value of
	// ForceNoTTYMode | ForceDumbTerminalMode.
	TerminalMode TerminalMode

	// NotTTY tells the spinner that the Writer should not be treated as a TTY.
	// This results in the animation being disabled, with the animation only
	// happening whenever the data is updated. This mode also renders each
	// update on new line, versus reusing the current line.
	//
	// Deprecated: use TerminalMode field instead by setting it to:
	// ForceNoTTYMode | ForceDumbTerminalMode. This will be removed in a future
	// release.
	NotTTY bool
}

// Spinner is a type representing an animated CLi terminal spinner. The Spinner
// is constructed by the New() function of this package, which accepts a Config
// struct as the only argument. Some of the configuration values cannot be
// changed after the spinner is constructed, so be sure to read the comments
// within the Config type.
//
// Please note, by default the spinner will hide the terminal cursor when
// animating the spinner. If you do not set Config.ShowCursor to true, you need
// to make sure to call the Stop() or StopFail() method to reset the cursor in
// the terminal. Otherwise, after the program exits the cursor will be hidden
// and the user will need to `reset` their terminal.
type Spinner struct {
	writer          io.Writer
	buffer          *bytes.Buffer
	colorAll        bool
	cursorHidden    bool
	suffixAutoColon bool
	termMode        TerminalMode
	spinnerAtEnd    bool

	status       *uint32
	lastPrintLen int
	cancelCh     chan struct{} // send: Stop(), close: StopFail(); both stop painter
	doneCh       chan struct{}
	pauseCh      chan struct{}
	unpauseCh    chan struct{}
	unpausedCh   chan struct{}

	// mutex hat and the fields wearing it
	mu                *sync.Mutex
	frequency         time.Duration
	chars             []character
	maxWidth          int
	index             int
	prefix            string
	suffix            string
	message           string
	colorFn           func(format string, a ...interface{}) string
	stopMsg           string
	stopChar          character
	stopColorFn       func(format string, a ...interface{}) string
	stopFailMsg       string
	stopFailChar      character
	stopFailColorFn   func(format string, a ...interface{}) string
	frequencyUpdateCh chan time.Duration
	dataUpdateCh      chan struct{}
}

const (
	statusStopped uint32 = iota
	statusStarting
	statusRunning
	statusStopping
	statusPausing
	statusPaused
	statusUnpausing
)

// New creates a new unstarted spinner. If stdout does not appear to be a TTY,
// this constructor implicitly sets cfg.NotTTY to true.
func New(cfg Config) (*Spinner, error) {
	if cfg.ShowCursor && cfg.HideCursor {
		return nil, errors.New("cfg.ShowCursor and cfg.HideCursor cannot be true")
	}

	if cfg.TerminalMode == 0 {
		cfg.TerminalMode = AutomaticMode
	}

	// AutomaticMode flag has been set, but so have others
	if termModeAuto(cfg.TerminalMode) && cfg.TerminalMode != AutomaticMode {
		return nil, errors.New("cfg.TerminalMode cannot have AutomaticMode flag set if others are set")
	}

	if termModeForceTTY(cfg.TerminalMode) && termModeForceNoTTY(cfg.TerminalMode) {
		return nil, errors.New("cfg.TerminalMode cannot have both ForceTTYMode and ForceNoTTYMode flags set")
	}

	if termModeForceDumb(cfg.TerminalMode) && termModeForceSmart(cfg.TerminalMode) {
		return nil, errors.New("cfg.TerminalMode cannot have both ForceDumbTerminalMode and ForceSmartTerminalMode flags set")
	}

	if cfg.HideCursor {
		cfg.ShowCursor = false
	}

	// cfg.NotTTY compatibility
	if cfg.TerminalMode == AutomaticMode && cfg.NotTTY {
		cfg.TerminalMode = ForceNoTTYMode | ForceDumbTerminalMode
	}

	// is this a dumb terminal / not a TTY?
	if cfg.TerminalMode == AutomaticMode && !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		cfg.TerminalMode = ForceNoTTYMode | ForceDumbTerminalMode
	}

	// if cfg.TerminalMode is still equal to AutomaticMode, this is a TTY
	if cfg.TerminalMode == AutomaticMode {
		cfg.TerminalMode = ForceTTYMode

		if os.Getenv("TERM") == "dumb" {
			cfg.TerminalMode |= ForceDumbTerminalMode
		} else {
			cfg.TerminalMode |= ForceSmartTerminalMode
		}
	}

	buf := bytes.NewBuffer(make([]byte, 2048))
	buf.Reset()

	s := &Spinner{
		buffer:            buf,
		mu:                &sync.Mutex{},
		frequency:         cfg.Frequency,
		status:            uint32Ptr(0),
		frequencyUpdateCh: make(chan time.Duration), // use unbuffered for now to avoid .Frequency() panic
		dataUpdateCh:      make(chan struct{}),

		colorAll:        cfg.ColorAll,
		cursorHidden:    !cfg.ShowCursor,
		spinnerAtEnd:    cfg.SpinnerAtEnd,
		suffixAutoColon: cfg.SuffixAutoColon,
		termMode:        cfg.TerminalMode,
		colorFn:         fmt.Sprintf,
		stopColorFn:     fmt.Sprintf,
		stopFailColorFn: fmt.Sprintf,
	}

	if err := s.Colors(cfg.Colors...); err != nil {
		return nil, err
	}

	if err := s.StopColors(cfg.StopColors...); err != nil {
		return nil, err
	}

	if err := s.StopFailColors(cfg.StopFailColors...); err != nil {
		return nil, err
	}

	if len(cfg.CharSet) == 0 {
		cfg.CharSet = CharSets[9]
	}

	// can only error if the charset is empty, and we prevent that above
	_ = s.CharSet(cfg.CharSet)

	if termModeForceNoTTY(s.termMode) {
		// hack to prevent the animation from running if not a TTY
		s.frequency = time.Duration(math.MaxInt64)
	}

	if cfg.Writer == nil {
		cfg.Writer = colorable.NewColorableStdout()
	}

	s.writer = cfg.Writer

	if len(cfg.Prefix) > 0 {
		s.Prefix(cfg.Prefix)
	}

	if len(cfg.Suffix) > 0 {
		s.Suffix(cfg.Suffix)
	}

	if len(cfg.Message) > 0 {
		s.Message(cfg.Message)
	}

	if len(cfg.StopMessage) > 0 {
		s.StopMessage(cfg.StopMessage)
	}

	if len(cfg.StopCharacter) > 0 {
		s.StopCharacter(cfg.StopCharacter)
	}

	if len(cfg.StopFailMessage) > 0 {
		s.StopFailMessage(cfg.StopFailMessage)
	}

	if len(cfg.StopFailCharacter) > 0 {
		s.StopFailCharacter(cfg.StopFailCharacter)
	}

	return s, nil
}

func (s *Spinner) notifyDataChange() {
	// non-blocking notification
	select {
	case s.dataUpdateCh <- struct{}{}:
	default:
	}
}

// SpinnerStatus describes the status of the spinner. See the package constants
// for the list of all possible statuses
type SpinnerStatus uint32

const (
	// SpinnerStopped is a stopped spinner
	SpinnerStopped SpinnerStatus = iota

	// SpinnerStarting is a starting spinner
	SpinnerStarting

	// SpinnerRunning is a running spinner
	SpinnerRunning

	// SpinnerStopping is a stopping spinner
	SpinnerStopping

	// SpinnerPausing is a pausing spinner
	SpinnerPausing

	// SpinnerPaused is a paused spinner
	SpinnerPaused

	// SpinnerUnpausing is an unpausing spinner
	SpinnerUnpausing
)

func (s SpinnerStatus) String() string {
	switch s {
	case SpinnerStopped:
		return "stopped"
	case SpinnerStarting:
		return "starting"
	case SpinnerRunning:
		return "running"
	case SpinnerStopping:
		return "stopping"
	case SpinnerPausing:
		return "pausing"
	case SpinnerPaused:
		return "paused"
	case SpinnerUnpausing:
		return "unpausing"
	default:
		return fmt.Sprintf("unknown (%d)", s)
	}
}

// Status returns the current status of the spinner. The returned value is of
// type SpinnerStatus, which can be compared against the exported Spinner*
// package-level constants (e.g., SpinnerRunning).
func (s *Spinner) Status() SpinnerStatus {
	return SpinnerStatus(atomic.LoadUint32(s.status))
}

// Start begins the spinner on the Writer in the Config provided to New(). Only
// possible error is if the spinner is already runninng.
func (s *Spinner) Start() error {
	// move us to the starting state
	if !atomic.CompareAndSwapUint32(s.status, statusStopped, statusStarting) {
		return errors.New("spinner already running or shutting down")
	}

	// we now have atomic guarantees of no other goroutines starting or running

	s.mu.Lock()

	if s.frequency < 1 && termModeForceTTY(s.termMode) {
		return errors.New("spinner Frequency duration must be greater than 0 when used within a TTY")
	}

	if len(s.chars) == 0 {
		s.mu.Unlock()

		// move us to the stopped state
		if !atomic.CompareAndSwapUint32(s.status, statusStarting, statusStopped) {
			panic("atomic invariant encountered")
		}

		return errors.New("before starting the spinner a CharSet must be set")
	}

	s.frequencyUpdateCh = make(chan time.Duration, 4)
	s.dataUpdateCh, s.cancelCh = make(chan struct{}, 1), make(chan struct{}, 1)

	s.mu.Unlock()

	// because of the atomic swap above, we know it's safe to mutate these
	// values outside of mutex
	s.doneCh = make(chan struct{})
	s.pauseCh = make(chan struct{}) // unbuffered since we want this to be synchronous

	go s.painter(s.cancelCh, s.dataUpdateCh, s.pauseCh, s.doneCh, s.frequencyUpdateCh)

	// move us to the running state
	if !atomic.CompareAndSwapUint32(s.status, statusStarting, statusRunning) {
		panic("atomic invariant encountered")
	}

	return nil
}

// Pause puts the spinner in a state where it no longer animates or renders
// updates to data. This function blocks until the spinner's internal painting
// goroutine enters a paused state.
//
// If you want to make a few configuration changes and have them to appear at
// the same time, like changing the suffix, message, and color, you can Pause()
// the spinner first and then Unpause() after making the changes.
//
// If the spinner is not running (stopped, paused, or in transition to another
// state) this returns an error.
func (s *Spinner) Pause() error {
	if !atomic.CompareAndSwapUint32(s.status, statusRunning, statusPausing) {
		return errors.New("spinner not running")
	}

	// set up the channels the painter will use
	s.unpauseCh, s.unpausedCh = make(chan struct{}), make(chan struct{})

	// inform the painter to pause as a blocking send
	s.pauseCh <- struct{}{}

	if !atomic.CompareAndSwapUint32(s.status, statusPausing, statusPaused) {
		panic("atomic invariant encountered")
	}

	return nil
}

// Unpause returns the spinner back to a running state after pausing. See
// Pause() documentation for more detail. This function blocks until the
// spinner's internal painting goroutine acknowledges the request to unpause.
//
// If the spinner is not paused this returns an error.
func (s *Spinner) Unpause() error {
	if !atomic.CompareAndSwapUint32(s.status, statusPaused, statusUnpausing) {
		return errors.New("spinner not paused")
	}

	s.unpause()

	if !atomic.CompareAndSwapUint32(s.status, statusUnpausing, statusRunning) {
		panic("atomic invariant encountered")
	}

	return nil
}

func (s *Spinner) unpause() {
	// tell the painter to unpause
	close(s.unpauseCh)

	// wait for the painter to signal it will continue
	<-s.unpausedCh

	// clear the no longer needed channels
	s.unpauseCh = nil
	s.unpausedCh = nil
}

// Stop disables the spinner, and prints the StopCharacter with the StopMessage
// using the StopColors. This blocks until the stopped message is printed. Only
// possible error is if the spinner is not running.
func (s *Spinner) Stop() error {
	return s.stop(false)
}

// StopFail disables the spinner, and prints the StopFailCharacter with the
// StopFailMessage using the StopFailColors. This blocks until the stopped
// message is printed. Only possible error is if the spinner is not running.
func (s *Spinner) StopFail() error {
	return s.stop(true)
}

func (s *Spinner) stop(fail bool) error {
	// move us to a stopping state to protect against concurrent Stop() calls
	wasRunning := atomic.CompareAndSwapUint32(s.status, statusRunning, statusStopping)
	wasPaused := atomic.CompareAndSwapUint32(s.status, statusPaused, statusStopping)

	if !wasRunning && !wasPaused {
		return errors.New("spinner not running or paused")
	}

	// we now have an atomic guarantees of no other threads invoking state changes

	if !fail {
		// this tells the painter to print the StopMessage and not the
		// StopFailMessage
		s.cancelCh <- struct{}{}
	}

	close(s.cancelCh)

	if wasPaused {
		s.unpause()
	}

	// wait for the painter to stop
	<-s.doneCh

	s.mu.Lock()

	s.dataUpdateCh = make(chan struct{})           // prevent panic() in various setter methods
	s.frequencyUpdateCh = make(chan time.Duration) // prevent panic() in .Frequency()

	s.mu.Unlock()

	// because of atomic swaps and channel receive above we know it's
	// safe to mutate these fields outside of the mutex
	s.index = 0
	s.cancelCh = nil
	s.doneCh = nil
	s.pauseCh = nil

	// move us to the stopped state
	if !atomic.CompareAndSwapUint32(s.status, statusStopping, statusStopped) {
		panic("atomic invariant encountered")
	}

	return nil
}

// handleFrequencyUpdate is for when the frequency was changed. This tries to
// see if we should fire the timer now, or change its current duration to match
// the new duration.
func handleFrequencyUpdate(newFrequency time.Duration, timer *time.Timer, lastTick time.Time) {
	// if timer fired, drain the channel
	if !timer.Stop() {
	timerLoop:
		for {
			select {
			case <-timer.C:
			default:
				break timerLoop
			}
		}
	}

	timeSince := time.Since(lastTick)

	// if we've exceeded the new delay trigger timer immediately
	if timeSince >= newFrequency {
		timer.Reset(0)
		return
	}

	timer.Reset(newFrequency - timeSince)
}

func (s *Spinner) painter(cancel, dataUpdate, pause <-chan struct{}, done chan<- struct{}, frequencyUpdate <-chan time.Duration) {
	timer := time.NewTimer(0)
	var lastTick time.Time

	for {
		select {
		case <-timer.C:
			lastTick = time.Now()

			s.paintUpdate(timer, true)

		case <-pause:
			<-s.unpauseCh
			close(s.unpausedCh)

		case <-dataUpdate:
			// if this is not a TTY: animate the spinner on the data update
			s.paintUpdate(timer, termModeForceNoTTY(s.termMode))

		case frequency := <-frequencyUpdate:
			handleFrequencyUpdate(frequency, timer, lastTick)

		case _, ok := <-cancel:
			defer close(done)

			timer.Stop()

			s.paintStop(ok)

			return
		}
	}
}

func (s *Spinner) paintUpdate(timer *time.Timer, animate bool) {
	s.mu.Lock()

	p := s.prefix
	m := s.message
	suf := s.suffix
	mw := s.maxWidth
	cFn := s.colorFn
	d := s.frequency
	index := s.index

	if animate {
		s.index++

		if s.index == len(s.chars) {
			s.index = 0
		}
	} else {
		// for data updates use the last spinner char
		index--

		if index < 0 {
			index = len(s.chars) - 1
		}
	}

	c := s.chars[index]

	s.mu.Unlock()

	defer s.buffer.Reset()

	if termModeForceSmart(s.termMode) {
		if err := erase(s.buffer); err != nil {
			panic(fmt.Sprintf("failed to erase line: %v", err))
		}

		if s.cursorHidden {
			if err := hideCursor(s.buffer); err != nil {
				panic(fmt.Sprintf("failed to hide cursor: %v", err))
			}
		}

		if _, err := paint(s.buffer, mw, c, p, m, suf, s.suffixAutoColon, s.colorAll, s.spinnerAtEnd, false, termModeForceNoTTY(s.termMode), cFn); err != nil {
			panic(fmt.Sprintf("failed to paint line: %v", err))
		}
	} else {
		if err := s.eraseDumbTerm(s.buffer); err != nil {
			panic(fmt.Sprintf("failed to erase line: %v", err))
		}

		n, err := paint(s.buffer, mw, c, p, m, suf, s.suffixAutoColon, false, s.spinnerAtEnd, false, termModeForceNoTTY(s.termMode), fmt.Sprintf)
		if err != nil {
			panic(fmt.Sprintf("failed to paint line: %v", err))
		}

		s.lastPrintLen = n
	}

	if s.buffer.Len() > 0 {
		if _, err := s.writer.Write(s.buffer.Bytes()); err != nil {
			panic(fmt.Sprintf("failed to output buffer to writer: %v", err))
		}
	}

	if animate {
		timer.Reset(d)
	}
}

func (s *Spinner) paintStop(chanOk bool) {
	var m string
	var c character
	var cFn func(format string, a ...interface{}) string

	s.mu.Lock()

	if chanOk {
		c = s.stopChar
		cFn = s.stopColorFn
		m = s.stopMsg
	} else {
		c = s.stopFailChar
		cFn = s.stopFailColorFn
		m = s.stopFailMsg
	}

	p := s.prefix
	suf := s.suffix
	mw := s.maxWidth

	s.mu.Unlock()

	defer s.buffer.Reset()

	if termModeForceSmart(s.termMode) {
		if err := erase(s.buffer); err != nil {
			panic(fmt.Sprintf("failed to erase line: %v", err))
		}

		if s.cursorHidden {
			if err := unhideCursor(s.buffer); err != nil {
				panic(fmt.Sprintf("failed to hide cursor: %v", err))
			}
		}

		if c.Size > 0 || len(m) > 0 {
			// paint the line with a newline as it's the final line
			if _, err := paint(s.buffer, mw, c, p, m, suf, s.suffixAutoColon, s.colorAll, s.spinnerAtEnd, true, termModeForceNoTTY(s.termMode), cFn); err != nil {
				panic(fmt.Sprintf("failed to paint line: %v", err))
			}
		}
	} else {
		if err := s.eraseDumbTerm(s.buffer); err != nil {
			panic(fmt.Sprintf("failed to erase line: %v", err))
		}

		if c.Size > 0 || len(m) > 0 {
			if _, err := paint(s.buffer, mw, c, p, m, suf, s.suffixAutoColon, false, s.spinnerAtEnd, true, termModeForceNoTTY(s.termMode), fmt.Sprintf); err != nil {
				panic(fmt.Sprintf("failed to paint line: %v", err))
			}
		}

		s.lastPrintLen = 0
	}

	if s.buffer.Len() > 0 {
		if _, err := s.writer.Write(s.buffer.Bytes()); err != nil {
			panic(fmt.Sprintf("failed to output buffer to writer: %v", err))
		}
	}
}

// erase clears the line
func erase(w io.Writer) error {
	_, err := fmt.Fprint(w, "\r\033[K\r")
	return err
}

// eraseDumbTerm clears the line on dumb terminals
func (s *Spinner) eraseDumbTerm(w io.Writer) error {
	if termModeForceNoTTY(s.termMode) {
		// non-TTY outputs use \n instead of line erasure,
		// so return early
		return nil
	}

	clear := "\r" + strings.Repeat(" ", s.lastPrintLen) + "\r"

	_, err := fmt.Fprint(w, clear)
	return err
}

func hideCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\r\033[?25l\r")
	return err
}

func unhideCursor(w io.Writer) error {
	_, err := fmt.Fprint(w, "\r\033[?25h\r")
	return err
}

// padChar pads the spinner character so suffix / message offset from left is
// consistent
func padChar(char character, maxWidth int) string {
	padSize := maxWidth - char.Size
	return char.Value + strings.Repeat(" ", padSize)
}

// paint writes a single line to the w, using the provided character, message,
// and color function
func paint(w io.Writer, maxWidth int, char character, prefix, message, suffix string, suffixAutoColon, colorAll, spinnerAtEnd, finalPaint, notTTY bool, colorFn func(format string, a ...interface{}) string) (int, error) {
	var output string

	switch char.Size {
	case 0:
		if colorAll {
			output = colorFn(message)
			break
		}

		output = message

	default:
		c := padChar(char, maxWidth)

		if spinnerAtEnd {
			if colorAll {
				output = colorFn("%s%s%s%s", message, prefix, c, suffix)
				break
			}

			output = fmt.Sprintf("%s%s%s%s", message, prefix, colorFn(c), suffix)
			break
		}

		if suffixAutoColon { // also implicitly !spinnerAtEnd
			if len(strings.TrimSpace(suffix)) > 0 && len(message) > 0 && message != "\n" {
				suffix += ": "
			}
		}

		if colorAll {
			output = colorFn("%s%s%s%s", prefix, c, suffix, message)
			break
		}

		output = fmt.Sprintf("%s%s%s%s", prefix, colorFn(c), suffix, message)
	}

	if finalPaint || notTTY {
		output += "\n"
	}

	return fmt.Fprint(w, output)
}

// Frequency updates the frequency of the spinner being animated.
func (s *Spinner) Frequency(d time.Duration) error {
	if d < 1 {
		return errors.New("duration must be greater than 0")
	}

	if termModeForceNoTTY(s.termMode) {
		// when output target is not a TTY, we don't animate spinner
		// so there is no need to update the frequency
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.frequency = d

	// non-blocking notification
	select {
	case s.frequencyUpdateCh <- d:
	default:
	}

	return nil
}

// Prefix updates the Prefix before the spinner character.
func (s *Spinner) Prefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prefix = prefix

	s.notifyDataChange()
}

// Suffix updates the Suffix printed after the spinner character and before the
// message. It's recommended that this start with an empty space.
func (s *Spinner) Suffix(suffix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.suffix = suffix

	s.notifyDataChange()
}

// Message updates the Message displayed after the suffix.
func (s *Spinner) Message(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.message = message

	s.notifyDataChange()
}

// Colors updates the github.com/fatih/colors for printing the spinner line.
// ColorAll config parameter controls whether only the spinner character is
// printed with these colors, or the whole line.
//
// StopColors() is the method to control the colors in the stop message.
func (s *Spinner) Colors(colors ...string) error {
	colorFn, err := colorFunc(colors...)
	if err != nil {
		return fmt.Errorf("failed to build color function: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.colorFn = colorFn

	s.notifyDataChange()

	return nil
}

// StopMessage updates the Message used when Stop() is called.
func (s *Spinner) StopMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopMsg = message

	s.notifyDataChange()
}

// StopColors updates the colors used for the stop message. See Colors() method
// documentation for more context.
//
// StopFailColors() is the method to control the colors in the failed stop
// message.
func (s *Spinner) StopColors(colors ...string) error {
	colorFn, err := colorFunc(colors...)
	if err != nil {
		return fmt.Errorf("failed to build stop color function: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopColorFn = colorFn

	s.notifyDataChange()

	return nil
}

// StopCharacter sets the single "character" to use for the spinner when
// stopping. Recommended character is ✓.
func (s *Spinner) StopCharacter(char string) {
	n := runewidth.StringWidth(char)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopChar = character{Value: char, Size: n}

	if n > s.maxWidth {
		s.maxWidth = n
	}

	s.notifyDataChange()
}

// StopFailMessage updates the Message used when StopFail() is called.
func (s *Spinner) StopFailMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopFailMsg = message

	s.notifyDataChange()
}

// StopFailColors updates the colors used for the StopFail message. See Colors() method
// documentation for more context.
func (s *Spinner) StopFailColors(colors ...string) error {
	colorFn, err := colorFunc(colors...)
	if err != nil {
		return fmt.Errorf("failed to build stop fail color function: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopFailColorFn = colorFn

	s.notifyDataChange()

	return nil
}

// StopFailCharacter sets the single "character" to use for the spinner when
// stopping for a failure. Recommended character is ✗.
func (s *Spinner) StopFailCharacter(char string) {
	n := runewidth.StringWidth(char)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopFailChar = character{Value: char, Size: n}

	if n > s.maxWidth {
		s.maxWidth = n
	}

	s.notifyDataChange()
}

// CharSet updates the set of characters (strings) to use for the spinner. You
// can provide your own, or use one from the yacspin.CharSets variable.
//
// The character sets available in the CharSets variable are from the
// https://github.com/briandowns/spinner project.
func (s *Spinner) CharSet(cs []string) error {
	if len(cs) == 0 {
		return errors.New("failed to set character set:  must provide at least one string")
	}

	chars, mw := setToCharSlice(cs)
	s.mu.Lock()
	defer s.mu.Unlock()

	if n := s.stopChar.Size; n > mw {
		mw = s.stopChar.Size
	}

	if n := s.stopFailChar.Size; n > mw {
		mw = n
	}

	s.chars = chars
	s.maxWidth = mw
	s.index = 0

	return nil
}

// Reverse flips the character set order of the spinner characters.
func (s *Spinner) Reverse() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, j := 0, len(s.chars)-1; i < j; {
		s.chars[i], s.chars[j] = s.chars[j], s.chars[i]
		i++
		j--
	}

	s.index = 0
}

func uint32Ptr(u uint32) *uint32 { return &u }
