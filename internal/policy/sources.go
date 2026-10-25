package policy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// policyName matches a dedicated AI policy document wherever it lives.
var policyName = regexp.MustCompile(`(?i)^(ai|llm)[-_ ]?(policy|usage|rules)`)

// linkTarget pulls the destination out of a markdown link.
var linkTarget = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// linkDefinition matches a reference-style definition, which is how
// python-attrs/attrs links its policy: [llm]: https://github.com/...
var linkDefinition = regexp.MustCompile(`(?m)^\[[^\]]+\]:\s*(\S+)`)

// githubBlob rewrites a browser URL into a repository path.
var githubBlob = regexp.MustCompile(`^https?://github\.com/[^/]+/[^/]+/(?:blob|tree)/[^/]+/(.+)$`)

// Discover picks the policy files out of a repository file listing. Order is
// the whole point: a dedicated AI policy outranks a general contributing
// guide, which outranks an agent instruction file, which outranks anything
// else that merely mentions AI.
func Discover(paths []string) []string {
	type scored struct {
		path string
		rank int
	}
	var picked []scored
	seen := map[string]bool{}

	for _, p := range paths {
		r := rank(p)
		if r < 0 || seen[p] {
			continue
		}
		seen[p] = true
		picked = append(picked, scored{p, r})
	}

	sort.SliceStable(picked, func(i, j int) bool {
		if picked[i].rank != picked[j].rank {
			return picked[i].rank < picked[j].rank
		}
		di, dj := strings.Count(picked[i].path, "/"), strings.Count(picked[j].path, "/")
		if di != dj {
			return di < dj
		}
		return picked[i].path < picked[j].path
	})

	out := make([]string, len(picked))
	for i, p := range picked {
		out[i] = p.path
	}
	return out
}

// rank orders a path by how authoritative it is as a statement of policy.
// -1 means the file is not a policy document at all.
func rank(p string) int {
	base := strings.ToLower(path.Base(p))
	stem := strings.TrimSuffix(base, path.Ext(base))

	// Anything buried deeper than a conventional docs folder is source code
	// that happens to share a name: tests/AGENTS.md, resources/claude.webp,
	// src/main/.../pull-request-template.ts.
	depth := strings.Count(p, "/")
	if depth > 2 {
		return -1
	}

	switch {
	case policyName.MatchString(stem):
		return 0
	case stem == "contributing":
		return 1
	case stem == "pull_request_template" || stem == "pull-request-template":
		return 2
	case stem == "agents" || stem == "claude" || stem == ".cursorrules" || stem == "copilot-instructions":
		if depth > 1 {
			return -1
		}
		return 3
	case stem == "readme" && strings.Count(p, "/") == 0:
		return 5
	}
	return -1
}

