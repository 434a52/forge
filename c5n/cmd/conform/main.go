// Command conform runs a language-neutral vector dataset through one runner per language
// and reports where they disagree with what the dataset expects.
//
// The split is deliberate. A runner executes cases and reports what it got, nothing more;
// the comparison lives here. That is what lets the same driver be pointed at somebody
// else's implementation of the same contract and audit it — there is one comparison
// implementation, and no runner grades its own work.
//
//	conform -vectors f8n/vectors/percentage.json \
//	        -runner "csharp=dotnet run --project f8n/dotnet/RunVector --no-build --" \
//	        -runner "ts=node f8n/ts/dist/run-vector.js"
//
// With -capture <name>, the named runner's output is written back into the dataset as the
// expected values. That is how the bulk of a dataset is produced: an implementation
// proposes the values once, and from that moment they are frozen — a later change in
// behaviour shows up as a divergence, and a human decides which reading was right. Edge
// cases are chosen for coverage the same way; what makes a captured value trustworthy is
// that the *other* implementation was written from the rule and not from the dataset.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// VectorFile mirrors the dataset's shape. Field order here is the field order written back
// on capture, so the file stays stable under repeated captures.
type VectorFile struct {
	Version int     `json:"version"`
	Subject string  `json:"subject"`
	Groups  []Group `json:"groups"`
}

type Group struct {
	Rule  string `json:"rule"`
	Note  string `json:"note,omitempty"`
	Cases []Case `json:"cases"`
}

// Case is one input and what it is expected to produce. Exactly one of Out and Error is
// recorded once captured: Error is a bare true rather than a message, because the message
// is not part of the contract — each language words its own, and pinning the text would
// make a reworded error a breaking change.
type Case struct {
	ID    string   `json:"id"`
	Op    string   `json:"op"`
	In    []string `json:"in"`
	Note  string   `json:"note,omitempty"`
	Out   *string  `json:"out,omitempty"`
	Error *bool    `json:"error,omitempty"`
}

// Result is what a runner reports for one case.
type Result struct {
	ID    string  `json:"id"`
	Out   *string `json:"out,omitempty"`
	Error *string `json:"error,omitempty"`
}

type runnerFlags []string

func (r *runnerFlags) String() string { return strings.Join(*r, ", ") }

func (r *runnerFlags) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func main() {
	var runners runnerFlags
	vectors := flag.String("vectors", "", "path to the vector dataset")
	capture := flag.String("capture", "", "write this runner's results back as the expected values")
	flag.Var(&runners, "runner", `a runner as "name=command" (repeatable); the dataset path is appended`)
	flag.Parse()

	if *vectors == "" || len(runners) == 0 {
		fmt.Fprintln(os.Stderr, "usage: conform -vectors <file> -runner \"name=command\" [-runner …] [-capture <name>]")
		os.Exit(2)
	}

	if err := run(*vectors, runners, *capture); err != nil {
		fmt.Fprintln(os.Stderr, "conform: "+err.Error())
		os.Exit(1)
	}
}

func run(vectorsPath string, runners []string, capture string) error {
	file, err := loadVectors(vectorsPath)
	if err != nil {
		return err
	}
	cases := allCases(file)

	results := map[string][]Result{}
	for _, spec := range runners {
		name, command, ok := strings.Cut(spec, "=")
		if !ok {
			return fmt.Errorf("runner %q is not in name=command form", spec)
		}
		out, err := invoke(command, vectorsPath)
		if err != nil {
			return fmt.Errorf("runner %s: %w", name, err)
		}
		if len(out) != len(cases) {
			return fmt.Errorf("runner %s reported %d results for %d cases", name, len(out), len(cases))
		}
		results[name] = out
	}

	if capture != "" {
		captured, ok := results[capture]
		if !ok {
			return fmt.Errorf("no runner named %q to capture from", capture)
		}
		return captureInto(file, vectorsPath, captured)
	}
	return compare(vectorsPath, cases, results, runners)
}

