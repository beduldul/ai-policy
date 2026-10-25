package policy

import (
	"regexp"
	"strings"
)

// aiTerm matches the words people use when they write an AI rule. The
// boundaries matter: "AI" must not fire inside "said" or "maintain".
var aiTerm = regexp.MustCompile(`(?i)\b(ai|artificial intelligence|llms?|chatgpt|copilot|claude|gpt-?[0-9a-z]*|language models?|machine[- ]generated|ai[- ]assisted|ai[- ]generated|generative)\b`)

// forbiddenCue is language that refuses a contribution. Each phrase is
// something a maintainer writes when the answer is no.
var forbiddenCue = regexp.MustCompile(`(?i)(do not|don't|does not|never|not accept|not allowed|closed|prohibited|banned|forbidden|reject|decline|not welcome|unwelcome|will not be reviewed)`)

// disclosureCue is language that accepts a contribution but demands a label.
var disclosureCue = regexp.MustCompile(`(?i)(disclos|attribut|co-authored-by|state (that|it|whether)|must be clearly|declare|mention (that|it)|note (that|in the)|identify|include|summar|provide)`)

// contributionNoun keeps the forbidden check honest. "We do not have an AI
// policy" also contains "do not" and "AI", and that sentence means the
// opposite of a ban, so a refusal only counts when it is about code.
var contributionNoun = regexp.MustCompile(`(?i)(contribution|pull request|\bprs?\b|\bcode\b|commit|patch|issue|review|change)`)

// Classify reads a policy document and returns every sentence that looks
// like an AI rule. It is deliberately line based: markdown keeps one rule per
// line, and a line number is what a reader needs to go check the source.
func Classify(path, text string) []Evidence {
	var out []Evidence
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || len(line) < 12 {
			continue
		}
		if !aiTerm.MatchString(line) {
			continue
		}
		// Headings and table rows carry no rule of their own.
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") {
			continue
		}
		kind := classifyLine(line)
		if kind == Allowed {
			continue
		}
		out = append(out, Evidence{
			Path: path,
			Line: i + 1,
			Text: collapse(line),
			Kind: kind,
		})
	}
	return out
}

// obscured matches a refusal aimed at hidden AI use rather than at AI use
// itself: "contributions that obscure their AI origin may be closed" demands a
// disclosure, it does not ban AI.
var obscured = regexp.MustCompile(`(?i)(obscur|undisclos|without disclos|conceal|hide|hidden)`)

func classifyLine(line string) Verdict {
	if forbiddenCue.MatchString(line) && contributionNoun.MatchString(line) && !obscured.MatchString(line) {
		// "AI-assisted contributions are welcome but must be clearly
		// attributed" contains no refusal cue, so it never reaches here.
		return Forbidden
	}
	if disclosureCue.MatchString(line) || obscured.MatchString(line) {
		return DisclosureRequired
	}
	return Allowed
}

// collapse folds whitespace so a long list item stays one printable line.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
