package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nssm-plus/internal/config"
	"nssm-plus/internal/service"
	"nssm-plus/internal/wrapper"
)

const usageText = `NSSM Plus - Windows Non-Sucking Service Manager Plus

Usage:
  nssm-plus <command> [arguments]

Commands:
  install <name> [options]   Install a new service
  remove <name>              Remove (uninstall) a service
  start <name>               Start a service
  stop <name>                Stop a service
  restart <name>             Restart a service
  status <name>              Query service status
  show <name>                Show full service configuration
  edit <name> [options]      Edit service configuration
  list [--json]              List all NSSM Plus managed services
  log <name> [--lines N]     Show service log (last N lines)
  export [name] [--output F] [--dir D] Export service config(s) to a JSON
                             bundle file, or split into per-service files
                             with --dir
  import <file|dir>          Import service config(s) from a JSON bundle
                             file or a directory of per-service config files;
                             configs are written to the config directory
  help                       Show this help message

Install options:
  --app <path>               Application path (required)
  --args <arguments>         Command line arguments
  --dir <directory>          Working directory
  --display <name>           Display name
  --desc <description>       Service description
  --start <type>             Start type: auto, demand, disabled (default: auto)
  --account <account>        Service account (default: LocalSystem)
  --password <password>      Service account password (insecure, visible in process list)
  --password-stdin           Read password from stdin (secure, recommended)
  --password-file <path>     Read password from file (secure)
  --env <KV pairs>           Environment variables (KEY=VAL;KEY2=VAL2)
  --restart-delay <seconds>  Restart delay on crash, 0=disabled (default: 0)
  --log-stdout <path>        Stdout log file path
  --log-stderr <path>        Stderr log file path
  --rotate-log               Enable log rotation

Edit options (same as install, all optional):
  --app, --args, --dir, --display, --desc, --start, --account,
  --password, --password-stdin, --password-file, --env,
  --restart-delay, --log-stdout, --log-stderr, --rotate-log

Examples:
  nssm-plus install MyService --app "C:\java\bin\java.exe" --args "-jar app.jar"
  nssm-plus install MyService --app "node.exe" --dir "C:\myapp" --start auto
  nssm-plus show MyService
  nssm-plus edit MyService --args "-jar app.jar --server.port=8081"
  nssm-plus list --json
  nssm-plus log MyService --lines 50
  nssm-plus export --output services.json
  nssm-plus export --dir "%ProgramData%\\NSSM-Plus\\configs"
  nssm-plus import services.json
`

