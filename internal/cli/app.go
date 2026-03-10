package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/agentepics/epics.sh/internal/cron"
	"github.com/agentepics/epics.sh/internal/doctor"
	"github.com/agentepics/epics.sh/internal/epic"
	"github.com/agentepics/epics.sh/internal/hostapi"
	"github.com/agentepics/epics.sh/internal/hosts"
	"github.com/agentepics/epics.sh/internal/install"
	"github.com/agentepics/epics.sh/internal/logutil"
	"github.com/agentepics/epics.sh/internal/plan"
	"github.com/agentepics/epics.sh/internal/resume"
	"github.com/agentepics/epics.sh/internal/state"
	"github.com/agentepics/epics.sh/internal/workspace"
	"golang.org/x/term"
)

type App struct {
	CWD           string
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	IsInteractive func() bool
}

type globalFlags struct {
	JSON  bool
	Quiet bool
	Yes   bool
}

func NewApp(cwd string, stdin io.Reader, stdout, stderr io.Writer) App {
	if cwd == "" {
		resolved, err := os.Getwd()
		if err == nil {
			cwd = resolved
		}
	}
	return App{
		CWD:    cwd,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		IsInteractive: func() bool {
			return stdinIsInteractive(stdin)
		},
	}
}

func (a App) Run(args []string) int {
	flags, rest, err := parseGlobalFlags(args)
	if err != nil {
		return a.fail(flags, err)
	}
	if len(rest) == 0 {
		a.printUsage()
		return 1
	}

	switch rest[0] {
	case "init":
		return a.runInit(flags, rest[1:])
	case "upgrade-skill-footer":
		return a.runUpgradeSkillFooter(flags, rest[1:])
	case "install":
		return a.runInstall(flags, rest[1:])
	case "validate":
		return a.runValidate(flags, rest[1:])
	case "info":
		return a.runInfo(flags, rest[1:])
	case "status":
		return a.runStatus(flags, rest[1:])
	case "resume":
		return a.runResume(flags, rest[1:])
	case "doctor":
		return a.runDoctor(flags, rest[1:])
	case "host":
		return a.runHost(flags, rest[1:])
	case "state":
		return a.runState(flags, rest[1:])
	case "plan":
		return a.runPlan(flags, rest[1:])
	case "log":
		return a.runLog(flags, rest[1:])
	case "cron":
		return a.runCron(flags, rest[1:])
	case "daemon":
		return a.runDaemon(flags, rest[1:])
	case "workspace":
		return a.runDaemonWorkspace(flags, rest[1:])
	case "route":
		return a.runDaemonRoute(flags, rest[1:])
	case "run":
		return a.runDaemonRun(flags, rest[1:])
	default:
		return a.fail(flags, fmt.Errorf("unknown command %q", rest[0]))
	}
}

func (a App) runInit(flags globalFlags, args []string) int {
	if len(args) > 0 {
		return a.fail(flags, errors.New("init does not accept additional arguments"))
	}

	slug := sanitizeSlug(filepath.Base(a.CWD))
	if slug == "" {
		slug = "new-epic"
	}
	title := titleFromSlug(slug)
	skillContent := epic.RefreshSkillFooter(fmt.Sprintf(`---
name: %s
description: Describe what this epic does, when to activate it, and that durable operating context lives in EPIC.md.
---

# %s

Use this epic when you need a durable, resumable workflow for this directory. `+"`EPIC.md`"+` is the authoritative source for lifecycle, state model, guardrails, and resume behavior.

See the **Agent Epics** section below if this is your first encounter with the Agent Epics system.

## Operating notes

- Start by reading `+"`EPIC.md`"+` before doing substantive work.
- Keep volatile plans, state, logs, and outputs under `+"`runtime/`"+` instead of rewriting this file.
`, slug, title))

	files := map[string]string{
		"SKILL.md": skillContent,
		"EPIC.md": `---
spec_version: 0.5.2
id: ` + slug + `
---

# ` + title + `

## Objective

Describe the durable objective this Epic helps complete.

## Success criteria

- Runtime state, plans, and logs stay aligned with the work.
- Resume context can be reconstructed across sessions.
`,
	}

	var created []string
	for name, content := range files {
		path := filepath.Join(a.CWD, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
			return a.fail(flags, err)
		}
		created = append(created, name)
	}

	payload := map[string]any{"created": created}
	if flags.JSON {
		return a.emitJSON(payload)
	}
	if len(created) == 0 {
		a.print("Epic package already initialized in the current directory.")
		return 0
	}
	a.print("Initialized Epic package:")
	for _, item := range created {
		a.print("  - " + item)
	}
	return 0
}

