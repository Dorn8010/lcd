// lcd - library change directory
// Version 0.4 (Fast Select + Direct)
// © 2025 by Alexander Dorn, MIT license
// https://github.com/Dorn8010/lcd

// To compile on Linux :
// sudo apt install golang && go build -o lcd lcd.go
// To compile on Mac :
// brew install go && go build -o lcd lcd.go
// To install locally
// sudo cp lcd /usr/local/bin/


package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// Constants
const (
	version    = "0.4-direct"
	dbFilename = ".lcd-tree.txt"
)

// Config holds command line arguments
type Config struct {
	help       bool
	version    bool
	verbose    bool
	printOnly  bool
	copyToClip bool
	rescan     bool
	newBaseDir string
	searchTerm string
}

func main() {
	// Parse Flags
	cfg := parseFlags()

	if cfg.help {
		printHelp()
		os.Exit(1)
	}

	if cfg.version {
		fmt.Printf("lcd version %s\n", version)
		os.Exit(1)
	}

	// Determine DB path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fatal("Could not determine user home directory: %v", err)
	}
	dbPath := filepath.Join(homeDir, dbFilename)

	// --- LOGIC FLOW ---

	// 1. Check if DB exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if cfg.verbose {
			fmt.Fprintf(os.Stderr, "Database not found at %s. Initializing...\n", dbPath)
		}
		cfg.rescan = true
	}

	// 2. Determine Base Directory
	baseDir := homeDir // Default
	if cfg.newBaseDir != "" {
		baseDir = cfg.newBaseDir
		cfg.rescan = true
	} else if !cfg.rescan {
		// Read existing base dir from file header
		f, err := os.Open(dbPath)
		if err == nil {
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				storedBase := strings.TrimSpace(scanner.Text())
				if storedBase != "" {
					baseDir = storedBase
				}
			}
			f.Close()
		}
	}

	// 3. Rescan if required
	if cfg.rescan {
		if cfg.verbose {
			fmt.Fprintf(os.Stderr, "Scanning directories starting from: %s\n", baseDir)
		}
		err := generateDatabase(dbPath, baseDir)
		if err != nil {
			fatal("Error generating database: %v", err)
		}
		if cfg.verbose {
			fmt.Fprintf(os.Stderr, "Scan complete. Database saved.\n")
		}
		
		if cfg.searchTerm == "" {
			fmt.Fprintf(os.Stderr, "Database updated.")
			os.Exit(1)
		}
	}

	// 4. Perform Search
	if cfg.searchTerm == "" {
		fatal("Please provide a directory name to search for.")
	}

	match, err := searchDatabaseOptimized(dbPath, cfg.searchTerm)
	if err != nil {
		fatal("%v", err)
	}

	// 5. Handle "Print" or "Copy" Actions (These exit early)
	if cfg.printOnly {
		fmt.Println(match)
		os.Exit(0)
	}

	if cfg.copyToClip {
		err := copyToClipboard(match)
		if err != nil {
			fatal("Failed to copy to clipboard: %v", err)
		}
		fmt.Printf("Copied to clipboard: %s\n", match)
		os.Exit(0)
	}

	// 6. DIRECT CHANGE DIRECTORY (Method 1)
	// Instead of printing, we replace the process.
	enterDirectory(match)
}

// --- CORE FUNCTION FOR METHOD 1 ---

func enterDirectory(targetPath string) {

	// A. Change the Go process's working directory
        currentDir, err := os.Getwd()
        if err == nil {
		// Clean both paths to resolve trailing slashes or relative components
		if filepath.Clean(currentDir) == filepath.Clean(targetPath) {
			fmt.Printf("Already in: %s\n", targetPath)
			os.Exit(0) // Stop here, do not spawn a new shell
		}
	}
        
        err = os.Chdir(targetPath)
	if err != nil {
		fatal("Could not enter directory %s: %v", targetPath, err)
	}

	// B. Detect the user's current shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
      
	// C. Prepare Arguments
	// argv[0] is the command name (by convention)
	// "-i" forces the shell to be interactive (load history/rc files)
	// Note: Some shells behave better if argv[0] starts with "-" (login shell),
	// but "-i" is the standard way to just "start a new interactive session".
	args := []string{shell, "-i"}


	fmt.Printf("cd %s\n", targetPath)
	// D. Execute
	env := os.Environ()
	err = syscall.Exec(shell, args, env)
	
	if err != nil {
		fatal("Failed to spawn new shell: %v", err)
	}
}