// LinkedPolicies returns the repository-relative paths of policy documents
// that body links to. python-attrs/attrs keeps its rules in .github/AI_POLICY.md
// and only links to it from CONTRIBUTING.md, so a single file is not enough.
func LinkedPolicies(fromPath, body string) []string {
	var out []string
	dir := path.Dir(fromPath)
	seen := map[string]bool{}

	targets := []string{}
	for _, m := range linkTarget.FindAllStringSubmatch(body, -1) {
		targets = append(targets, m[1])
	}
	for _, m := range linkDefinition.FindAllStringSubmatch(body, -1) {
		targets = append(targets, m[1])
	}

	for _, target := range targets {
		// A github.com/.../blob/... URL is already relative to the
		// repository root, so it must not be joined to the current file's
		// directory a second time.
		fromRoot := false
		if g := githubBlob.FindStringSubmatch(target); g != nil {
			target, fromRoot = g[1], true
		}
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
			strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		target = strings.TrimPrefix(target, "./")
		resolved := path.Clean(target)
		if !fromRoot {
			resolved = path.Clean(path.Join(dir, target))
		}
		if !looksLikePolicy(resolved) || seen[resolved] {
			continue
		}
		seen[resolved] = true
		out = append(out, resolved)
	}
	// A dedicated AI policy is fetched before a general guide.
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

func looksLikePolicy(p string) bool { return rank(p) >= 0 }

// FromDir reads the policy files of a local clone.
func FromDir(root string) (*Result, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	var paths []string
	walkErr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable directories are not fatal
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "target", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.Count(rel, "/") > 3 {
			return nil
		}
		if looksLikePolicy(rel) {
			paths = append(paths, rel)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	res, ok := read(Discover(paths), func(p string) (string, error) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		return string(b), err
	})
	if !ok {
		return &Result{Verdict: Unknown}, nil
	}
	return res, nil
}

// Client fetches policy files over the network.
type Client struct {
	// RawURL is the raw content host. Tests point it at a local server.
	RawURL string
	// APIURL is the REST host, used only to list the repository tree.
	APIURL string
	// Token is optional; public repositories work without one.
	Token string
	HTTP  *http.Client
}

// NewClient returns a client with sane timeouts.
func NewClient() *Client {
	c := &Client{
		RawURL: "https://raw.githubusercontent.com",
		APIURL: "https://api.github.com",
		Token:  os.Getenv("GITHUB_TOKEN"),
		HTTP:   &http.Client{Timeout: 20 * time.Second},
	}
	return c
}

// FromGitHub reads the policy files of a public repository at its default
// branch, following links to dedicated AI policies. It then merges in the
// organisation's .github repository, because that is where pallets keeps the
// rules that apply to every repository it owns.
func (c *Client) FromGitHub(owner, repo string) (*Result, error) {
	res := &Result{Verdict: Allowed}
	found := false

	if paths, err := c.tree(owner, repo); err == nil {
		if r, ok := read(Discover(paths), func(p string) (string, error) {
			return c.raw(owner, repo, p)
		}); ok {
			res, found = r, true
		}
	}

	if paths, err := c.tree(owner, ".github"); err == nil {
		if r, ok := read(Discover(paths), func(p string) (string, error) {
			return c.raw(owner, ".github", p)
		}); ok {
			if found {
				merge(res, r, owner+"/.github")
			} else {
				res, found = r, true
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no policy file found in %s/%s (is the repository public?)", owner, repo)
	}
	return res, nil
}

// merge folds the organisation-wide rules into a repository result. Evidence
// from the shared repository is prefixed so the reader can tell where a rule
// came from.
func merge(dst, src *Result, label string) {
	for _, e := range src.Evidence {
		e.Path = label + ":" + e.Path
		dst.add(e)
	}
	dst.Checked = append(dst.Checked, src.Checked...)
}

// read loads the given paths and then any policy document they link to.
func read(paths []string, fetch func(string) (string, error)) (*Result, bool) {
	if len(paths) == 0 {
		return nil, false
	}
	res := &Result{Verdict: Allowed}
	queue := append([]string(nil), paths...)
	seen := map[string]bool{}

	for i := 0; i < len(queue) && len(res.Checked) < 12; i++ {
		p := queue[i]
		if seen[p] {
			continue
		}
		seen[p] = true

		body, err := fetch(p)
		if err != nil {
			res.Missing = append(res.Missing, p)
			continue
		}
		res.Checked = append(res.Checked, p)
		apply(res, p, body)

		for _, link := range LinkedPolicies(p, body) {
			if !seen[link] {
				queue = append(queue, link)
			}
		}
	}

	if len(res.Checked) == 0 {
		return nil, false
	}
	return res, true
}

func (c *Client) tree(owner, repo string) ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/trees/HEAD?recursive=1", c.APIURL, owner, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listing %s/%s: %s", owner, repo, resp.Status)
	}

	var payload struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Tree))
	for _, t := range payload.Tree {
		if t.Type == "blob" {
			out = append(out, t.Path)
		}
	}
	return out, nil
}

func (c *Client) raw(owner, repo, p string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/HEAD/%s", c.RawURL, owner, repo, p)
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", p, resp.Status)
	}
	// Policy docs are small; 1 MiB stops a hostile server from streaming.
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// apply feeds one document into the result.
func apply(res *Result, p, body string) {
	for _, e := range Classify(p, body) {
		res.add(e)
	}
}

// SplitTarget turns "owner/repo" or a github.com URL into its two halves.
func SplitTarget(s string) (owner, repo string, err error) {
	s = strings.TrimSuffix(strings.TrimPrefix(s, "https://github.com/"), "/")
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected owner/repo, got %q", s)
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}
