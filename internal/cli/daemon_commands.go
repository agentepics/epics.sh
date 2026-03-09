package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agentepics/epics.sh/internal/daemon"
	daemonservice "github.com/agentepics/epics.sh/internal/daemon/service"
	daemonstore "github.com/agentepics/epics.sh/internal/daemon/store"
	"github.com/agentepics/epics.sh/internal/doctor"
)

func (a App) runDaemon(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: daemon <install|uninstall|start|stop|restart|status|logs|doctor>"))
	}
	switch args[0] {
	case "install":
		return a.runDaemonService(flags, "install")
	case "uninstall":
		return a.runDaemonService(flags, "uninstall")
	case "start":
		return a.runDaemonService(flags, "start")
	case "stop":
		return a.runDaemonService(flags, "stop")
	case "restart":
		return a.runDaemonService(flags, "restart")
	case "status":
		return a.runDaemonStatus(flags)
	case "logs":
		return a.runDaemonLogs(flags, args[1:])
	case "doctor":
		return a.runDaemonDoctor(flags)
	default:
		return a.fail(flags, errors.New("expected: daemon <install|uninstall|start|stop|restart|status|logs|doctor>"))
	}
}

func (a App) runDaemonWorkspace(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: workspace <register|list|inspect>"))
	}
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	switch args[0] {
	case "register":
		payload, err := parseWorkspaceRegisterArgs(a.CWD, args[1:])
		if err != nil {
			return a.fail(flags, err)
		}
		var record daemonstore.WorkspaceRecord
		if err := client.Call(context.Background(), "workspace.register", payload, &record); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(record)
		}
		a.print(fmt.Sprintf("%s\t%s\t%s\t%s", record.ID, record.DisplayName, record.Health, record.Path))
		return 0
	case "list":
		var items []daemonstore.WorkspaceRecord
		if err := client.Call(context.Background(), "workspace.list", map[string]any{}, &items); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(items)
		}
		for _, item := range items {
			a.print(fmt.Sprintf("%s\t%s\t%s\t%s", item.ID, item.DisplayName, item.Health, item.Path))
		}
		return 0
	case "inspect":
		if len(args) != 2 {
			return a.fail(flags, errors.New("expected: workspace inspect <workspace-id>"))
		}
		var record daemonstore.WorkspaceRecord
		if err := client.Call(context.Background(), "workspace.inspect", map[string]string{"workspaceId": args[1]}, &record); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(record)
		}
		a.print(fmt.Sprintf("Workspace: %s", record.ID))
		a.print(fmt.Sprintf("Name: %s", record.DisplayName))
		a.print(fmt.Sprintf("Path: %s", record.Path))
		a.print(fmt.Sprintf("Health: %s", record.Health))
		if record.HealthMessage != "" {
			a.print(fmt.Sprintf("Health message: %s", record.HealthMessage))
		}
		return 0
	default:
		return a.fail(flags, errors.New("expected: workspace <register|list|inspect>"))
	}
}

