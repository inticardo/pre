//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	binaryName = "pre"
	mainPkg    = "./cmd/pre"
)

func main() {
	var command string
	var cgoFlag string

	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if len(os.Args) > 2 {
		cgoFlag = os.Args[2]
		if cgoFlag != "static" && cgoFlag != "dynamic" {
			fmt.Printf("Invalid CGO flag: %s (use 'static' or 'dynamic')\n", cgoFlag)
			printUsage()
			os.Exit(1)
		}
	}

	switch command {
	case "cross":
		crossCompile(cgoFlag)
	case "clean":
		clean()
	case "vendor":
		vendor()
	case "dev":
		if cgoFlag != "" {
			fmt.Println("Warning: Build mode ignored in dev (uses auto-detection)")
		}
		buildDev()
	case "build", "":
		build(cgoFlag)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run build.go [command] [mode]")
	fmt.Println("Commands:")
	fmt.Println("  build, (empty)  - Build for production (default: static)")
	fmt.Println("  cross          - Cross-compile for multiple platforms (default: static)")
	fmt.Println("  dev            - Build for development with race detector")
	fmt.Println("  vendor         - Refresh the vendor directory with the module dependencies")
	fmt.Println("  clean          - Remove the bin and vendor directories")
	fmt.Println("")
	fmt.Println("Build modes (not applicable to dev):")
	fmt.Println("  static         - Static binaries (CGO_ENABLED=0, no external dependencies)")
	fmt.Println("  dynamic        - Dynamic binaries (CGO_ENABLED=1, links with system libraries)")
}

// version reports the version string injected into the binary at build time.
func version() string {
	out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(out))
}

// ldflags returns the linker flags used for production builds.
func ldflags() string {
	return fmt.Sprintf("-w -s -X main.version=%s", version())
}

func build(cgoFlag string) {
	setupBuild()

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	cgoEnabled := "CGO_ENABLED=0"
	note := "Static binaries created (no external dependencies)"
	if cgoFlag == "dynamic" {
		cgoEnabled = "CGO_ENABLED=1"
		note = "Dynamic binaries created (CGO enabled, Go networking)"
	}
	buildFlags := fmt.Sprintf("-mod=vendor -tags netgo -ldflags=%q", ldflags())

	env := append(os.Environ(), cgoEnabled)

	runCmdWithEnv(env, fmt.Sprintf("go build %s -o %s %s", buildFlags, filepath.Join("bin", binaryName+ext), mainPkg))

	fmt.Println("Build complete!")
	fmt.Printf("Note: %s\n", note)
}

func buildDev() {
	setupBuild()
	fmt.Println("Building for development...")

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	fmt.Println("Attempting build with race detector...")

	env := append(os.Environ(), "CGO_ENABLED=1")
	buildFlagsRace := "-mod=vendor -race -tags netgo -gcflags=\"all=-N -l\""

	// Try building with the race detector first, falling back to a plain development build.
	raceCmd := fmt.Sprintf("go build %s -o %s %s", buildFlagsRace, filepath.Join("bin", binaryName+ext), mainPkg)
	if !tryBuildWithRace(env, raceCmd) {
		fmt.Println("Race detector not available, building without race detection...")
		buildFlagsDev := "-mod=vendor -tags netgo -gcflags=\"all=-N -l\" -ldflags=\"-X main.version=dev\""

		runCmd(fmt.Sprintf("go build %s -o %s %s", buildFlagsDev, filepath.Join("bin", binaryName+ext), mainPkg))

		fmt.Println("Development build complete!")
		fmt.Println("Note: Binaries include debugging symbols (race detector not available)")
	} else {
		runCmdWithEnv(env, raceCmd)

		fmt.Println("Development build complete!")
		fmt.Println("Note: Binaries include race detector and debugging symbols")
	}
}

