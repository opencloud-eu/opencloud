package command

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/opencloud-eu/opencloud/opencloud/pkg/register"
	"github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/version"
	"github.com/pkg/xattr"
	"github.com/rogpeppe/go-internal/lockedfile"
	"github.com/spf13/cobra"
)

// BenchmarkCommand is the entrypoint for the benchmark commands.
func BenchmarkCommand(cfg *config.Config) *cobra.Command {
	benchCmd := &cobra.Command{
		Use:   "benchmark",
		Short: "cli tools to test low and high level performance",
	}
	benchCmd.AddCommand(BenchmarkClientCommand(cfg), BenchmarkSyscallsCommand(cfg))
	return benchCmd
}

// BenchmarkClientCommand is the entrypoint for the benchmark client command.
func BenchmarkClientCommand(cfg *config.Config) *cobra.Command {
	benchClientCmd := &cobra.Command{
		Use:   "client",
		Short: "Start a client that continuously makes web requests and prints stats. The options mimic curl, but we default to PROPFIND requests.",
		RunE: func(cmd *cobra.Command, args []string) error {
			jobs, err := cmd.Flags().GetInt("jobs")
			if err != nil {
				return err
			}
			insecure, _ := cmd.Flags().GetBool("insecure")
			opt := clientOptions{
				url:      args[0],
				insecure: insecure,
				jobs:     jobs,
				headers:  make(map[string]string),
			}

			if d, _ := cmd.Flags().GetString("data-raw"); d != "" {
				opt.request = "POST"
				opt.headers["Content-Type"] = "application/x-www-form-urlencoded"
				opt.data = []byte(d)
			}

			if d, _ := cmd.Flags().GetString("data"); d != "" {
				opt.request = "POST"
				opt.headers["Content-Type"] = "application/x-www-form-urlencoded"
				if strings.HasPrefix(d, "@") {
					filePath := strings.TrimPrefix(d, "@")
					var data []byte
					var err error

					// read from file or stdin and trim trailing newlines
					if filePath == "-" {
						data, err = os.ReadFile("/dev/stdin")
					} else {
						data, err = os.ReadFile(filePath)
					}
					if err != nil {
						log.Fatal(errors.New("could not read data from file '" + filePath + "': " + err.Error()))
					}

					// clean byte array similar to curl's --data parameter
					// It removes leading/trailing whitespace and converts line breaks to spaces

					// Trim leading and trailing whitespace
					data = bytes.TrimSpace(data)

					// Replace newlines and carriage returns with spaces
					data = bytes.ReplaceAll(data, []byte("\r\n"), []byte(" "))
					data = bytes.ReplaceAll(data, []byte("\n"), []byte(" "))
					data = bytes.ReplaceAll(data, []byte("\r"), []byte(" "))

					// Replace multiple spaces with single space
					for bytes.Contains(data, []byte("  ")) {
						data = bytes.ReplaceAll(data, []byte("  "), []byte(" "))
					}

					opt.data = data
				} else {
					opt.data = []byte(d)
				}
			}

			if d, _ := cmd.Flags().GetString("data-binary"); d != "" {
				opt.request = "POST"
				opt.headers["Content-Type"] = "application/x-www-form-urlencoded"
				if strings.HasPrefix(d, "@") {
					filePath := strings.TrimPrefix(d, "@")
					var data []byte
					var err error
					if filePath == "-" {
						data, err = os.ReadFile("/dev/stdin")
					} else {
						data, err = os.ReadFile(filePath)
					}
					if err != nil {
						log.Fatal(errors.New("could not read data from file '" + filePath + "': " + err.Error()))
					}
					opt.data = data
				} else {
					opt.data = []byte(d)
				}
			}

			// override method if specified
			if request, _ := cmd.Flags().GetString("request"); request != "" {
				opt.request = request
			}

			if opt.url == "" {
				log.Fatal(errors.New("no URL specified"))
			}

			headersSlice, err := cmd.Flags().GetStringSlice("header")
			if err != nil {
				return err
			}
			for _, h := range headersSlice {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					log.Fatal(errors.New("invalid header '" + h + "'"))
				}
				opt.headers[parts[0]] = strings.TrimSpace(parts[1])
			}

			rate, _ := cmd.Flags().GetString("rate")
			if rate != "" {
				parts := strings.SplitN(rate, "/", 2)
				num, err := strconv.Atoi(parts[0])
				if err != nil {
					fmt.Println(err)
				}
				unit := time.Hour // default
				if len(parts) == 2 {
					switch parts[1] {
					case "s":
						unit = time.Second
					case "m":
						unit = time.Minute
					case "d":
						unit = time.Hour * 24
					default:
						log.Fatal(errors.New("unsupported rate unit. Use s, m, h or d"))
					}
				}
				opt.rateDelay = unit / time.Duration(num)
			}

			user, _ := cmd.Flags().GetString("user")
			opt.auth = func() string {
				return "Basic " + base64.StdEncoding.EncodeToString([]byte(user))
			}

			btc, _ := cmd.Flags().GetString("bearer-token-command")
			if btc != "" {
				parts := strings.SplitN(btc, " ", 2)
				var cmd *exec.Cmd
				opt.auth = func() string {
					if len(parts) > 1 {
						cmd = exec.Command(parts[0], parts[1])
					} else {
						cmd = exec.Command(parts[0])
					}
					output, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Println(err)
					}
					return "Bearer " + string(output)
				}
			}

			every, err := cmd.Flags().GetInt("every")
			if err != nil {
				return err
			}
			if every != 0 {
				opt.ticker = time.NewTicker(time.Second * time.Duration(every))
				defer opt.ticker.Stop()
			}

			// Set up signal handling for Ctrl+C
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal, shutting down...")
				cancel()
			}()
			return client(ctx, opt)

		},
	}

	// flags mimicing curl
	benchClientCmd.Flags().StringP("request", "X", "PROPFIND", "Specifies a custom request method to use when communicating with the HTTP server.")
	benchClientCmd.Flags().StringP("user", "u", "admin:admin", "Specify the user name and password to use for server authentication.")
	benchClientCmd.Flags().BoolP("insecure", "k", false, "Skip the TLS verification step and proceed without checking.")
	benchClientCmd.Flags().StringP("data", "d", "", "Sends the specified data in a POST request to the HTTP server, in the same way that a browser does when a user has filled in an HTML form and presses the submit button. If you start the data with the letter @, the rest should be a file name to read the data from, or - if you want to read the data from stdin. When -d, --data is told to read from a file like that, carriage returns and newlines are stripped out. If you do not want the @ character to have a special interpretation use --data-raw instead.")
	benchClientCmd.Flags().StringP("data-raw", "", "", "Sends the specified data in a request to the HTTP server.")
	benchClientCmd.Flags().StringP("data-binary", "", "", "This posts data exactly as specified with no extra processing whatsoever. If you start the data with the letter @, the rest should be a file name to read the data from, or - if you want to read the data from stdin.")
	benchClientCmd.Flags().StringSliceP("headers", "H", []string{}, "Extra header to include in information sent.")
	benchClientCmd.Flags().String("rate", "", "Specify the maximum transfer frequency you allow a client to use - in number of transfer starts per time unit (sometimes called request rate). The request rate is provided as \"N/U\" where N is an integer number and U is a time unit. Supported units are 's' (second), 'm' (minute), 'h' (hour) and 'd' /(day, as in a 24 hour unit). The default time unit, if no \"/U\" is provided, is number of transfers per hour.")

	// other flags
	benchClientCmd.Flags().IntP("jobs", "j", 1, "Number of parallel clients to start. Defaults to 1.")
	benchClientCmd.Flags().Int("every", 0, "Aggregate stats every time this amount of seconds has passed.")
	benchClientCmd.Flags().String("bearer-token-command", "", "Command to execute for a bearer token, e.g. 'oidc-token opencloud'. When set, disables basic auth.")

	return benchClientCmd
}