func (a App) runDaemonRoute(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: route <upsert|list|inspect|disable|enable>"))
	}
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	switch args[0] {
	case "upsert":
		payload, err := parseRouteUpsertArgs(args[1:])
		if err != nil {
			return a.fail(flags, err)
		}
		var route daemonstore.RouteRecord
		if err := client.Call(context.Background(), "route.upsert", payload, &route); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(route)
		}
		a.print(fmt.Sprintf("%s\t%s\t%s\t%s", route.ID, route.Type, route.WorkspaceID, route.EpicSlug))
		return 0
	case "list":
		var routes []daemonstore.RouteRecord
		if err := client.Call(context.Background(), "route.list", map[string]any{}, &routes); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(routes)
		}
		for _, route := range routes {
			a.print(fmt.Sprintf("%s\t%s\t%s\t%s", route.ID, route.Type, boolLabel(route.Enabled), route.SelectedAdapter))
		}
		return 0
	case "inspect":
		if len(args) != 2 {
			return a.fail(flags, errors.New("expected: route inspect <route-id>"))
		}
		var route daemonstore.RouteRecord
		if err := client.Call(context.Background(), "route.inspect", map[string]string{"routeId": args[1]}, &route); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(route)
		}
		a.print(fmt.Sprintf("Route: %s", route.ID))
		a.print(fmt.Sprintf("Type: %s", route.Type))
		a.print(fmt.Sprintf("Workspace: %s", route.WorkspaceID))
		a.print(fmt.Sprintf("Epic: %s", route.EpicSlug))
		a.print(fmt.Sprintf("Enabled: %s", boolLabel(route.Enabled)))
		if route.SelectedAdapter != "" {
			a.print(fmt.Sprintf("Adapter: %s", route.SelectedAdapter))
		}
		return 0
	case "enable", "disable":
		if len(args) != 2 {
			return a.fail(flags, fmt.Errorf("expected: route %s <route-id>", args[0]))
		}
		action := "route." + args[0]
		var route daemonstore.RouteRecord
		if err := client.Call(context.Background(), action, map[string]string{"routeId": args[1]}, &route); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(route)
		}
		a.print(fmt.Sprintf("%s\t%s", route.ID, boolLabel(route.Enabled)))
		return 0
	default:
		return a.fail(flags, errors.New("expected: route <upsert|list|inspect|disable|enable>"))
	}
}

func (a App) runDaemonRun(flags globalFlags, args []string) int {
	if len(args) == 0 {
		return a.fail(flags, errors.New("expected: run <list|inspect>"))
	}
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	switch args[0] {
	case "list":
		payload, err := parseRunListArgs(args[1:])
		if err != nil {
			return a.fail(flags, err)
		}
		var runs []daemonstore.RunRecord
		if err := client.Call(context.Background(), "run.list", payload, &runs); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(runs)
		}
		for _, run := range runs {
			a.print(fmt.Sprintf("%s\t%s\t%s\t%s", run.ID, run.RouteID, run.Outcome, run.EnqueuedAt))
		}
		return 0
	case "inspect":
		if len(args) != 2 {
			return a.fail(flags, errors.New("expected: run inspect <run-id>"))
		}
		var run daemonstore.RunRecord
		if err := client.Call(context.Background(), "run.inspect", map[string]string{"runId": args[1]}, &run); err != nil {
			return a.fail(flags, err)
		}
		if flags.JSON {
			return a.emitJSON(run)
		}
		a.print(fmt.Sprintf("Run: %s", run.ID))
		a.print(fmt.Sprintf("Route: %s", run.RouteID))
		a.print(fmt.Sprintf("Outcome: %s", run.Outcome))
		if run.FailureReason != "" {
			a.print(fmt.Sprintf("Failure: %s", run.FailureReason))
		}
		return 0
	default:
		return a.fail(flags, errors.New("expected: run <list|inspect>"))
	}
}

func (a App) runDaemonService(flags globalFlags, action string) int {
	home, err := daemonstore.ResolveHome()
	if err != nil {
		return a.fail(flags, err)
	}
	binaryPath, err := daemonservice.ResolveBinary()
	if err != nil {
		return a.fail(flags, err)
	}
	manager := daemonservice.NewManager(home, binaryPath)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch action {
	case "install":
		err = manager.Install(ctx)
	case "uninstall":
		err = manager.Uninstall(ctx)
	case "start":
		err = manager.Start(ctx)
	case "stop":
		err = manager.Stop(ctx)
	case "restart":
		err = manager.Restart(ctx)
	}
	if err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(map[string]any{"ok": true, "action": action})
	}
	a.print("daemon " + action + " complete")
	return 0
}

