package harness

import (
	"encoding/json"
	"fmt"
	"strings"
)

func runLiveSessionStep(imageTag, imageProfile, workspaceDir, scenarioDir string, index int, step Step, log *operationLogger) StepResult {
	spec, err := withDefaultLiveSessionSpec(step.LiveSession)
	if err != nil {
		return StepResult{
			Name:    step.Name,
			Command: []string{"live-session"},
			Passed:  false,
			Error:   err.Error(),
		}
	}

	script, err := buildLiveSessionShell(spec)
	if err != nil {
		return StepResult{
			Name:    step.Name,
			Command: []string{"live-session"},
			Passed:  false,
			Error:   err.Error(),
		}
	}

	wrapped := step
	wrapped.LiveSession = nil
	wrapped.Program = "sh"
	wrapped.Args = []string{"-lc", script}
	wrapped.Stdin = ""
	return runStep(imageTag, imageProfile, workspaceDir, scenarioDir, index, wrapped, log)
}

func withDefaultLiveSessionSpec(spec *LiveSessionSpec) (LiveSessionSpec, error) {
	if spec == nil {
		return LiveSessionSpec{}, fmt.Errorf("live session step is missing configuration")
	}
	normalized := *spec
	if strings.TrimSpace(normalized.ArtifactDir) == "" {
		normalized.ArtifactDir = ".claude-live-session"
	}
	if strings.TrimSpace(normalized.BootstrapPrompt) == "" {
		normalized.BootstrapPrompt = "Respond exactly PREPARED"
	}
	if strings.TrimSpace(normalized.BootstrapExpect) == "" {
		normalized.BootstrapExpect = "PREPARED"
	}
	if len(normalized.Turns) == 0 {
		return LiveSessionSpec{}, fmt.Errorf("live session spec must define at least one turn")
	}
	for index, turn := range normalized.Turns {
		if strings.TrimSpace(turn.Name) == "" {
			return LiveSessionSpec{}, fmt.Errorf("live session turn %d is missing a name", index+1)
		}
		if strings.TrimSpace(turn.Prompt) == "" {
			return LiveSessionSpec{}, fmt.Errorf("live session turn %q is missing a prompt", turn.Name)
		}
		if strings.TrimSpace(turn.Expected) == "" {
			return LiveSessionSpec{}, fmt.Errorf("live session turn %q is missing an expected marker", turn.Name)
		}
		if strings.Contains(normalizeLiveSessionText(turn.Prompt), normalizeLiveSessionText(turn.Expected)) {
			return LiveSessionSpec{}, fmt.Errorf("live session turn %q prompt must not contain its expected marker", turn.Name)
		}
	}
	return normalized, nil
}

func normalizeLiveSessionText(value string) string {
	replacer := strings.NewReplacer("\r", "", "\n", "", "\t", "", " ", "")
	return strings.ToLower(replacer.Replace(value))
}

