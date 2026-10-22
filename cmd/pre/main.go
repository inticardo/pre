// Command pre is a lightweight source code preprocessor.
//
// It processes self-contained .pre files, which are made of a YAML frontmatter
// describing the files to generate and a Jet template body used to render them.
//
// Usage:
//
//	pre [options] <file.pre>
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/inticardo/pre/internal/engine"
)

// version is the version of the binary. It is overridden at build time with
// the -ldflags "-X main.version=..." option.
var version = "dev"

const usage = `pre - lightweight source code preprocessor

Usage:
  pre [options] <file.pre>

Options:
  -h, --help             Print this help message and exit
  -V, --version          Print the version of pre and exit
  -v, --verbose          Print the files being generated to stderr
      --stdout           Print the generated content to stdout instead of
                         writing the files (requires a single target)
      --target <output>  Process only the target with the given output name

Arguments:
  file.pre       The .pre file to process. The generated files are resolved
                 relative to it, regardless of the current directory.
                 Use "-" to read the source from stdin: includes and generated
                 files are then resolved relative to the current directory.

Examples:
  pre templates/repositories.pre
  pre --verbose templates/repositories.pre
  pre --stdout --target user_repository.go templates/repositories.pre
  cat templates/repositories.pre | pre --stdout -
  pre --version
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "pre: %v\n", err)
		os.Exit(1)
	}
}

// run parses the command line arguments and processes the requested file.
//
// It returns an error instead of exiting, so that it can be tested. The
// help and version flags are handled internally and never produce an error.
// Diagnostics such as the verbose progress messages are written to stderr, so
// that stdout only carries the generated content when --stdout is used.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("pre", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		help     bool
		showVer  bool
		verbose  bool
		toStdout bool
		target   string
	)
	flags.BoolVar(&help, "h", false, "print usage information and exit")
	flags.BoolVar(&help, "help", false, "print usage information and exit")
	flags.BoolVar(&showVer, "V", false, "print the version and exit")
	flags.BoolVar(&showVer, "version", false, "print the version and exit")
	flags.BoolVar(&verbose, "v", false, "print the files being generated to stderr")
	flags.BoolVar(&verbose, "verbose", false, "print the files being generated to stderr")
	flags.BoolVar(&toStdout, "stdout", false, "print the generated content to stdout instead of writing files")
	flags.StringVar(&target, "target", "", "process only the target with the given output name")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if help {
		fmt.Fprint(stdout, usage)
		return nil
	}

	if showVer {
		fmt.Fprintf(stdout, "pre %s\n", version)
		return nil
	}

	switch flags.NArg() {
	case 1:
	case 0:
		return errors.New("missing file argument: usage: pre [options] <file.pre>")
	default:
		return fmt.Errorf("too many arguments: expected a single .pre file, got %d", flags.NArg())
	}

	opts := engine.Options{Target: target}
	if verbose {
		opts.Log = stderr
	}
	if toStdout {
		opts.Stdout = stdout
	}

	if flags.Arg(0) == "-" {
		return engine.ProcessReader("<stdin>", stdin, ".", opts)
	}

	return engine.ProcessWithOptions(flags.Arg(0), opts)
}