func (a App) runDaemonStatus(flags globalFlags) int {
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	var status map[string]any
	if err := client.Call(context.Background(), "daemon.status", map[string]any{}, &status); err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(status)
	}
	a.print(fmt.Sprintf("Status: %v", status["status"]))
	a.print(fmt.Sprintf("Started: %v", status["startedAt"]))
	a.print(fmt.Sprintf("Webhook: %v", status["webhookHTTPAddr"]))
	a.print(fmt.Sprintf("Socket: %v", status["adminSocketPath"]))
	return 0
}

func (a App) runDaemonLogs(flags globalFlags, args []string) int {
	limit := 100
	if len(args) > 1 {
		return a.fail(flags, errors.New("expected: daemon logs [N]"))
	}
	if len(args) == 1 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			return a.fail(flags, fmt.Errorf("invalid log count %q", args[0]))
		}
		limit = n
	}
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	var result struct {
		Path  string   `json:"path"`
		Lines []string `json:"lines"`
	}
	if err := client.Call(context.Background(), "daemon.logs", map[string]int{"limit": limit}, &result); err != nil {
		return a.fail(flags, err)
	}
	if flags.JSON {
		return a.emitJSON(result)
	}
	for _, line := range result.Lines {
		a.print(line)
	}
	return 0
}

func (a App) runDaemonDoctor(flags globalFlags) int {
	client, err := daemon.NewDefaultClient()
	if err != nil {
		return a.fail(flags, err)
	}
	var result doctor.Result
	if err := client.Call(context.Background(), "daemon.doctor", map[string]any{}, &result); err != nil {
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

func parseWorkspaceRegisterArgs(cwd string, args []string) (map[string]string, error) {
	path := cwd
	displayName := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return nil, errors.New("workspace register flag --name requires a value")
			}
			displayName = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--") {
				return nil, fmt.Errorf("unknown workspace register flag %q", args[i])
			}
			path = args[i]
		}
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	return map[string]string{"path": path, "displayName": displayName}, nil
}

func parseRouteUpsertArgs(args []string) (map[string]any, error) {
	payload := map[string]any{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--reselect-adapter":
			payload["reselectAdapter"] = true
		default:
			if !strings.HasPrefix(arg, "--") {
				return nil, fmt.Errorf("unexpected route upsert argument %q", arg)
			}
			if i+1 >= len(args) {
				return nil, fmt.Errorf("route upsert flag %s requires a value", arg)
			}
			key := strings.TrimPrefix(arg, "--")
			value := args[i+1]
			i++
			switch key {
			case "id":
				payload["routeId"] = value
			case "type":
				payload["type"] = value
			case "workspace":
				payload["workspaceId"] = value
			case "epic":
				payload["epicSlug"] = value
			case "provider":
				payload["provider"] = value
			case "endpoint":
				payload["endpointKey"] = value
			case "job":
				payload["jobName"] = value
			case "cron":
				payload["cronExpr"] = value
			case "preferred-adapter":
				payload["preferredAdapter"] = value
			case "pinned-adapter":
				payload["pinnedAdapter"] = value
			case "auth":
				payload["authMode"] = value
			case "secret":
				payload["secretValue"] = value
			case "hmac-header":
				payload["hmacHeader"] = value
			case "overlap":
				payload["overlapPolicy"] = value
			default:
				return nil, fmt.Errorf("unknown route upsert flag %q", arg)
			}
		}
	}
	if _, ok := payload["reselectAdapter"]; !ok {
		payload["reselectAdapter"] = false
	}
	return payload, nil
}

func parseRunListArgs(args []string) (map[string]any, error) {
	payload := map[string]any{"limit": 100}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if i+1 >= len(args) {
			return nil, fmt.Errorf("run list flag %q requires a value", arg)
		}
		value := args[i+1]
		i++
		switch arg {
		case "--route":
			payload["routeId"] = value
		case "--workspace":
			payload["workspaceId"] = value
		case "--limit":
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid run limit %q", value)
			}
			payload["limit"] = n
		default:
			return nil, fmt.Errorf("unknown run list flag %q", arg)
		}
	}
	return payload, nil
}

func boolLabel(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}