type clientOptions struct {
	request   string
	url       string
	auth      func() string
	insecure  bool
	headers   map[string]string
	rateDelay time.Duration
	data      []byte
	ticker    *time.Ticker
	jobs      int
}

func client(ctx context.Context, o clientOptions) error {
	type stat struct {
		job      int
		duration time.Duration
		status   int
	}
	stats := make(chan stat)
	for i := 0; i < o.jobs; i++ {
		go func(i int) {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: o.insecure,
				},
			}
			client := &http.Client{Transport: tr}

			cookies := map[string]*http.Cookie{}
			for {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}

				req, err := http.NewRequest(o.request, o.url, bytes.NewReader(o.data))
				if err != nil {
					log.Printf("client %d: could not create request: %s\n", i, err)
					return
				}
				req.Header.Set("Authorization", strings.TrimSpace(o.auth()))
				for k, v := range o.headers {
					req.Header.Set(k, v)
				}
				for _, cookie := range cookies {
					req.AddCookie(cookie)
				}

				start := time.Now()
				res, err := client.Do(req)
				duration := -time.Until(start)
				if err != nil {
					// Check if error is due to context cancellation
					if ctx.Err() != nil {
						return
					}
					log.Printf("client %d: could not create request: %s\n", i, err)
					time.Sleep(time.Second)
				} else {
					res.Body.Close()
					select {
					case stats <- stat{
						job:      i,
						duration: duration,
						status:   res.StatusCode,
					}:
					case <-ctx.Done():
						return
					}
					for _, c := range res.Cookies() {
						cookies[c.Name] = c
					}
				}
				// Sleep with context awareness
				if o.rateDelay > duration {
					select {
					case <-time.After(o.rateDelay - duration):
					case <-ctx.Done():
						return
					}
				}
			}
		}(i)
	}

	numRequests := 0
	if o.ticker == nil {
		// no ticker, just write every request
		for {
			select {
			case stat := <-stats:
				numRequests++
				fmt.Printf("req %d took %v and returned status %d\n", numRequests, stat.duration, stat.status)
			case <-ctx.Done():
				fmt.Println("\nShutting down...")
				return nil
			}
		}
	}

	var duration time.Duration
	for {
		select {
		case stat := <-stats:
			numRequests++
			duration += stat.duration
		case <-o.ticker.C:
			if numRequests > 0 {
				fmt.Printf("%d req at %v/req\n", numRequests, duration/time.Duration(numRequests))
				numRequests = 0
				duration = 0
			}
		case <-ctx.Done():
			if numRequests > 0 {
				fmt.Printf("\n%d req at %v/req\n", numRequests, duration/time.Duration(numRequests))
			}
			fmt.Println("Shutting down...")
			return nil
		}
	}

}

