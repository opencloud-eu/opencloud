package inotifywaitgo

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// WatchPath starts watching a path for new files and returns the file name (abspath) when a new file is finished writing.
func WatchPath(s *Settings) {
	// Check if inotifywait is installed
	if ok, err := checkDependencies(); !ok || err != nil {
		s.ErrorChan <- errors.New(NOT_INSTALLED)
		return
	}

	// Check if the directory exists
	if _, err := os.Stat(s.Dir); os.IsNotExist(err) {
		s.ErrorChan <- errors.New(DIR_NOT_EXISTS)
		return
	}

	// Stop any existing inotifywait processes
	if s.KillOthers {
		killOthers()
	}

	// Generate shell command
	cmdString, err := GenerateShellCommands(s)
	if err != nil {
		s.ErrorChan <- err
		return
	}

	// Start inotifywait in the input directory and watch for close_write events
	cmd := exec.Command(cmdString[0], cmdString[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.ErrorChan <- err
		return
	}

	if err := cmd.Start(); err != nil {
		s.ErrorChan <- err
		return
	}

	logger := s.Log
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	// Read the output of inotifywait and split it into lines
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		logger.Debug(line)

		parts, err := parseLine(line)
		if err != nil || len(parts) < 2 {
			s.ErrorChan <- errors.New(INVALID_OUTPUT)
			continue
		}

		prefix, file := parts[0], parts[2]
		eventStrs := strings.Split(parts[1], ",")

		if s.Verbose {
			for _, eventStr := range eventStrs {
				logger.Debug("eventStr: <%s>, <%s>", eventStr, line)
			}
		}

		events, isDir := parseEvents(eventStrs, line, s)
		if events == nil {
			continue
		}

		event := FileEvent{
			Filename: prefix + file,
			Events:   events,
			IsDir:    isDir,
		}

		// Send the file name to the channel
		s.FileEvents <- event
	}

	if err := scanner.Err(); err != nil {
		s.ErrorChan <- err
	}
}

// parseLine parses a line of inotifywait output.
func parseLine(line string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(line))
	return r.Read()
}

// parseEvents parses event strings into EVENT types.
func parseEvents(eventStrs []string, line string, s *Settings) ([]EVENT, bool) {
	var events []EVENT
	isDir := false

	for _, eventStr := range eventStrs {
		if eventStr == FlagIsdir {
			isDir = true
			continue
		}

		eventStr = strings.ToLower(eventStr)
		event, ok := EVENT_MAP_REVERSE[eventStr]
		if !ok {
			s.ErrorChan <- fmt.Errorf("invalid eventStr: <%s>, <%s>", eventStr, line)
			return nil, false
		}
		events = append(events, EVENT(event))
	}

	return events, isDir
}
