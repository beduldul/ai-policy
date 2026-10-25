# ai-policy

Know whether a repository accepts AI-assisted contributions **before** you open
the pull request.

Repositories disagree, and the rules hide in different places. `pallets` bans AI
contributions outright — in `pallets/.github`, not in the repository you are
looking at. `python-attrs/attrs` keeps its rules in `.github/AI_POLICY.md` and
only links to it from `CONTRIBUTING.md`. `go-playground/validator` accepts AI
work but requires a `Co-Authored-By:` trailer. `stablyai/orca` requires a code
review summary from your agent.

`ai-policy` reads those files, finds the sentences that state a rule, and tells
you which of the three situations you are in — with the exact line it read.

## Install

```sh
go install github.com/beduldul/ai-policy@latest
```

Or build from a clone:

```sh
git clone https://github.com/beduldul/ai-policy
cd ai-policy
go build -o ai-policy .
```

## Usage

```sh
ai-policy pallets/click
```

```
FORBIDDEN  pallets/click

  pallets/.github:CONTRIBUTING.md:22
  "AI-generated PRs and issues are closed on sight, without review or discussion."

  Read 4 file(s): [docs/contributing.md .github/pull_request_template.md README.md CONTRIBUTING.md]
```

```sh
ai-policy go-playground/validator
```

```
DISCLOSURE REQUIRED  go-playground/validator

  .github/CONTRIBUTING.md:13
  "AI-assisted contributions are welcome but must be clearly attributed. If you are an AI agent or using an AI agent to contribute:"

  .github/CONTRIBUTING.md:15
  "- **Commits**: Include a `Co-Authored-By:` trailer identifying the AI model used."

  .github/CONTRIBUTING.md:16
  "- **Pull requests and issues**: State that the content was AI-generated or AI-assisted in the description."
```

Check a local clone instead of the network:

```sh
ai-policy --path .
```

Machine readable, for a script:

```sh
ai-policy --json pallets/click
```

Just the verdict:

```sh
ai-policy --quiet pallets/click
```

## Exit codes

| Code | Verdict | Meaning |
|---:|---|---|
| `0` | `ALLOWED` | No rule mentioning AI was found |
| `10` | `DISCLOSURE REQUIRED` | AI work is accepted, but must be declared |
| `20` | `FORBIDDEN` | AI contributions are rejected |
| `1` | — | Error (bad argument, network failure, private repository) |

The codes are distinct so a CI step can gate on "forbidden" without also
blocking "needs a disclosure line".

## Where it looks

In priority order, per repository:

1. A dedicated AI policy — `.github/AI_POLICY.md`, `AI_POLICY.md`, `LLM_POLICY.md`
2. `CONTRIBUTING.md` (also `docs/`, `.github/`, any capitalisation)
3. `.github/PULL_REQUEST_TEMPLATE.md`
4. `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, `.github/copilot-instructions.md`
5. `README.md`, only if nothing above exists

Policy documents linked from any of those are followed one level deep, including
reference-style markdown links and `github.com/.../blob/...` URLs. The
organisation's `.github` repository is merged in as well, because default
community health files apply to every repository the organisation owns.

## How it decides

No model is called at runtime. Each line of each policy document is checked for
three things:

- a term for AI (`AI`, `LLM`, `Copilot`, `Claude`, `generative`, …)
- a refusal cue (`do not`, `not accept`, `closed`, `prohibited`, `banned`, …)
  together with a word for the thing being refused (`contribution`, `PR`,
  `commit`, `patch`, …)
- a disclosure cue (`disclose`, `attribute`, `Co-Authored-By`, `state that`,
  `include`, …)

A line with a refusal cue and a contribution noun is `FORBIDDEN`. A line that
asks for a label is `DISCLOSURE REQUIRED`. A refusal aimed at *hidden* AI use —
"contributions that obscure their AI origin may be closed" — is treated as a
disclosure rule, not a ban. The strictest verdict in the repository wins.

Every verdict prints the sentence it came from, so you can check the reasoning
yourself. That is the point: the tool is a fast way to find the paragraph, not a
replacement for reading it.

## Limitations

- **Public repositories only.** Private repositories need an authenticated API
  call, which this version does not do. `--path` works on a local clone of
  anything.
- **English only.** The cue lists are English phrases.
- **Heuristics, not comprehension.** A policy phrased in a way none of the cues
  cover will come back `ALLOWED`. When a repository mentions AI at all, the
  matched lines are printed, so a wrong verdict is visible rather than silent.
- **One level of links.** A policy linked from a linked policy is not followed.

## Tests

```sh
go test ./...
```

The classifier is tested against phrasings taken from real repositories,
including the two that are easy to get wrong: a policy that says there is no AI
policy, and a disclosure rule whose only threat is aimed at hiding AI use.

## License

MIT
