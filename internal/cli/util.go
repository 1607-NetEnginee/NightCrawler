package cli

import "os"

// isTerminal reports whether f is connected to an interactive
// terminal. Implemented via os.File.Stat() + ModeCharDevice rather
// than an ioctl, so we don't pull in golang.org/x/sys just for this
// one probe. Trade-off: a small false-negative class on exotic stdio
// (e.g. some pty multiplexers), but correct for >99% of real terminals
// and all common CI runners.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