func (a App) runUpgradeSkillFooter(flags globalFlags, args []string) int {
	arg, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}

	target, err := a.resolvePackageTarget(arg)
	if err != nil {
		return a.fail(flags, err)
	}

	skillPath, changed, err := epic.UpgradeSkillFooter(target)
	if err != nil {
		return a.fail(flags, err)
	}

	payload := map[string]any{
		"path":    skillPath,
		"changed": changed,
		"marker":  epic.CanonicalSkillFooterMarker,
	}
	if flags.JSON {
		return a.emitJSON(payload)
	}
	if changed {
		a.print("Refreshed Agent Epics footer in " + skillPath)
		return 0
	}
	a.print("SKILL.md already has the current Agent Epics footer.")
	return 0
}

func (a App) runInstall(flags globalFlags, args []string) int {
	installFlags, err := parseInstallArgs(args)
	if err != nil {
		return a.fail(flags, err)
	}

	host := installFlags.Host
	if host == "" {
		host, err = a.selectHost()
		if err != nil {
			return a.fail(flags, err)
		}
	}

	adapter, err := hosts.Resolve(host)
	if err != nil {
		return a.fail(flags, err)
	}

	result, err := install.Install(a.CWD, installFlags.Source, host, func(slug string) string {
		return adapter.InstallDir(a.CWD, slug)
	})
	if err != nil {
		return a.fail(flags, err)
	}

	hostSetup, err := adapter.Setup(a.CWD)
	if err != nil {
		return a.fail(flags, err)
	}

	if flags.JSON {
		return a.emitJSON(map[string]any{
			"install":    result,
			"host":       host,
			"host_setup": hostSetup,
		})
	}

	a.print(fmt.Sprintf("Installed %s for %s into %s", result.Package.Title, host, result.Install.InstalledDir))
	if len(result.Diagnostics) > 0 {
		a.print(fmt.Sprintf("Validation diagnostics: %d", len(result.Diagnostics)))
	}
	a.printHostSetupResult(host, hostSetup)
	return 0
}

func (a App) runValidate(flags globalFlags, args []string) int {
	arg, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}
	target, err := a.resolvePackageTarget(arg)
	if err != nil {
		return a.fail(flags, err)
	}

	pkg, diagnostics, err := epic.Validate(target)
	if err != nil {
		return a.fail(flags, err)
	}

	payload := map[string]any{
		"package":     pkg,
		"diagnostics": diagnostics,
		"valid":       !epic.HasErrors(diagnostics),
	}
	if flags.JSON {
		code := a.emitJSON(payload)
		if epic.HasErrors(diagnostics) {
			return 1
		}
		return code
	}

	if len(diagnostics) == 0 {
		a.print(fmt.Sprintf("%s is valid.", pkg.Title))
		return 0
	}

	for _, diagnostic := range diagnostics {
		a.print(fmt.Sprintf("%s: %s (%s)", strings.ToUpper(diagnostic.Level), diagnostic.Message, diagnostic.Path))
	}
	if epic.HasErrors(diagnostics) {
		return 1
	}
	return 0
}

func (a App) runInfo(flags globalFlags, args []string) int {
	arg, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}
	target, record, err := a.resolvePackageReference(arg)
	if err != nil {
		return a.fail(flags, err)
	}

	pkg, diagnostics, err := epic.Validate(target)
	if err != nil {
		return a.fail(flags, err)
	}

	payload := map[string]any{
		"package":     pkg,
		"diagnostics": diagnostics,
		"install":     record,
	}
	if flags.JSON {
		return a.emitJSON(payload)
	}

	a.print(fmt.Sprintf("Title: %s", pkg.Title))
	a.print(fmt.Sprintf("Slug: %s", pkg.Slug))
	a.print(fmt.Sprintf("Root: %s", filepath.ToSlash(pkg.Root)))
	if pkg.Summary != "" {
		a.print(fmt.Sprintf("Summary: %s", pkg.Summary))
	}
	if record.Source != "" {
		a.print(fmt.Sprintf("Source: %s", record.Source))
	}
	if record.Host != "" {
		a.print(fmt.Sprintf("Host: %s", record.Host))
	}
	if record.InstalledDir != "" {
		a.print(fmt.Sprintf("Installed: %s", record.InstalledDir))
	}
	if record.Version != "" {
		a.print(fmt.Sprintf("Version: %s", record.Version))
	}
	if record.Digest != "" {
		a.print(fmt.Sprintf("Digest: %s", record.Digest))
	}
	return 0
}

