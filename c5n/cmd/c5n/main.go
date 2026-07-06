// Command c5n is the codegen engine: it reads a typed schema and data and
// emits typed, cross-language-conformant code for C#, TypeScript, and beyond.
// This is the seed — just the CLI shell for now.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
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
		// The real codegen lands here: read schema + data, emit code.
		fmt.Println("build: not implemented yet")
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
	fmt.Fprintln(os.Stderr, "  build     read schema + data, emit code (not implemented yet)")
}