func crossCompile(cgoFlag string) {
	setupBuild()
	fmt.Println("Cross-compiling for multiple platforms...")

	type platform struct {
		goos   string
		goarch string
		ext    string
	}

	platforms := []platform{
		// Primary desktop/server platforms
		{"windows", "amd64", ".exe"}, // Windows 64-bit
		{"linux", "amd64", ""},       // Linux 64-bit

		// ARM platforms (increasingly popular)
		{"linux", "arm", ""},    // ARM 32-bit Linux (Raspberry Pi 2/3)
		{"linux", "arm64", ""},  // ARM 64-bit Linux (Raspberry Pi 4, AWS Graviton)
		{"darwin", "arm64", ""}, // Apple Silicon (M1/M2/M3 Macs)
	}

	cgoEnabled := "0"
	note := "Static binaries created for all platforms"
	if cgoFlag == "dynamic" {
		cgoEnabled = "1"
		note = "Dynamic binaries created for all platforms (CGO enabled, Go networking)"
		fmt.Println("Warning: CGO_ENABLED=1 may cause issues with cross-compilation")
	}
	buildFlags := fmt.Sprintf("-mod=vendor -tags netgo -ldflags=%q", ldflags())

	for _, platform := range platforms {
		fmt.Printf("Building for %s/%s...\n", platform.goos, platform.goarch)

		env := append(os.Environ(), "GOOS="+platform.goos, "GOARCH="+platform.goarch, "CGO_ENABLED="+cgoEnabled)

		binary := fmt.Sprintf("%s-%s-%s%s", binaryName, platform.goos, platform.goarch, platform.ext)

		runCmdWithEnv(env, fmt.Sprintf("go build %s -o %s %s", buildFlags, filepath.Join("bin", binary), mainPkg))
	}

	fmt.Println("Cross-compilation complete!")
	fmt.Printf("Note: %s\n", note)
}

func clean() {
	fmt.Println("Cleaning...")

	if err := os.RemoveAll("bin"); err != nil {
		fmt.Printf("Warning: Failed to remove bin directory: %v\n", err)
	} else {
		fmt.Println("Removed bin directory")
	}

	if err := os.RemoveAll("vendor"); err != nil {
		fmt.Printf("Warning: Failed to remove vendor directory: %v\n", err)
	} else {
		fmt.Println("Removed vendor directory")
	}

	fmt.Println("Clean complete!")
}

// vendor refreshes the vendor directory with the module dependencies.
// The vendor directory lives in the project root and is used by every build
// through the -mod=vendor flag, making the compilation hermetic and offline.
func vendor() {
	fmt.Println("Vendoring dependencies...")

	runCmd("go mod tidy")
	runCmd("go mod vendor")

	fmt.Println("Vendor directory ready!")
}

// setupBuild performs common setup tasks for all build operations.
// The vendor directory is only generated when it is missing, so a build with
// an up to date vendor directory never needs the network nor the module cache.
func setupBuild() {
	if _, err := os.Stat(filepath.Join("vendor", "modules.txt")); err != nil {
		fmt.Println("Vendor directory not found, generating it...")
		runCmd("go mod tidy")
		runCmd("go mod vendor")
	} else {
		fmt.Println("Using vendored dependencies...")
	}

	fmt.Println("Building...")
	os.MkdirAll("bin", 0755)
}

// parseCommand splits a command string into program and arguments.
// Handles quoted arguments properly.
func parseCommand(cmdStr string) (string, []string) {
	parts := []string{}
	current := ""
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmdStr); i++ {
		c := cmdStr[i]

		switch c {
		case ' ', '\t':
			if !inQuote {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(c)
			}
		case '"', '\'':
			if !inQuote {
				inQuote = true
				quoteChar = c
			} else if c == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) == 0 {
		return "", nil
	}

	return parts[0], parts[1:]
}

// tryBuildWithRace attempts to build with race detector and returns true if successful.
func tryBuildWithRace(env []string, cmdStr string) bool {
	name, args := parseCommand(cmdStr)
	cmd := exec.Command(name, args...)
	cmd.Env = env
	// Suppress output for the test build
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func runCmd(cmdStr string) {
	name, args := parseCommand(cmdStr)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running '%s': %v\n", cmdStr, err)
		os.Exit(1)
	}
}

func runCmdWithEnv(env []string, cmdStr string) {
	name, args := parseCommand(cmdStr)
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running '%s': %v\n", cmdStr, err)
		os.Exit(1)
	}
}