// BenchmarkSyscallsCommand is the entrypoint for the benchmark syscalls command.
func BenchmarkSyscallsCommand(cfg *config.Config) *cobra.Command {
	benchSysCallCmd := &cobra.Command{
		Use:   "syscalls",
		Short: "test the performance of syscalls",
		RunE: func(cmd *cobra.Command, args []string) error {

			path, _ := cmd.Flags().GetString("path")
			if path == "" {
				f, err := os.CreateTemp("", "opencloud-bench-temp-")
				if err != nil {
					log.Fatal(err)
				}
				path = f.Name()
				f.Close()
				defer os.Remove(path)
			}

			iterations, err := cmd.Flags().GetInt("iterations")
			if err != nil {
				return err
			}
			return benchmark(iterations, path)
		},
	}
	benchSysCallCmd.Flags().String("path", "", "Path to test")
	benchSysCallCmd.Flags().Int("iterations", 100, "Number of iterations to execute")
	return benchSysCallCmd
}

func benchmark(iterations int, path string) error {
	tests := map[string]func() error{
		"lockedfile open(wo,c,t) close": func() error {
			for i := 0; i < iterations; i++ {
				lockedFile, err := lockedfile.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
				if err != nil {
					return err
				}
				lockedFile.Close()
			}
			return nil
		},
		"stat": func() error {
			for i := 0; i < iterations; i++ {
				_, err := os.Stat(path)
				if err != nil {
					return err
				}
			}
			return nil
		},
		"fopen(ro) close": func() error {
			for i := 0; i < iterations; i++ {
				h, err := os.OpenFile(path, os.O_RDONLY, 0600)
				if err != nil {
					return err
				}
				h.Close()
			}
			return nil
		},
		"fopen(wo,t) write close": func() error {
			for i := 0; i < iterations; i++ {
				h, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0600)
				if err != nil {
					return err
				}
				_, err = h.WriteString("1234567890")
				if err != nil {
					h.Close()
					return err
				}
				h.Close()
			}
			return nil
		},
		"fopen(ro) read close": func() error {
			for i := 0; i < iterations; i++ {
				bytes := make([]byte, 0, 10)
				h, err := os.OpenFile(path, os.O_RDONLY, 0600)
				if err != nil {
					return err
				}
				_, err = h.Read(bytes)
				if err != nil {
					h.Close()
					return err
				}
				h.Close()
			}
			return nil
		},
		"xattr-set": func() error {
			for i := 0; i < iterations; i++ {
				err := xattr.Set(path, "user.test", []byte("123456"))
				if err != nil {
					return err
				}
			}
			return nil
		},
		"xattr-get": func() error {
			for i := 0; i < iterations; i++ {
				_, err := xattr.Get(path, "user.test")
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	fmt.Println("Version: " + version.GetString())
	fmt.Printf("Compiled: %s\n", version.Compiled())
	fmt.Printf("Path: %s\n", path)
	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Println("")

	cfg := tablewriter.Config{
		Header: tw.CellConfig{
			Formatting: tw.CellFormatting{
				AutoFormat: tw.Off,
			},
		},
		Row: tw.CellConfig{
			ColumnAligns: []tw.Align{
				tw.AlignLeft,
				tw.AlignRight,
				tw.AlignRight,
				tw.AlignRight,
			},
		},
	}

	table := tablewriter.NewTable(os.Stdout, tablewriter.WithConfig(cfg))
	table.Header([]string{"Test", "Iterations", "dur/it", "total"})
	for _, t := range []string{"lockedfile open(wo,c,t) close", "stat", "fopen(wo,t) write close", "fopen(ro) close", "fopen(ro) read close", "xattr-set", "xattr-get"} {
		start := time.Now()
		err := tests[t]()
		end := time.Now()
		delta := end.Sub(start)
		if err != nil {
			table.Append([]string{t, fmt.Sprintf("%d", iterations), err.Error(), err.Error()})
		} else {
			table.Append([]string{t, fmt.Sprintf("%d", iterations), strconv.Itoa(int(delta.Nanoseconds())/iterations) + "ns", delta.String()})
		}
	}
	table.Render()
	return nil
}

func init() {
	register.AddCommand(BenchmarkCommand)
}