func Run(args []string) {
	if len(args) < 2 {
		fmt.Print(usageText)
		os.Exit(0)
	}

	cmd := strings.ToLower(args[1])
	log.Printf("[cli] Run: command=%q, args=%v", cmd, args[2:])
	mgr := service.NewManager()

	switch cmd {
	case "help", "-h", "--help", "/?":
		fmt.Print(usageText)

	case "install":
		runInstall(mgr, args[2:])

	case "remove", "uninstall", "delete":
		runRemove(mgr, args[2:])

	case "start":
		runSimple(mgr, args[2:], mgr.Start, "start", "started")

	case "stop":
		runSimple(mgr, args[2:], mgr.Stop, "stop", "stopped")

	case "restart":
		runSimple(mgr, args[2:], mgr.Restart, "restart", "restarted")

	case "status":
		runStatus(mgr, args[2:])

	case "show", "get":
		runShow(mgr, args[2:])

	case "edit", "set":
		runEdit(mgr, args[2:])

	case "list":
		runList(mgr, args[2:])

	case "log":
		runLog(mgr, args[2:])

	case "export":
		runExport(mgr, args[2:])

	case "import":
		runImport(mgr, args[2:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n%s", cmd, usageText)
		os.Exit(1)
	}
}

func runInstall(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		fmt.Fprintln(os.Stderr, "Usage: nssm-plus install <name> --app <path> [options]")
		os.Exit(1)
	}

	cfg := service.ServiceConfig{
		ServiceName: args[0],
		StartType:   "auto",
	}

	envPairs := make(map[string]string)
	passwordStdin := false
	passwordFile := ""

	parseInstallArgs(args[1:], &cfg, envPairs, &passwordStdin, &passwordFile)

	if cfg.AppPath == "" {
		log.Printf("[cli] install: --app is missing for service %q", cfg.ServiceName)
		fmt.Fprintln(os.Stderr, "Error: --app is required for install")
		os.Exit(1)
	}

	resolvePassword(&cfg, passwordStdin, passwordFile)

	if cfg.Password != "" && !passwordStdin && passwordFile == "" {
		fmt.Fprintln(os.Stderr, "Warning: --password is visible in the process list. Consider using --password-stdin or --password-file instead.")
	}

	if len(envPairs) > 0 {
		cfg.Environment = envPairs
	}

	if cfg.DisplayName == "" {
		cfg.DisplayName = cfg.ServiceName
	}

	log.Printf("[cli] install: name=%q, app=%q, args=%q, dir=%q, startType=%q, account=%q, envCount=%d, restartDelay=%d",
		cfg.ServiceName, cfg.AppPath, cfg.Arguments, cfg.WorkDir, cfg.StartType, cfg.Account, len(cfg.Environment), cfg.RestartDelay)

	fmt.Printf("Installing service '%s'...\n", cfg.ServiceName)
	fmt.Printf("  App:      %s\n", cfg.AppPath)
	if cfg.Arguments != "" {
		fmt.Printf("  Args:     %s\n", cfg.Arguments)
	}
	if cfg.WorkDir != "" {
		fmt.Printf("  WorkDir:  %s\n", cfg.WorkDir)
	}
	fmt.Printf("  Start:    %s\n", cfg.StartType)

	if err := mgr.Install(cfg); err != nil {
		log.Printf("[cli] install: failed for service %q: %v", cfg.ServiceName, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] install: service %q installed successfully", cfg.ServiceName)
	fmt.Printf("Service '%s' installed successfully.\n", cfg.ServiceName)
}

func parseInstallArgs(args []string, cfg *service.ServiceConfig, envPairs map[string]string, passwordStdin *bool, passwordFile *string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--app" || arg == "-a":
			i++
			if i < len(args) {
				cfg.AppPath = args[i]
			}
		case strings.HasPrefix(arg, "--app="):
			cfg.AppPath = arg[6:]
		case arg == "--args":
			i++
			if i < len(args) {
				cfg.Arguments = args[i]
			}
		case strings.HasPrefix(arg, "--args="):
			cfg.Arguments = arg[7:]
		case arg == "--dir" || arg == "-d":
			i++
			if i < len(args) {
				cfg.WorkDir = args[i]
			}
		case strings.HasPrefix(arg, "--dir="):
			cfg.WorkDir = arg[6:]
		case arg == "--display":
			i++
			if i < len(args) {
				cfg.DisplayName = args[i]
			}
		case strings.HasPrefix(arg, "--display="):
			cfg.DisplayName = arg[10:]
		case arg == "--desc":
			i++
			if i < len(args) {
				cfg.Description = args[i]
			}
		case strings.HasPrefix(arg, "--desc="):
			cfg.Description = arg[7:]
		case arg == "--start" || arg == "-s":
			i++
			if i < len(args) {
				cfg.StartType = args[i]
			}
		case strings.HasPrefix(arg, "--start="):
			cfg.StartType = arg[8:]
		case arg == "--account":
			i++
			if i < len(args) {
				cfg.Account = args[i]
			}
		case strings.HasPrefix(arg, "--account="):
			cfg.Account = arg[10:]
		case arg == "--password" || arg == "-p":
			i++
			if i < len(args) {
				cfg.Password = args[i]
			}
		case strings.HasPrefix(arg, "--password="):
			cfg.Password = arg[11:]
		case arg == "--password-stdin":
			*passwordStdin = true
		case arg == "--password-file":
			i++
			if i < len(args) {
				*passwordFile = args[i]
			}
		case strings.HasPrefix(arg, "--password-file="):
			*passwordFile = arg[16:]
		case arg == "--env" || arg == "-e":
			i++
			if i < len(args) {
				parseEnvPairs(args[i], envPairs)
			}
		case strings.HasPrefix(arg, "--env="):
			parseEnvPairs(arg[6:], envPairs)
		case arg == "--restart-delay":
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil {
					cfg.RestartDelay = n
				}
			}
		case strings.HasPrefix(arg, "--restart-delay="):
			if n, err := strconv.Atoi(arg[16:]); err == nil {
				cfg.RestartDelay = n
			}
		case arg == "--log-stdout":
			i++
			if i < len(args) {
				cfg.LogStdout = args[i]
			}
		case strings.HasPrefix(arg, "--log-stdout="):
			cfg.LogStdout = arg[13:]
		case arg == "--log-stderr":
			i++
			if i < len(args) {
				cfg.LogStderr = args[i]
			}
		case strings.HasPrefix(arg, "--log-stderr="):
			cfg.LogStderr = arg[13:]
		case arg == "--rotate-log":
			cfg.RotateLog = true
		}
	}
}

