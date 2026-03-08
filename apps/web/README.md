# Web App

This directory will hold the `epics.sh` website.

The website is both:

- the public Agent Epic directory
- the official home for the `epics` CLI

The website should render source-derived content from `registry/` and release
metadata rather than hardcoded package data.

Public curated sample Epics authored by the project should come from the
separate `agentepics/epics` repository, while this app renders their listing
and install metadata through the local registry.

Planned stack:

- Astro
- TypeScript
- MDX
- static generation from `registry/`

Current first-version surface:

- landing page
- directory index with basic filters
- Epic detail pages with top-of-page install commands
- compatibility page
- CLI install, downloads, releases, and changelog pages
- docs overview and manual pages

The website should render:

- the public Epic directory
- installation flows
- CLI downloads, releases, changelog, and manual
- compatibility information
- project documentation
