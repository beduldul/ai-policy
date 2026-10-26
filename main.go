// Command ai-policy reports whether a repository accepts AI-assisted
// contributions, and shows the sentences it decided from.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/beduldul/ai-policy/internal/policy"
)

// Exit codes are separate so a CI step can tell "not allowed" from "needs a
// disclosure line" without parsing the output.
const (
	exitAllowed    = 0
	exitError      = 1
	exitDisclosure = 10
	exitForbidden  = 20
)

func main() {
	var (
		path    = flag.String("path", "", "check a local clone instead of a remote repository")
		asJSON  = flag.Bool("json", false, "print machine readable output")
		quiet   = flag.Bool("quiet", false, "print only the verdict")
		explain = flag.Bool("explain", false, "show lines that mention AI but matched no rule")
		version = flag.Bool("version", false, "print the version and exit")
	)
	flag.Usage = usage
	flag.Parse()

	if *version {
		fmt.Println("ai-policy 0.3.0")
		return
	}

	if (*path == "") == (flag.NArg() == 0) {
		usage()
		os.Exit(exitError)
	}

	var (
		res   *policy.Result
		err   error
		label string
	)

	switch {
	case *path != "":
		abs, absErr := filepath.Abs(*path)
		if absErr != nil {
			fail(absErr)
		}
		label = filepath.Base(abs)
		res, err = policy.FromDir(abs)
	default:
		target := flag.Arg(0)
		owner, repo, splitErr := policy.SplitTarget(target)
		if splitErr != nil {
			fail(splitErr)
		}
		label = owner + "/" + repo
		res, err = policy.NewClient().FromGitHub(owner, repo)
	}
	if err != nil {
		fail(err)
	}

	if *asJSON {
		out, marshalErr := json.MarshalIndent(struct {
			Target string `json:"target"`
			*policy.Result
		}{label, res}, "", "  ")
		if marshalErr != nil {
			fail(marshalErr)
		}
		fmt.Println(string(out))
	} else if *quiet {
		fmt.Println(res.Verdict)
	} else {
		print(res, label, *explain)
	}

	switch res.Verdict {
	case policy.Forbidden:
		os.Exit(exitForbidden)
	case policy.DisclosureRequired:
		os.Exit(exitDisclosure)
	}
}

func print(res *policy.Result, label string, explain bool) {
	fmt.Printf("%s  %s\n", res.Verdict, label)

	if len(res.Evidence) == 0 {
		fmt.Println("\n  No rule mentioning AI was found.")
	} else {
		for _, e := range res.Evidence {
			fmt.Printf("\n  %s:%d\n  %q\n", e.Path, e.Line, e.Text)
		}
	}

	if explain {
		if len(res.NearMisses) == 0 {
			fmt.Println("\n  No lines mentioning AI were skipped.")
		} else {
			fmt.Printf("\n  Skipped %d line(s) that mention AI:\n", len(res.NearMisses))
			for _, n := range res.NearMisses {
				fmt.Printf("\n  %s:%d  (%s)\n  %q\n", n.Path, n.Line, n.Reason, n.Text)
			}
		}
	}

	if len(res.Checked) > 0 {
		fmt.Printf("\n  Read %d file(s): %v\n", len(res.Checked), res.Checked)
	}
	fmt.Println()
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `ai-policy — does this repository accept AI-assisted contributions?

Usage:
  ai-policy <owner/repo>        check a public repository
  ai-policy --path <dir>        check a local clone
  ai-policy --json <target>     machine readable output
  ai-policy --quiet <target>    print only the verdict
  ai-policy --explain <target>  also show lines that mention AI but decided nothing
  ai-policy --version           print the version and exit

Exit codes:
  0   ALLOWED               no AI rule found
  10  DISCLOSURE REQUIRED   AI work is accepted but must be declared
  20  FORBIDDEN             AI contributions are rejected
  1   error
`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ai-policy:", err)
	os.Exit(exitError)
}