func resolvePassword(cfg *service.ServiceConfig, passwordStdin bool, passwordFile string) {
	if passwordStdin {
		fmt.Print("Enter password: ")
		reader := bufio.NewReader(os.Stdin)
		pw, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password from stdin: %v\n", err)
			os.Exit(1)
		}
		cfg.Password = strings.TrimRight(pw, "\r\n")
	} else if passwordFile != "" {
		pw, err := os.ReadFile(passwordFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password file: %v\n", err)
			os.Exit(1)
		}
		cfg.Password = strings.TrimRight(string(pw), "\r\n")
	}
}

func parseEnvPairs(s string, env map[string]string) {
	pairs := strings.Split(s, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			val := strings.TrimSpace(pair[idx+1:])
			env[key] = val
		}
	}
}

func runRemove(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		os.Exit(1)
	}

	name := args[0]
	log.Printf("[cli] remove: name=%q", name)
	fmt.Printf("Removing service '%s'...\n", name)

	if err := mgr.Remove(name); err != nil {
		log.Printf("[cli] remove: failed for service %q: %v", name, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] remove: service %q removed successfully", name)
	fmt.Printf("Service '%s' removed successfully.\n", name)
}

type svcOp func(string) error

func runSimple(mgr *service.Manager, args []string, op svcOp, verb, pastTense string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: service name is required\n")
		os.Exit(1)
	}

	name := args[0]
	log.Printf("[cli] %s: name=%q", verb, name)
	fmt.Printf("%s service '%s'...\n", strings.Title(verb), name)

	if err := op(name); err != nil {
		log.Printf("[cli] %s: failed for service %q: %v", verb, name, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] %s: service %q %s successfully", verb, name, pastTense)
	fmt.Printf("Service '%s' %s successfully.\n", name, pastTense)
}

func runStatus(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		os.Exit(1)
	}

	name := args[0]
	log.Printf("[cli] status: name=%q", name)

	status, err := mgr.GetStatus(name)
	if err != nil {
		log.Printf("[cli] status: failed for service %q: %v", name, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] status: service %q is %s", name, status)
	fmt.Printf("Service '%s': %s\n", name, status)
}

func runShow(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		os.Exit(1)
	}

	name := args[0]
	log.Printf("[cli] show: name=%q", name)

	cfg, err := mgr.GetServiceConfig(name)
	if err != nil {
		log.Printf("[cli] show: failed for service %q: %v", name, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Service:     %s\n", cfg.ServiceName)
	fmt.Printf("DisplayName: %s\n", cfg.DisplayName)
	fmt.Printf("Description: %s\n", cfg.Description)
	fmt.Printf("AppPath:     %s\n", cfg.AppPath)
	fmt.Printf("Arguments:   %s\n", cfg.Arguments)
	fmt.Printf("WorkDir:     %s\n", cfg.WorkDir)
	fmt.Printf("StartType:   %s\n", cfg.StartType)
	fmt.Printf("Account:     %s\n", cfg.Account)
	fmt.Printf("LogStdout:   %s\n", cfg.LogStdout)
	fmt.Printf("LogStderr:   %s\n", cfg.LogStderr)
	fmt.Printf("RotateLog:   %v\n", cfg.RotateLog)
	fmt.Printf("RestartDelay:%d\n", cfg.RestartDelay)
	fmt.Printf("RestartTimeout: %d\n", cfg.RestartTimeout)

	if len(cfg.Environment) > 0 {
		fmt.Println("Environment:")
		for k, v := range cfg.Environment {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}

	if len(cfg.Dependencies) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(cfg.Dependencies, ", "))
	}

	for i := 1; i < len(args); i++ {
		if args[i] == "--json" {
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(data))
			return
		}
	}
}

