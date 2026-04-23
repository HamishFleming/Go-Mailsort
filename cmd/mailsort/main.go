package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/HamishFleming/Go-Mailsort/internal/cli"
	"github.com/HamishFleming/Go-Mailsort/internal/config"
)

var (
	configFile string
	verbose    bool
	dryRun     bool
)

func main() {
	flag.StringVar(&configFile, "config", ".mailsort.yaml", "config file")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&dryRun, "dry-run", false, "don't actually move emails")
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cli.Verbose = verbose
	cli.DryRun = dryRun
	log.SetOutput(os.Stderr)

	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	var runErr error
	switch cmd {
	case "scan":
		runErr = cli.Scan(cfg)
	case "preview":
		runErr = cli.Preview(cfg)
	case "apply":
		runErr = cli.Apply(cfg)
	case "rules":
		runErr = cli.Rules(cfg, flag.Args()[1:])
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

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: mailsort <command> [options]")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  scan      List unread emails")
		fmt.Println("  preview   Show which emails match which rules")
		fmt.Println("  apply     Move matching emails")
		fmt.Println("  rules     Manage rules (list, add, remove, update)")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
	}
}