func (a App) runResume(flags globalFlags, args []string) int {
	arg, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}
	target, _, err := a.resolvePackageReference(arg)
	if err != nil {
		return a.fail(flags, err)
	}

	pkg, diagnostics, err := epic.Validate(target)
	if err != nil {
		return a.fail(flags, err)
	}
	if epic.HasErrors(diagnostics) {
		return a.fail(flags, errors.New("cannot resume an invalid Epic package"))
	}

	result, err := resume.Build(pkg)
	if err != nil {
		return a.fail(flags, err)
	}

	if flags.JSON {
		return a.emitJSON(result)
	}
	a.print(result.Context)
	return 0
}

func (a App) runStatus(flags globalFlags, args []string) int {
	arg, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}
	target, record, err := a.resolvePackageReference(arg)
	if err != nil {
		return a.fail(flags, err)
	}

	pkg, diagnostics, err := epic.Validate(target)
	if err != nil {
		return a.fail(flags, err)
	}
	if epic.HasErrors(diagnostics) {
		return a.fail(flags, errors.New("cannot inspect status for an invalid Epic package"))
	}

	result, err := resume.Build(pkg)
	if err != nil {
		return a.fail(flags, err)
	}

	payload := map[string]any{
		"package":   pkg,
		"install":   record,
		"statePath": result.StatePath,
		"planPath":  result.PlanPath,
		"nextStep":  result.NextStep,
	}
	if len(result.LogPaths) > 0 {
		payload["latestLogPath"] = result.LogPaths[len(result.LogPaths)-1]
	}
	if len(result.RecentLogNotes) > 0 {
		payload["latestLogNote"] = result.RecentLogNotes[len(result.RecentLogNotes)-1]
	}
	if flags.JSON {
		return a.emitJSON(payload)
	}

	a.print(fmt.Sprintf("Epic: %s", pkg.Title))
	a.print(fmt.Sprintf("Slug: %s", pkg.Slug))
	if record.Host != "" {
		a.print(fmt.Sprintf("Host: %s", record.Host))
	}
	if record.InstalledDir != "" {
		a.print(fmt.Sprintf("Installed: %s", record.InstalledDir))
	}
	if result.PlanPath != "" {
		a.print(fmt.Sprintf("Current plan: %s", result.PlanPath))
	}
	if result.NextStep != "" {
		a.print(fmt.Sprintf("Next step: %s", result.NextStep))
	}
	if len(result.LogPaths) > 0 {
		a.print(fmt.Sprintf("Latest log: %s", result.LogPaths[len(result.LogPaths)-1]))
	}
	if len(result.RecentLogNotes) > 0 {
		a.print(fmt.Sprintf("Latest note: %s", result.RecentLogNotes[len(result.RecentLogNotes)-1]))
	}
	return 0
}

func (a App) runDoctor(flags globalFlags, args []string) int {
	if len(args) > 0 {
		return a.fail(flags, errors.New("doctor does not accept additional arguments"))
	}

	result, err := doctor.Run(a.CWD)
	if err != nil {
		return a.fail(flags, err)
	}

	if flags.JSON {
		code := a.emitJSON(result)
		if doctor.HasFailures(result) {
			return 1
		}
		return code
	}

	for _, check := range result.Checks {
		a.print(fmt.Sprintf("%s: %s - %s", strings.ToUpper(check.Status), check.Name, check.Message))
	}
	if doctor.HasFailures(result) {
		return 1
	}
	return 0
}