func runEdit(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		fmt.Fprintln(os.Stderr, "Usage: nssm-plus edit <name> [options]")
		os.Exit(1)
	}

	name := args[0]
	log.Printf("[cli] edit: name=%q", name)

	current, err := mgr.GetServiceConfig(name)
	if err != nil {
		log.Printf("[cli] edit: failed to get current config for %q: %v", name, err)
		fmt.Fprintf(os.Stderr, "Error: failed to get current config: %v\n", err)
		os.Exit(1)
	}

	envPairs := make(map[string]string)
	if len(current.Environment) > 0 {
		envPairs = current.Environment
	}

	passwordStdin := false
	passwordFile := ""

	parseInstallArgs(args[1:], current, envPairs, &passwordStdin, &passwordFile)

	resolvePassword(current, passwordStdin, passwordFile)

	if len(envPairs) > 0 {
		current.Environment = envPairs
	}

	fmt.Printf("Updating service '%s'...\n", name)
	if err := mgr.Modify(name, *current); err != nil {
		log.Printf("[cli] edit: failed for service %q: %v", name, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] edit: service %q updated successfully", name)
	fmt.Printf("Service '%s' updated successfully.\n", name)
}

func runList(mgr *service.Manager, args []string) {
	useJSON := false
	for _, arg := range args {
		if arg == "--json" || arg == "-j" {
			useJSON = true
		}
	}

	log.Printf("[cli] list: json=%v", useJSON)

	services, err := mgr.ListServices()
	if err != nil {
		log.Printf("[cli] list: failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[cli] list: found %d services", len(services))

	if useJSON {
		data, err := json.MarshalIndent(services, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	if len(services) == 0 {
		fmt.Println("No NSSM Plus managed services found.")
		return
	}

	fmt.Printf("%-30s %-20s %-12s %-10s\n", "NAME", "DISPLAY NAME", "STATUS", "START TYPE")
	fmt.Println(strings.Repeat("-", 72))

	for _, svc := range services {
		displayName := svc.DisplayName
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}
		fmt.Printf("%-30s %-20s %-12s %-10s\n", svc.Name, displayName, svc.Status, svc.StartType)
	}

	fmt.Printf("\nTotal: %d service(s)\n", len(services))
}

func runLog(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: service name is required")
		os.Exit(1)
	}

	name := args[0]
	lines := 50

	for i := 1; i < len(args); i++ {
		if args[i] == "--lines" || args[i] == "-n" {
			i++
			if i < len(args) {
				if n, err := strconv.Atoi(args[i]); err == nil && n > 0 {
					lines = n
				}
			}
		} else if strings.HasPrefix(args[i], "--lines=") {
			if n, err := strconv.Atoi(args[i][8:]); err == nil && n > 0 {
				lines = n
			}
		}
	}

	log.Printf("[cli] log: name=%q, lines=%d", name, lines)

	logPath := wrapper.LogPath(name)
	data, err := os.ReadFile(logPath)
	if err != nil {
		log.Printf("[cli] log: failed to read log file %q: %v", logPath, err)
		fmt.Fprintf(os.Stderr, "Error reading log file: %v\n", err)
		os.Exit(1)
	}

	allLines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	for _, line := range allLines[start:] {
		fmt.Println(line)
	}
}

func runExport(mgr *service.Manager, args []string) {
	var targetName string
	outputFile := ""
	outputDir := ""

	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") && targetName == "" {
			targetName = args[i]
		} else if args[i] == "--output" || args[i] == "-o" {
			i++
			if i < len(args) {
				outputFile = args[i]
			}
		} else if strings.HasPrefix(args[i], "--output=") {
			outputFile = args[i][9:]
		} else if args[i] == "--dir" || args[i] == "-d" {
			i++
			if i < len(args) {
				outputDir = args[i]
			}
		} else if strings.HasPrefix(args[i], "--dir=") {
			outputDir = args[i][6:]
		}
	}

	log.Printf("[cli] export: target=%q, output=%q, dir=%q", targetName, outputFile, outputDir)

	var configs []service.ServiceConfig

	if targetName != "" {
		cfg, err := mgr.GetServiceConfig(targetName)
		if err != nil {
			log.Printf("[cli] export: failed to get config for %q: %v", targetName, err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		configs = []service.ServiceConfig{*cfg}
	} else {
		services, err := mgr.ListServices()
		if err != nil {
			log.Printf("[cli] export: failed to list services: %v", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for _, svc := range services {
			cfg, err := mgr.GetServiceConfig(svc.Name)
			if err != nil {
				log.Printf("[cli] export: skipping service %q: %v", svc.Name, err)
				fmt.Fprintf(os.Stderr, "Warning: failed to get config for '%s': %v\n", svc.Name, err)
				continue
			}
			configs = append(configs, *cfg)
		}
	}

	if len(configs) == 0 {
		log.Printf("[cli] export: no services to export")
		fmt.Fprintln(os.Stderr, "No services to export")
		os.Exit(1)
	}

	log.Printf("[cli] export: exporting %d service(s)", len(configs))

	// --dir mode: write one single-service config file per service
	if outputDir != "" {
		cfgMgr := config.NewManager()
		exported := 0
		for _, cfg := range configs {
			if err := cfgMgr.SaveServiceTo(outputDir, cfg); err != nil {
				log.Printf("[cli] export: failed to write config for %q: %v", cfg.ServiceName, err)
				fmt.Fprintf(os.Stderr, "Warning: failed to export '%s': %v\n", cfg.ServiceName, err)
				continue
			}
			exported++
		}
		if exported == 0 {
			fmt.Fprintln(os.Stderr, "Error: no service config could be written")
			os.Exit(1)
		}
		fmt.Printf("Exported %d/%d service config(s) to directory %s\n", exported, len(configs), outputDir)
		return
	}

	if outputFile == "" {
		data, err := json.MarshalIndent(config.ConfigFile{Services: configs}, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	cfgMgr := config.NewManager()
	if err := cfgMgr.SaveToFile(outputFile, configs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exported %d service(s) to %s\n", len(configs), outputFile)
}

func runImport(mgr *service.Manager, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: config file or directory path is required")
		fmt.Fprintln(os.Stderr, "Usage: nssm-plus import <file|dir>")
		os.Exit(1)
	}

	path := args[0]
	log.Printf("[cli] import: path=%q", path)

	cfgMgr := config.NewManager()

	var configs []service.ServiceConfig

	info, err := os.Stat(path)
	if err != nil {
		log.Printf("[cli] import: failed to stat %q: %v", path, err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		files, err := cfgMgr.LoadDir(path)
		if err != nil {
			log.Printf("[cli] import: failed to load config dir %q: %v", path, err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, f := range files {
			if f.Error != "" {
				fmt.Fprintf(os.Stderr, "Warning: skipping %s: %s\n", filepath.Base(f.FilePath), f.Error)
				continue
			}
			configs = append(configs, f.Config)
		}
	} else {
		configs, err = cfgMgr.LoadFromFile(path)
		if err != nil {
			log.Printf("[cli] import: failed to load config from %q: %v", path, err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	if len(configs) == 0 {
		log.Printf("[cli] import: no service configurations found in %q", path)
		fmt.Fprintln(os.Stderr, "No service configurations found")
		os.Exit(1)
	}

	log.Printf("[cli] import: found %d service(s)", len(configs))

	installed := 0
	for _, cfg := range configs {
		if cfg.AppPath == "" {
			log.Printf("[cli] import: skipping %q - no appPath", cfg.ServiceName)
			fmt.Fprintf(os.Stderr, "Warning: skipping '%s' - no appPath\n", cfg.ServiceName)
			continue
		}
		if cfg.DisplayName == "" {
			cfg.DisplayName = cfg.ServiceName
		}
		if cfg.StartType == "" {
			cfg.StartType = "auto"
		}

		// Persist the per-service config file first so the service shows up
		// in the config directory view even if the install itself fails.
		if err := cfgMgr.SaveService(cfg); err != nil {
			log.Printf("[cli] import: failed to save config file for %q: %v", cfg.ServiceName, err)
			fmt.Fprintf(os.Stderr, "Warning: failed to save config file for '%s': %v\n", cfg.ServiceName, err)
		}

		log.Printf("[cli] import: installing service %q (app=%q)", cfg.ServiceName, cfg.AppPath)
		fmt.Printf("Installing service '%s'...\n", cfg.ServiceName)
		if err := mgr.Install(cfg); err != nil {
			log.Printf("[cli] import: failed to install service %q: %v", cfg.ServiceName, err)
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			continue
		}
		installed++
		fmt.Printf("  Installed successfully.\n")
	}

	log.Printf("[cli] import: completed, installed %d/%d service(s)", installed, len(configs))
	fmt.Printf("\nImported %d/%d service(s)\n", installed, len(configs))
}
