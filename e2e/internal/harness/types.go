package harness

type CopySpec struct {
	From string
	To   string
}

type Step struct {
	Name              string
	Program           string
	Args              []string
	Workdir           string
	Stdin             string
	Env               map[string]string
	PassEnv           []string
	ExpectExitCode    int
	StdoutEquals      string
	StdoutContains    []string
	StdoutNotContains []string
	StderrEquals      string
	StderrContains    []string
	StderrNotContains []string
}

type FileAssertion struct {
	Path        string
	MustExist   bool
	Contains    []string
	NotContains []string
	Equals      string
}

type Scenario struct {
	Name         string
	Description  string
	Tags         []string
	ImageProfile string
	RequiredEnv  []string
	Copies       []CopySpec
	SeedFiles    map[string]string
	Steps        []Step
	Files        []FileAssertion
}

type StepResult struct {
	Name               string   `json:"name"`
	Command            []string `json:"command"`
	ExitCode           int      `json:"exitCode"`
	Stdout             string   `json:"stdout"`
	Stderr             string   `json:"stderr"`
	Passed             bool     `json:"passed"`
	LogPath            string   `json:"logPath,omitempty"`
	EventLogPath       string   `json:"eventLogPath,omitempty"`
	BeforeManifestPath string   `json:"beforeManifestPath,omitempty"`
	AfterManifestPath  string   `json:"afterManifestPath,omitempty"`
	StartedAt          string   `json:"startedAt,omitempty"`
	EndedAt            string   `json:"endedAt,omitempty"`
	DurationMillis     int64    `json:"durationMillis,omitempty"`
	Error              string   `json:"error,omitempty"`
}

type ScenarioResult struct {
	Name                 string       `json:"name"`
	Passed               bool         `json:"passed"`
	Skipped              bool         `json:"skipped,omitempty"`
	ArtifactDir          string       `json:"artifactDir"`
	ScenarioLogPath      string       `json:"scenarioLogPath,omitempty"`
	ScenarioEventLogPath string       `json:"scenarioEventLogPath,omitempty"`
	InitialManifestPath  string       `json:"initialManifestPath,omitempty"`
	PreparedManifestPath string       `json:"preparedManifestPath,omitempty"`
	FinalManifestPath    string       `json:"finalManifestPath,omitempty"`
	Steps                []StepResult `json:"steps"`
	Error                string       `json:"error,omitempty"`
}

type Summary struct {
	RunID           string            `json:"runId"`
	ImageTags       map[string]string `json:"imageTags,omitempty"`
	ArtifactRoot    string            `json:"artifactRoot"`
	RunLogPath      string            `json:"runLogPath,omitempty"`
	RunEventLogPath string            `json:"runEventLogPath,omitempty"`
	PassedCount     int               `json:"passedCount"`
	FailedCount     int               `json:"failedCount"`
	SkippedCount    int               `json:"skippedCount,omitempty"`
	ScenarioCount   int               `json:"scenarioCount"`
	Results         []ScenarioResult  `json:"results"`
}

type WorkspaceManifestEntry struct {
	Path   string `json:"path"`
	IsDir  bool   `json:"isDir"`
	Mode   string `json:"mode"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
}

type WorkspaceManifest struct {
	GeneratedAt string                   `json:"generatedAt"`
	Root        string                   `json:"root"`
	Entries     []WorkspaceManifestEntry `json:"entries"`
}
