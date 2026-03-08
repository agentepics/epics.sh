package hostapi

type Result struct {
	Created   []string `json:"created,omitempty"`
	Unchanged []string `json:"unchanged,omitempty"`
}

type Adapter interface {
	Name() string
	InstallDir(cwd, slug string) string
	Setup(cwd string) (Result, error)
}
