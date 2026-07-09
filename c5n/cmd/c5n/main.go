// Command c5n is the codegen engine: it reads a typed schema and data and
// emits typed, cross-language-conformant code for C#, TypeScript, and beyond.
// This is the seed — just the CLI shell for now.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/434a52/forge/c5n/internal/c5n"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "version":
		fmt.Println("c5n " + buildVersion())
	case "build":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := c5n.Build(dir); err != nil {
			fmt.Fprintln(os.Stderr, "c5n build: "+err.Error())
			os.Exit(1)
		}
	case "check":
		dir := "."
		if len(os.Args) >= 3 {
			dir = os.Args[2]
		}
		if err := c5n.Check(dir); err != nil {
			fmt.Fprintln(os.Stderr, "c5n check: "+err.Error())
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "c5n: unknown command %q\n", command)
		printUsage()
		os.Exit(1)
	}
}

// buildVersion returns the module version embedded at build time. It comes from
// the git tag when installed (go install ...@vX.Y.Z); local builds report "dev".
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: c5n <command>")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  version   print the c5n version")
	fmt.Fprintln(os.Stderr, "  build [dir]   read schema + data under dir (default .), emit code")
	fmt.Fprintln(os.Stderr, "  check [dir]   verify committed output matches sources; non-zero on drift")
}
