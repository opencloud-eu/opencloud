package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// allowedCommands defines the list of commands that can be benchmarked
var allowedCommands = map[string]bool{
	"ls":     true,
	"echo":   true,
	"cat":    true,
	"grep":   true,
	"find":   true,
	"wc":     true,
	"sort":   true,
	"uniq":   true,
	"head":   true,
	"tail":   true,
	"date":   true,
	"sleep":  true,
}

func Benchmark() *cli.Command {
	return &cli.Command{
		Name:      "benchmark",
		Usage:     "Run benchmark on allowed commands",
		ArgsUsage: "<command> [args...]",
		Action:    runBenchmark,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "runs",
				Value: 1,
				Usage: "Number of runs",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output file",
			},
		},
	}
}

func runBenchmark(c *cli.Context) error {
	runs := c.Int("runs")
	output := c.String("output")

	if c.NArg() == 0 {
		return errors.New("missing command to benchmark")
	}
	
	args := c.Args().Slice()
	command := args[0]
	commandArgs := args[1:]

	// Security: Validate command against allowlist
	commandBase := filepath.Base(command)
	if !allowedCommands[commandBase] {
		return fmt.Errorf("command '%s' is not allowed for benchmarking. Allowed commands: %v", 
			command, getAllowedCommandsList())
	}

	var results []time.Duration
	for i := 0; i < runs; i++ {
		start := time.Now()
		cmd := exec.Command(commandBase, commandArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
		results = append(results, time.Since(start))
	}

	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return err
		}
		defer f.Close()
		for i, result := range results {
			fmt.Fprintf(f, "Run %d: %s\n", i+1, result)
		}
	}

	for i, result := range results {
		fmt.Printf("Run %d: %s\n", i+1, result)
	}

	return nil
}

func getAllowedCommandsList() string {
	var commands []string
	for cmd := range allowedCommands {
		commands = append(commands, cmd)
	}
	return strings.Join(commands, ", ")
}