func buildLiveSessionShell(spec LiveSessionSpec) (string, error) {
	turnsJSON, err := json.MarshalIndent(spec.Turns, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`set -e
artifact_dir=%s
mkdir -p "$artifact_dir"
mkdir -p "$HOME" "$XDG_CONFIG_HOME"
cat > "$HOME/.claude.json" <<'JSON'
{
  "numStartups": 1,
  "hasCompletedOnboarding": true
}
JSON
cat > "$HOME/settings.json" <<'JSON'
{
  "skipDangerousModePermissionPrompt": true
}
JSON
cat > "$artifact_dir/session-turns.json" <<'JSON'
%s
JSON
claude -p %s --dangerously-skip-permissions --output-format text > "$artifact_dir/bootstrap.txt"
grep -Fq %s "$artifact_dir/bootstrap.txt"
EPICS_E2E_LIVE_SESSION_ARTIFACT_DIR="$artifact_dir" python3 - <<'PY'
import json
import os
import pty
import re
import select
import signal
import subprocess
import time
from pathlib import Path

artifact_dir = Path(os.environ["EPICS_E2E_LIVE_SESSION_ARTIFACT_DIR"])
raw_transcript_path = artifact_dir / "transcript.raw.txt"
clean_transcript_path = artifact_dir / "transcript.txt"
summary_path = artifact_dir / "session-summary.json"
turn_specs = json.loads((artifact_dir / "session-turns.json").read_text(encoding="utf-8"))

ansi_csi_re = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]")
ansi_osc_re = re.compile(r"\x1b\][^\x07]*(?:\x07|\x1b\\\\)")
other_escape_re = re.compile(r"\x1b[@-_]")

def strip_ansi(text: str) -> str:
    text = ansi_osc_re.sub("", text)
    text = ansi_csi_re.sub("", text)
    text = other_escape_re.sub("", text)
    return text.replace("\r", "\n")

def normalized(text: str) -> str:
    return re.sub(r"\s+", "", text).lower()

def send_keys(data: bytes, pause: float = 0.35) -> None:
    os.write(master, data)
    time.sleep(pause)

def is_ready_prompt(text: str) -> bool:
    normalized_text = normalized(text)
    return (
        "❯" in text
        and "quicksafetycheck" not in normalized_text
        and "doyouwanttousethisapikey" not in normalized_text
        and "bypasspermissionsmode" not in normalized_text
        and "selectloginmethod" not in normalized_text
        and "choosethetextstyle" not in normalized_text
    )

raw_chunks = []
clean_chunks = []

master, slave = pty.openpty()
env = os.environ.copy()
env["TERM"] = "dumb"
proc = subprocess.Popen(
    ["claude", "--dangerously-skip-permissions"],
    cwd=os.getcwd(),
    stdin=slave,
    stdout=slave,
    stderr=slave,
    env=env,
    text=False,
)
os.close(slave)

def record(data: bytes) -> str:
    raw = data.decode("utf-8", "replace")
    clean = strip_ansi(raw)
    raw_chunks.append(raw)
    clean_chunks.append(clean)
    raw_transcript_path.write_text("".join(raw_chunks), encoding="utf-8")
    clean_transcript_path.write_text("".join(clean_chunks), encoding="utf-8")
    return clean

def read_until(predicate, timeout: float) -> str:
    deadline = time.time() + timeout
    collected = []
    while time.time() < deadline:
        ready, _, _ = select.select([master], [], [], 0.2)
        if master in ready:
            try:
                data = os.read(master, 65536)
            except OSError:
                break
            if not data:
                break
            clean = record(data)
            collected.append(clean)
            joined = "".join(collected)
            if predicate(joined):
                return joined
    raise SystemExit("timed out waiting for interactive Claude output")

for _ in range(10):
    initial = read_until(
        lambda text: is_ready_prompt(text)
        or "quicksafetycheck" in normalized(text)
        or "doyouwanttousethisapikey" in normalized(text)
        or ("bypasspermissionsmode" in normalized(text) and "yes,iaccept" in normalized(text)),
        60,
    )
    normalized_initial = normalized(initial)
    if "quicksafetycheck" in normalized_initial:
        send_keys(b"\r")
        continue
    if "doyouwanttousethisapikey" in normalized_initial:
        send_keys(b"\x1b[A")
        send_keys(b"\r")
        continue
    if "yes,iaccept" in normalized_initial and "bypasspermissionsmode" in normalized_initial:
        send_keys(b"\x1b[B")
        send_keys(b"\r")
        continue
    if is_ready_prompt(initial):
        break
else:
    raise SystemExit("did not reach interactive Claude prompt")

turn_results = []
for spec in turn_specs:
    if proc.poll() is not None:
        raise SystemExit("claude process exited before " + spec["name"])
    send_keys(spec["prompt"].encode("utf-8") + b"\r", pause=0.1)
    expected_marker = normalized(spec["expected"])
    chunk = read_until(lambda text, marker=expected_marker: marker in normalized(text), 240)
    time.sleep(1)
    combined = chunk
    if not is_ready_prompt(chunk):
        try:
            combined += read_until(is_ready_prompt, 45)
        except SystemExit:
            pass
    turn_results.append(
        {
            "name": spec["name"],
            "expected": spec["expected"],
            "matched": expected_marker in normalized(combined),
        }
    )

report = {
    "pid": proc.pid,
    "artifactDir": str(artifact_dir),
    "transcriptPath": str(clean_transcript_path),
    "turnCount": len(turn_results),
    "turns": turn_results,
}
summary_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")

if proc.poll() is None:
    proc.send_signal(signal.SIGINT)
    time.sleep(1)
if proc.poll() is None:
    proc.terminate()
    time.sleep(1)
if proc.poll() is None:
    proc.kill()

for turn in turn_results:
    if not turn["matched"]:
        raise SystemExit(1)
PY

cat "$artifact_dir/session-summary.json"
cat "$artifact_dir/transcript.txt"
`, shellQuote(spec.ArtifactDir), string(turnsJSON), shellQuote(spec.BootstrapPrompt), shellQuote(spec.BootstrapExpect)), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