// invoke runs one runner with the dataset path appended and decodes its results.
func invoke(command, vectorsPath string) ([]Result, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, errors.New("empty command")
	}
	parts = append(parts, vectorsPath)

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", command, err)
	}

	var results []Result
	if err := json.Unmarshal(stdout, &results); err != nil {
		return nil, fmt.Errorf("%s: output is not a result array: %w", command, err)
	}
	return results, nil
}

// compare checks every runner against the recorded expectation, reporting each divergence
// with what was wanted and what arrived.
func compare(vectorsPath string, cases []*Case, results map[string][]Result, runners []string) error {
	fmt.Printf("%s — %d cases\n", vectorsPath, len(cases))

	var uncaptured []string
	for _, c := range cases {
		if c.Out == nil && c.Error == nil {
			uncaptured = append(uncaptured, c.ID)
		}
	}
	if len(uncaptured) > 0 {
		sort.Strings(uncaptured)
		return fmt.Errorf("%d case(s) have no expected value yet (%s%s)\nrun again with -capture <runner> to record them",
			len(uncaptured), strings.Join(uncaptured[:min(3, len(uncaptured))], ", "),
			map[bool]string{true: ", …", false: ""}[len(uncaptured) > 3])
	}

	failed := 0
	for _, spec := range runners {
		name, _, _ := strings.Cut(spec, "=")
		var divergences []string
		for i, c := range cases {
			if problem := check(c, results[name][i]); problem != "" {
				divergences = append(divergences, fmt.Sprintf("    %s: %s", c.ID, problem))
			}
		}
		fmt.Printf("  %-8s %d/%d ok\n", name, len(cases)-len(divergences), len(cases))
		for _, d := range divergences {
			fmt.Println(d)
		}
		failed += len(divergences)
	}

	if failed > 0 {
		return fmt.Errorf("%d divergence(s) from the dataset", failed)
	}
	return nil
}

// check compares one recorded expectation against one runner's result.
func check(c *Case, got Result) string {
	if c.Error != nil && *c.Error {
		if got.Error == nil {
			return fmt.Sprintf("expected a rejection, got %q", derefString(got.Out))
		}
		return "" // any message is fine — the wording is not part of the contract
	}
	if got.Error != nil {
		return fmt.Sprintf("expected %q, was rejected: %s", derefString(c.Out), *got.Error)
	}
	if derefString(got.Out) != derefString(c.Out) {
		return fmt.Sprintf("expected %q, got %q", derefString(c.Out), derefString(got.Out))
	}
	return ""
}

// captureInto records a runner's results as the dataset's expected values, reporting what
// changed so a capture is never a silent rewrite of an existing expectation.
func captureInto(file *VectorFile, path string, results []Result) error {
	cases := allCases(file)
	added, changed := 0, 0

	for i, c := range cases {
		got := results[i]
		wasSet := c.Out != nil || c.Error != nil
		before := describe(c)

		if got.Error != nil {
			yes := true
			c.Out, c.Error = nil, &yes
		} else {
			value := derefString(got.Out)
			c.Out, c.Error = &value, nil
		}

		switch {
		case !wasSet:
			added++
		case before != describe(c):
			changed++
			fmt.Printf("  changed %s: %s → %s\n", c.ID, before, describe(c))
		}
	}

	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("%s — %d captured, %d changed\n", path, added, changed)
	if changed > 0 {
		fmt.Println("a changed expectation means behaviour moved: check which reading is right before committing")
	}
	return nil
}

func describe(c *Case) string {
	if c.Error != nil && *c.Error {
		return "rejected"
	}
	if c.Out != nil {
		return *c.Out
	}
	return "(none)"
}

func allCases(file *VectorFile) []*Case {
	var cases []*Case
	for gi := range file.Groups {
		for ci := range file.Groups[gi].Cases {
			cases = append(cases, &file.Groups[gi].Cases[ci])
		}
	}
	return cases
}

func loadVectors(path string) (*VectorFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file VectorFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &file, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