func (a App) runHost(flags globalFlags, args []string) int {
	if len(args) != 2 {
		return a.fail(flags, fmt.Errorf("expected: host <setup|doctor> <%s>", strings.Join(hosts.Supported(), "|")))
	}

	adapter, err := hosts.Resolve(args[1])
	if err != nil {
		return a.fail(flags, err)
	}

	switch args[0] {
	case "setup":
		result, err := adapter.Setup(a.CWD)
		if err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(result)
		}
		a.printHostSetupResult(args[1], result)
		return 0
	case "doctor":
		checks, err := adapter.Doctor(a.CWD)
		if err != nil {
			return a.fail(flags, err)
		}
		result := doctor.Result{Checks: checks}
		if flags.JSON {
			code := a.emitJSON(result)
			if doctor.HasFailures(result) {
				return 1
			}
			return code
		}
		for _, check := range result.Checks {
			a.print(fmt.Sprintf("%s: %s - %s", strings.ToUpper(check.Status), check.Name, check.Message))
		}
		if doctor.HasFailures(result) {
			return 1
		}
		return 0
	default:
		return a.fail(flags, fmt.Errorf("expected: host <setup|doctor> <%s>", strings.Join(hosts.Supported(), "|")))
	}
}

func (a App) runState(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: state <get|set> ..."))
	}
	switch args[0] {
	case "get":
		return a.runStateGet(flags, args[1:])
	case "set":
		return a.runStateSet(flags, args[1:])
	default:
		return a.fail(flags, errors.New("expected: state <get|set> ..."))
	}
}

func (a App) runStateGet(flags globalFlags, args []string) int {
	key, err := requireAtMostOneArg(args)
	if err != nil {
		return a.fail(flags, err)
	}

	value, snapshot, err := state.Get(a.CWD, key)
	if err != nil {
		return a.fail(flags, err)
	}

	if key == "" {
		return a.emitJSON(snapshot.Data)
	}
	if flags.JSON {
		return a.emitJSON(map[string]any{
			"key":   key,
			"value": value,
		})
	}
	if str, ok := value.(string); ok {
		a.print(str)
		return 0
	}
	return a.emitJSON(value)
}

func (a App) runStateSet(flags globalFlags, args []string) int {
	if len(args) != 2 {
		return a.fail(flags, errors.New("expected: state set <key> <value>"))
	}

	snapshot, value, err := state.Set(a.CWD, args[0], args[1])
	if err != nil {
		return a.fail(flags, err)
	}

	path := epic.RelativePath(a.CWD, snapshot.Path)
	if flags.JSON {
		return a.emitJSON(map[string]any{
			"key":   args[0],
			"value": value,
			"path":  path,
		})
	}
	a.print(path)
	return 0
}

func (a App) runPlan(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: plan <list|current|create> ..."))
	}
	switch args[0] {
	case "list":
		return a.runPlanList(flags, args[1:])
	case "current":
		return a.runPlanCurrent(flags, args[1:])
	case "create":
		return a.runPlanCreate(flags, args[1:])
	default:
		return a.fail(flags, errors.New("expected: plan <list|current|create> ..."))
	}
}

func (a App) runPlanList(flags globalFlags, args []string) int {
	if len(args) > 0 {
		return a.fail(flags, errors.New("plan list does not accept additional arguments"))
	}

	entries, err := plan.List(a.CWD)
	if err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(entries)
	}
	for _, entry := range entries {
		if entry.Title != "" {
			a.print(fmt.Sprintf("%s\t%s", entry.Path, entry.Title))
			continue
		}
		a.print(entry.Path)
	}
	return 0
}

func (a App) runPlanCurrent(flags globalFlags, args []string) int {
	if len(args) > 0 {
		return a.fail(flags, errors.New("plan current does not accept additional arguments"))
	}

	entry, content, err := plan.Current(a.CWD)
	if err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(map[string]any{
			"path":    entry.Path,
			"content": content,
		})
	}
	a.print(strings.TrimRight(content, "\n"))
	return 0
}

func (a App) runPlanCreate(flags globalFlags, args []string) int {
	title := strings.Join(args, " ")
	entry, err := plan.Create(a.CWD, title)
	if err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(entry)
	}
	a.print(entry.Path)
	return 0
}

func (a App) runLog(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: log <recent|create> ..."))
	}
	switch args[0] {
	case "recent":
		return a.runLogRecent(flags, args[1:])
	case "create":
		return a.runLogCreate(flags, args[1:])
	default:
		return a.fail(flags, errors.New("expected: log <recent|create> ..."))
	}
}

