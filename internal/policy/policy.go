// Package policy decides whether a repository accepts AI-assisted
// contributions, and shows the text it made that decision from.
package policy

import "fmt"

// Verdict is the answer for one repository.
type Verdict int

const (
	// Unknown means no policy file could be read at all.
	Unknown Verdict = iota
	// Allowed means no rule mentioning AI was found.
	Allowed
	// DisclosureRequired means AI work is accepted but must be declared.
	DisclosureRequired
	// Forbidden means the repository rejects AI contributions outright.
	Forbidden
)

func (v Verdict) String() string {
	switch v {
	case Allowed:
		return "ALLOWED"
	case DisclosureRequired:
		return "DISCLOSURE REQUIRED"
	case Forbidden:
		return "FORBIDDEN"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON keeps the wire format stable for --json consumers.
func (v Verdict) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", v.String())), nil
}

// Evidence is one sentence that drove the verdict.
type Evidence struct {
	// Path is repository-relative, e.g. ".github/CONTRIBUTING.md".
	Path string
	// Line is 1-based, pointing at the first line of Text.
	Line int
	// Text is the sentence, whitespace collapsed.
	Text string
	// Kind is what this sentence looks like on its own.
	Kind Verdict
}

// Result is everything the CLI needs to print.
type Result struct {
	Verdict  Verdict
	Evidence []Evidence
	// Checked lists the files that existed and were read.
	Checked []string
	// Missing lists the files that were looked for and not found.
	Missing []string
}

// Forbidden evidence outranks disclosure evidence: one clear "we do not
// accept AI contributions" settles the question no matter what else the
// document says.
func (r *Result) add(e Evidence) {
	if e.Kind == Forbidden {
		r.Verdict = Forbidden
		r.Evidence = append(r.Evidence, e)
		return
	}
	if r.Verdict != Forbidden {
		r.Verdict = DisclosureRequired
	}
	r.Evidence = append(r.Evidence, e)
}
