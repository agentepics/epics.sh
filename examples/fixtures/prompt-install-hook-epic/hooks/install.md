---
enabled: true
type: prompt
timeout: 300
---

Write the file `runtime/prompt-hook-output.json` in the current directory.
Its full contents must be exactly:

```json
{"status":"ok","trigger":"install","epic_id":"prompt-install-hook-epic"}
```

If the `runtime` directory does not exist, create it.
Do not modify any other files.
