package policy

import (
	"reflect"
	"testing"
)

func TestDiscover(t *testing.T) {
	paths := []string{
		"README.md",
		"docs/contributing.md",    // pallets/click spells it lower case
		".github/CONTRIBUTING.md", // most repositories
		".github/AI_POLICY.md",    // python-attrs/attrs
		".github/PULL_REQUEST_TEMPLATE.md",
		"AGENTS.md",
		"src/main.go",
		".github/workflows/ci.yml",
		"resources/claude.webp", // a picture, not a policy
		"docs/ai-policy.png",
	}

	got := Discover(paths)
	want := []string{
		".github/AI_POLICY.md",    // dedicated policy outranks everything
		".github/CONTRIBUTING.md", // contributing guide next
		"docs/contributing.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"AGENTS.md", // agent instructions last
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestDiscoverPrefersShallowPaths(t *testing.T) {
	got := Discover([]string{
		"docs/deep/nested/CONTRIBUTING.md", // too deep, ignored
		"CONTRIBUTING.md",
		"docs/CONTRIBUTING.md",
	})
	want := []string{"CONTRIBUTING.md", "docs/CONTRIBUTING.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover() = %v, want %v", got, want)
	}
}

func TestLinkedPolicies(t *testing.T) {
	body := `See our [AI policy][llm] and the [contributing guide](../docs/contributing.md).

[llm]: https://github.com/python-attrs/attrs/blob/main/.github/AI_POLICY.md

Also [upstream](https://example.com/AI_POLICY.md), [anchor](#ai), and [email](mailto:x@y.z).
`
	got := LinkedPolicies(".github/CONTRIBUTING.md", body)
	want := []string{".github/AI_POLICY.md", "docs/contributing.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LinkedPolicies() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestLinkedPoliciesIgnoresUnrelatedLinks(t *testing.T) {
	body := "Read [the docs](docs/install.md) and [the api](api/reference.md)."
	if got := LinkedPolicies("CONTRIBUTING.md", body); len(got) != 0 {
		t.Fatalf("expected no policy links, got %v", got)
	}
}

func TestLinkedPoliciesStripsAnchors(t *testing.T) {
	body := "See [the AI section](CONTRIBUTING.md#ai-agents) and [again](CONTRIBUTING.md)."
	got := LinkedPolicies(".github/CONTRIBUTING.md", body)
	want := []string{".github/CONTRIBUTING.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LinkedPolicies() = %v, want %v", got, want)
	}
}

func TestDiscoverUsesReadmeOnlyAsFallback(t *testing.T) {
	if got := Discover([]string{"README.md", "CONTRIBUTING.md"}); !reflect.DeepEqual(got, []string{"CONTRIBUTING.md"}) {
		t.Fatalf("README should be dropped when CONTRIBUTING exists, got %v", got)
	}
	if got := Discover([]string{"README.md"}); !reflect.DeepEqual(got, []string{"README.md"}) {
		t.Fatalf("README should be used when it is all there is, got %v", got)
	}
}
