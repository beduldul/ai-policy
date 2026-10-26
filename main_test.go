package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

// The help text drifted from the flag set: --quiet and --version existed but
// were never documented, so `ai-policy --help` never mentioned them. Every flag
// the binary accepts has to appear here.
func TestUsageDocumentsEveryFlag(t *testing.T) {
	var buf bytes.Buffer
	flag.CommandLine.SetOutput(&buf)
	defer flag.CommandLine.SetOutput(os.Stderr)

	usage()
	out := buf.String()

	for _, name := range []string{"path", "json", "quiet", "explain", "version"} {
		if !strings.Contains(out, "--"+name) {
			t.Errorf("usage() does not mention --%s", name)
		}
	}
}