func (a App) runLogRecent(flags globalFlags, args []string) int {
	limit := 3
	if len(args) > 1 {
		return a.fail(flags, errors.New("expected: log recent [N]"))
	}
	if len(args) == 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 0 {
			return a.fail(flags, fmt.Errorf("invalid log count %q", args[0]))
		}
		limit = n
	}

	entries, err := logutil.Recent(a.CWD, limit)
	if err != nil {
		return a.fail(flags, err)
	}

	for i := range entries {
		entries[i].Path = epic.RelativePath(a.CWD, entries[i].Path)
	}
	if flags.JSON {
		return a.emitJSON(entries)
	}
	for i, entry := range entries {
		if i > 0 {
			a.print("---")
		}
		a.print(strings.TrimRight(entry.Content, "\n"))
	}
	return 0
}

func (a App) runLogCreate(flags globalFlags, args []string) int {
	path, err := logutil.Create(a.CWD, strings.Join(args, " "))
	if err != nil {
		return a.fail(flags, err)
	}
	relative := epic.RelativePath(a.CWD, path)
	if flags.JSON {
		return a.emitJSON(map[string]any{"path": relative})
	}
	a.print(relative)
	return 0
}

func (a App) runCron(flags globalFlags, args []string) int {
	if len(args) != 1 || args[0] != "validate" {
		return a.fail(flags, errors.New("expected: cron validate"))
	}

	diagnostics, err := cron.Validate(a.CWD)
	if err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		exitCode := 0
		for _, diagnostic := range diagnostics {
			if diagnostic.Level == "error" {
				exitCode = 1
				break
			}
		}
		if code := a.emitJSON(diagnostics); code != 0 {
			return code
		}
		return exitCode
	}
	if len(diagnostics) == 0 {
		a.print("cron.d is valid.")
		return 0
	}
	for _, diagnostic := range diagnostics {
		a.print(fmt.Sprintf("%s: %s (%s)", strings.ToUpper(diagnostic.Level), diagnostic.Message, diagnostic.Path))
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == "error" {
			return 1
		}
	}
	return 0
}

func (a App) resolvePackageTarget(arg string) (string, error) {
	if arg == "" {
		return a.CWD, nil
	}
	if exists(arg) {
		return arg, nil
	}
	if path := filepath.Join(a.CWD, arg); exists(path) {
		return path, nil
	}
	return arg, nil
}

func (a App) resolvePackageReference(arg string) (string, workspace.InstallRecord, error) {
	if arg == "" {
		pkg, _, err := epic.Validate(a.CWD)
		if err == nil && (pkg.SkillPath != "" || pkg.EpicPath != "") {
			return a.CWD, workspace.InstallRecord{}, nil
		}

		installs, err := workspace.LoadInstalls(a.CWD)
		if err == nil && len(installs) == 1 {
			record := installs[0]
			return filepath.Join(a.CWD, filepath.FromSlash(record.InstalledDir)), record, nil
		}
		return "", workspace.InstallRecord{}, errors.New("could not determine a package from the current directory; pass a path or installed slug")
	}

	if exists(arg) {
		return arg, workspace.InstallRecord{}, nil
	}

	localPath := filepath.Join(a.CWD, arg)
	if exists(localPath) {
		return localPath, workspace.InstallRecord{}, nil
	}

	record, ok, err := workspace.FindInstall(a.CWD, arg)
	if err != nil {
		return "", workspace.InstallRecord{}, err
	}
	if ok {
		return filepath.Join(a.CWD, filepath.FromSlash(record.InstalledDir)), record, nil
	}

	return "", workspace.InstallRecord{}, fmt.Errorf("could not resolve %q as a local path or installed slug", arg)
}

func (a App) printUsage() {
	a.print("Usage: epics [--json] [--quiet] [--yes] <command>")
	a.print(fmt.Sprintf("Commands: init, upgrade-skill-footer, install, validate, info, status, resume, doctor, host <setup|doctor> <%s>, state, plan, log, cron, daemon, workspace, route, run", strings.Join(hosts.Supported(), "|")))
}

func parseGlobalFlags(args []string) (globalFlags, []string, error) {
	var flags globalFlags
	var rest []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			flags.JSON = true
		case "--quiet":
			flags.Quiet = true
		case "--yes":
			flags.Yes = true
		default:
			rest = args[i:]
			return flags, rest, nil
		}
	}

	return flags, rest, nil
}

