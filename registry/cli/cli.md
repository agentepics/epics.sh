---
title: epics CLI
description: Install the epics CLI, download platform builds, and read the changelog.
---

# epics CLI

The website reflects the real CLI, not a separate web-only workflow. Install the binary, then use `epics install`, `epics validate`, `epics resume`, and host setup commands from the same model the directory renders.

## Install

### macOS/Linux

```bash
curl -fsSL https://epics.sh/install.sh | sh
```

### Windows

```powershell
iwr https://epics.sh/install.ps1 -useb | iex
```

## Downloads

### macOS (Apple Silicon)

- Target: `darwin-arm64`
- Artifact: #
- Checksum: `sha256:7d667a260bf0bc21d8c43b8d9487691911c502b7d0a50a3b4d3a36100b252308`

### macOS (Intel)

- Target: `darwin-amd64`
- Artifact: #
- Checksum: `sha256:1f6609e7400568d13f0e2daf5c4cb9db39d50f6be5d11c51bf4067ff6f5c88c5`

### Linux (x64)

- Target: `linux-amd64`
- Artifact: #
- Checksum: `sha256:d8ace9c5e4c098ecb4f9fd1d136ead3eeff6a4f4f9d977434e32d009e84f16ad`

### Windows (x64)

- Target: `windows-amd64`
- Artifact: #
- Checksum: `sha256:bcb5b383fb876cc4bbbe4f8658257fcae8559f594790e4d0471a7ccf7e67159d`

## Changelog

### 0.1.0 (Mar 7, 2026)

- First public web preview with registry-driven Epic pages
- Cross-host support labeling and install command generation
- CLI downloads, releases, changelog, and docs surfaces

### 0.0.1 (Mar 5, 2026)

- CLI scaffold
- Initial host research docs
- Roadmap and website product definition
