package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/HamishFleming/Go-Mailsort/internal/cli"
	"github.com/HamishFleming/Go-Mailsort/internal/config"
	"github.com/HamishFleming/Go-Mailsort/internal/imapdebug"
)

var (
	configFile  string
	verbose     bool
	dryRun      bool
	summaryMD   bool
	summaryPath string
)

func main() {
	flag.StringVar(&configFile, "config", ".mailsort.yaml", "config file")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&dryRun, "dry-run", false, "don't actually modify emails")
	flag.BoolVar(&summaryMD, "summary-md", false, "write a Markdown summary report")
	flag.StringVar(&summaryPath, "summary-path", "", "Markdown summary output path")
	flag.Parse()
	dryRun = dryRun || hasArg(os.Args[1:], "--dry-run") || hasArg(os.Args[1:], "-dry-run")
	summaryMD = summaryMD || hasArg(os.Args[1:], "--summary-md") || hasArg(os.Args[1:], "-summary-md")
	if path, ok := argValue(os.Args[1:], "--summary-path"); ok {
		summaryPath = path
	} else if path, ok := argValue(os.Args[1:], "-summary-path"); ok {
		summaryPath = path
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cli.Verbose = verbose
	cli.DryRun = dryRun
	cli.SummaryMarkdown = summaryMD
	cli.SummaryPath = summaryPath
	log.SetOutput(os.Stderr)

	cfg, err := config.LoadMainConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	cmd := flag.Arg(0)

	rulesDir := cfg.RulesDir
	if rulesDir == "" {
		rulesDir = filepath.Join(filepath.Dir(configFile), "rules")
	}

	if cmd != "imap-debug" {
		rules, err := config.LoadRulesFromDir(rulesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load rules: %v\n", err)
			os.Exit(1)
		}
		cfg.Rules = rules
	}

	var runErr error
	switch cmd {
	case "init":
		runErr = cli.Init(cfg)
	case "scan":
		runErr = cli.Scan(cfg)
	case "preview":
		runErr = cli.Preview(cfg)
	case "apply":
		runErr = cli.Apply(cfg)
	case "rules":
		runErr = cli.Rules(cfg, rulesDir, flag.Args()[1:])
	case "imap-debug":
		runErr = imapdebug.Run(cfg, flag.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		flag.Usage()
		os.Exit(1)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", runErr)
		os.Exit(1)
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func argValue(args []string, name string) (string, bool) {
	prefix := name + "="
	for i, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix), true
		}
		if arg == name && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: mailsort <command> [options]")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  init      Create missing IMAP folders required by active rules")
		fmt.Println("  scan      List unread emails")
		fmt.Println("  preview   Show which emails match which rules")
		fmt.Println("  apply     Move matching emails")
		fmt.Println("  rules     Manage rules (list, add, remove, update, enable, disable)")
		fmt.Println("  imap-debug IMAP debugging toolkit")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}
}