type installArgs struct {
	Host   string
	Source string
}

func parseInstallArgs(args []string) (installArgs, error) {
	var result installArgs
	hostSpecified := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--host":
			if i+1 >= len(args) {
				return installArgs{}, errors.New("install flag --host requires a value")
			}
			hostSpecified = true
			result.Host = args[i+1]
			i++
		case strings.HasPrefix(arg, "--host="):
			hostSpecified = true
			result.Host = strings.TrimPrefix(arg, "--host=")
		case strings.HasPrefix(arg, "--"):
			return installArgs{}, fmt.Errorf("unknown install flag %q", arg)
		default:
			if result.Source != "" {
				return installArgs{}, errors.New("install requires exactly one <repo-path> or local directory")
			}
			result.Source = arg
		}
	}

	if result.Source == "" {
		return installArgs{}, errors.New("install requires exactly one <repo-path> or local directory")
	}
	if !hostSpecified {
		return result, nil
	}
	if strings.TrimSpace(result.Host) == "" {
		return installArgs{}, errors.New("install flag --host requires a non-empty value")
	}
	return result, nil
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func titleFromSlug(slug string) string {
	parts := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(slug))
	for i := range parts {
		if len(parts[i]) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	if len(parts) == 0 {
		return "New Epic"
	}
	return strings.Join(parts, " ")
}

func requireAtMostOneArg(args []string) (string, error) {
	if len(args) > 1 {
		return "", errors.New("expected at most one argument")
	}
	if len(args) == 0 {
		return "", nil
	}
	return args[0], nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a App) emitJSON(payload any) int {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintln(a.Stderr, err.Error())
		return 1
	}
	raw = append(raw, '\n')
	_, _ = a.Stdout.Write(raw)
	return 0
}

func (a App) print(line string) {
	if a.Stdout == nil {
		return
	}
	fmt.Fprintln(a.Stdout, line)
}

func (a App) selectHost() (string, error) {
	supportedHosts := hosts.Supported()
	if len(supportedHosts) == 0 {
		return "", errors.New("no supported hosts are available")
	}
	if a.IsInteractive == nil || !a.IsInteractive() {
		return "", errors.New("install requires --host <host> when stdin is not interactive")
	}

	a.print("Select host:")
	for index, host := range supportedHosts {
		a.print(fmt.Sprintf("  %d. %s", index+1, host))
	}
	if a.Stdout != nil {
		fmt.Fprintf(a.Stdout, "Host [%s]: ", supportedHosts[0])
	}

	reader := bufio.NewReader(a.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	selection := strings.TrimSpace(line)
	if selection == "" {
		return supportedHosts[0], nil
	}
	if choice, err := strconv.Atoi(selection); err == nil {
		if choice >= 1 && choice <= len(supportedHosts) {
			return supportedHosts[choice-1], nil
		}
		return "", fmt.Errorf("invalid host selection %q", selection)
	}
	for _, host := range supportedHosts {
		if selection == host {
			return host, nil
		}
	}
	return "", fmt.Errorf("unsupported host %q", selection)
}

func (a App) printHostSetupResult(host string, result hostapi.Result) {
	label := host
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}
	a.print(fmt.Sprintf("%s workspace setup complete.", label))
	if len(result.Created) > 0 {
		sort.Strings(result.Created)
		a.print("Created:")
		for _, path := range result.Created {
			a.print("  - " + path)
		}
	}
	if len(result.Unchanged) > 0 {
		sort.Strings(result.Unchanged)
		a.print("Preserved existing:")
		for _, path := range result.Unchanged {
			a.print("  - " + path)
		}
	}
	if len(result.Skipped) > 0 {
		sort.Strings(result.Skipped)
		a.print("Skipped (content differs):")
		for _, path := range result.Skipped {
			a.print("  - " + path)
		}
	}
}

func (a App) fail(flags globalFlags, err error) int {
	if flags.JSON {
		_ = a.emitJSON(map[string]any{"error": err.Error()})
		return 1
	}
	if a.Stderr != nil {
		fmt.Fprintln(a.Stderr, "error:", err.Error())
	}
	return 1
}

func stdinIsInteractive(reader io.Reader) bool {
	if os.Getenv("EPICS_FORCE_INTERACTIVE") == "1" {
		return true
	}
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
