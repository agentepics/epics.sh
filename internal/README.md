# Internal Packages

This directory will hold Go packages shared by the `epics` CLI.

Planned areas:

- Epic parsing and validation
- registry loading
- digest and install metadata handling
- install manifests
- host adapters
- runtime capability inspection

These packages should treat registry metadata as the canonical index and should
not assume public Epic source repos live inside this monorepo.
