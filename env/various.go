package env

import (
	"bufio"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Load environment variables from the given file.
func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		b, a, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		os.Setenv(b, a)
	}
	return nil
}

// MustHave returns the named environment variable. If not set, it writes to
// [os.Stderr] and terminates the program.
func MustHave(name string) string {
	x := os.Getenv(name)
	if x == "" {
		os.Stderr.WriteString("missing " + name)
		os.Exit(1)
	}
	return x
}

// Signal returns a channel signalling termination.
func Signal() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}
