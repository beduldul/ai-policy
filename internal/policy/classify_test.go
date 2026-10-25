package policy

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		want  Verdict
		lines []int
	}{
		{
			name: "a ban on AI pull requests",
			text: `# Contributing

We welcome bug reports and small fixes.

Please do not use AI tools to generate pull requests. Such contributions will be closed
without review.
`,
			want:  Forbidden,
			lines: []int{5},
		},
		{
			name: "a disclosure rule with trailers and a threat aimed at hiding it",
			text: `## AI Agents

AI-assisted contributions are welcome but must be clearly attributed.

- **Commits**: Include a Co-Authored-By: trailer identifying the AI model used.
- **Pull requests**: State that the content was AI-generated or AI-assisted.

Contributions that obscure their AI origin may be closed.
`,
			want:  DisclosureRequired,
			lines: []int{3, 5, 6, 8},
		},
		{
			name: "disclosure without a threat",
			text: `## AI

AI-assisted contributions are welcome but must be clearly attributed.

- **Commits**: Include a Co-Authored-By: trailer identifying the AI model used.
`,
			want:  DisclosureRequired,
			lines: []int{3, 5},
		},
		{
			name: "no AI rule at all",
			text: `# Contributing

Run go test before opening a pull request. Keep commits small.

See the issue template for details.
`,
			want: Allowed,
		},
		{
			name: "a policy that says there is no AI policy",
			text: `We do not have any AI policy yet. Ask a maintainer if unsure.
`,
			want: Allowed,
		},
		{
			name: "headings alone are not a rule",
			text: `## AI Agents

Nothing else here.
`,
			want: Allowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(".github/CONTRIBUTING.md", tc.text)
			r := Result{Verdict: Allowed}
			for _, e := range got {
				r.add(e)
			}
			if r.Verdict != tc.want {
				t.Fatalf("verdict = %v, want %v (evidence: %+v)", r.Verdict, tc.want, got)
			}
			if len(tc.lines) != len(got) {
				t.Fatalf("got %d evidence lines, want %d: %+v", len(got), len(tc.lines), got)
			}
			for i, want := range tc.lines {
				if got[i].Line != want {
					t.Errorf("evidence[%d].Line = %d, want %d", i, got[i].Line, want)
				}
			}
		})
	}
}

func TestClassifyIgnoresAIMentionedInPassing(t *testing.T) {
	text := `Thanks to the AI team for the review tooling.
`
	if got := Classify("CONTRIBUTING.md", text); len(got) != 0 {
		t.Fatalf("expected no evidence, got %+v", got)
	}
}