// --- HELPER FUNCTIONS ---

func parseFlags() Config {
	var cfg Config
	flag.BoolVar(&cfg.help, "h", false, "Show help")
	flag.BoolVar(&cfg.help, "help", false, "Show help")
	flag.BoolVar(&cfg.version, "version", false, "Show version")
	flag.BoolVar(&cfg.verbose, "v", false, "Verbose output")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&cfg.printOnly, "print", false, "Just print the directory path")
	flag.BoolVar(&cfg.copyToClip, "copy", false, "Copy path to clipboard")
	flag.BoolVar(&cfg.rescan, "rescan", false, "Force a rescan")
	flag.StringVar(&cfg.newBaseDir, "newbasedir", "", "Set a new root directory")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		cfg.searchTerm = strings.Join(args, " ")
	}
	return cfg
}

func generateDatabase(dbPath string, baseDir string) error {
	file, err := os.Create(dbPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(baseDir + "\n")
	if err != nil {
		return err
	}
        fmt.Printf("(Re-)Scanning directory tree from %s\n", baseDir)
	err = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Permission denied or other error, skip directory
			return filepath.SkipDir
		}

		// CHANGE: We now allow hidden directories (starting with ".")
		// We only skip ".git" specifically because it contains thousands of 
		// internal files that are useless for navigation and slow down the search.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		if d.IsDir() {
			_, err := writer.WriteString(path + "\n")
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return writer.Flush()
}


func searchDatabaseOptimized(dbPath string, term string) (string, error) {
	file, err := os.Open(dbPath)
	if err != nil {
		return "", fmt.Errorf("could not open database: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header

	termLower := strings.ToLower(term)
	
	var bestExact string
	var bestPartial string
	
	bestExactLen := int(^uint(0) >> 1)
	bestPartialLen := int(^uint(0) >> 1)

	for scanner.Scan() {
		path := scanner.Text()
		
		lastSlash := strings.LastIndexByte(path, '/')
		name := path
		if lastSlash >= 0 {
			name = path[lastSlash+1:]
		}
		
		nameLower := strings.ToLower(name)
		pathLen := len(path)

		if nameLower == termLower {
			if pathLen < bestExactLen {
				bestExact = path
				bestExactLen = pathLen
			}
			continue
		}

		if strings.Contains(nameLower, termLower) {
			if pathLen < bestPartialLen {
				bestPartial = path
				bestPartialLen = pathLen
			}
		}
	}

	if bestExact != "" {
		return bestExact, nil
	}
	if bestPartial != "" {
		return bestPartial, nil
	}

	return "", fmt.Errorf("directory not found: %s", term)
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found")
		}
	default:
		return fmt.Errorf("unsupported OS")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func printHelp() {
	fmt.Printf(`lcd - Library Change Directory
Usage: lcd [options] <directory_name_or_fragment>

Description:
  Fast directory navigation using a cached directory tree (~/.lcd-tree.txt).
  First run will index your home directory automatically.
  Typing exit in the CLI brings you back to the old directory.
  (C) 2025 by Alexander Dorn, MIT license, Ver. %s

Options:
  --verbose, -v      Show detailed logs during operation
  --print            Print the found path to stdout (do not cd)
  --copy             Copy the found path to system clipboard
  --rescan           Force a rescan of the filesystem
  --newbasedir <dir> Set a new root directory for scanning (implies --rescan)
  --version          Show version info
  --help, -h         Show this help message

Search Logic:
  1. Searches for an Exact Match (case-insensitive) of the directory name.
  2. If not found, searches for a Partial Match.
`, version)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
