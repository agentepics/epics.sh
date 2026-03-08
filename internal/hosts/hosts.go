package hosts

import (
	"fmt"
	"sort"

	"github.com/agentepics/epics.sh/internal/hostapi"
	"github.com/agentepics/epics.sh/internal/hosts/claude"
)

func Supported() []string {
	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Resolve(host string) (hostapi.Adapter, error) {
	adapter, ok := adapters[host]
	if !ok {
		return nil, fmt.Errorf("unsupported host %q", host)
	}
	return adapter, nil
}

func Setup(cwd, host string) (hostapi.Result, error) {
	adapter, err := Resolve(host)
	if err != nil {
		return hostapi.Result{}, err
	}
	return adapter.Setup(cwd)
}

var adapters = map[string]hostapi.Adapter{
	"claude": claude.Adapter{},
}
