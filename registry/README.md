# Registry

This directory will hold the Git-backed Epic registry.

Structure:

- `epics/` for listing metadata and generated data for published Epics
- `schemas/` for metadata and manifest schemas

The registry is not replaced by the separate `agentepics/epics` repository.

Recommended split:

- `agentepics/epics` holds public curated Epic source repositories authored by
  the project
- `registry/` holds the index, schemas, compatibility metadata, install
  metadata, digest data, and other source-of-truth data used by the website and
  CLI
