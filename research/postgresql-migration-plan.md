# PostgreSQL Migration Plan for OSV.dev

## Table of Contents
1. [Overview](#overview)
2. [Current Architecture](#current-architecture)
3. [Environment Configuration](#environment-configuration)
4. [Backend Storage Configuration](#backend-storage-configuration)
5. [Phase 1: Understand Storage Abstraction](#phase-1-understand-storage-abstraction)
6. [Phase 2: Create PostgreSQL Implementation](#phase-2-create-postgresql-implementation)
7. [Phase 3: Key Design Decisions](#phase-3-key-design-decisions)
8. [Phase 4: Implementation Steps](#phase-4-implementation-steps)
9. [Phase 5: Testing Strategy](#phase-5-testing-strategy)
10. [Phase 6: Gradual Rollout](#phase-6-gradual-rollout)
11. [Garage S3 Configuration](#garage-s3-configuration)
12. [Critical Files Reference](#critical-files-reference)

---

## Overview

The osv.dev project currently uses **Google Cloud Datastore** for metadata/indexes and **Google Cloud Storage (GCS)** for full vulnerability records. This document outlines a comprehensive migration plan to support **PostgreSQL** as an alternative backend database while maintaining compatibility with the existing architecture.

**Key Insight from AGENTS.md (Line 15-16):**
> "There are long-term plans to migrate the database backend to PostgreSQL, but this is not yet concrete. **Rule**: Any new Go code interacting with the database must be abstracted behind interfaces to facilitate this eventual migration. Much of this abstraction is already in place."

This plan leverages the **existing interface-based abstractions** already present in the codebase.

---

## Current Architecture

### Hybrid Storage Model

OSV.dev uses a **two-tier storage approach**:

┌─────────────────────────────────────┐ │ Metadata & Indexes │ │ (Google Cloud Datastore) │ │ - Vulnerability metadata │ │ - Affected versions/commits │ │ - Relations (aliases, related) │ │ - Source repositories │ └─────────────────────────────────────┘ ↓ ┌─────────────────────────────────────┐ │ Full Vulnerability Records │ │ (Google Cloud Storage - GCS) │ │ - Protocol buffer blobs (.pb files)│ │ - Path: all/pb/{id}.pb │ └─────────────────────────────────────┘

### Datastore Entities

| Entity | Kind | Purpose | Example Fields |
|--------|------|---------|-----------------|
| **Vulnerability** | `Vulnerability` | Index metadata | `source_id`, `modified`, `modified_raw`, `is_withdrawn` |
| **AffectedVersions** | `AffectedVersions` | Package matching | `ecosystem`, `name`, `versions`, `coarse_min`, `coarse_max` |
| **AffectedCommits** | `AffectedCommits` | Commit matching | `bug_id` (vuln ID), `commits`, `page` (paginated) |
| **ListedVulnerability** | `ListedVulnerability` | Website /list page | `id`, `summary`, `packages`, `severities` |
| **SourceRepository** | `SourceRepository` | Data source config | `name`, `id_prefixes`, `type`, `git`/`bucket`/`rest` |
| **AliasGroup** | `AliasGroup` | Alias relationships | `bug_ids`, `last_modified` |
| **UpstreamGroup** | `UpstreamGroup` | Upstream relations | `db_id`, `upstream_ids`, `upstream_hierarchy` |
| **RelatedGroup** | `RelatedGroup` | Related vulnerabilities | `related_ids`, `modified` |
| **RepoIndex** | `RepoIndex` | Git index data | `name`, `commit`, `tag`, `file_exts`, `file_count` |
| **UbuntuPackageMapping** | `UbuntuPackageMapping` | Ubuntu binary→source | Key: `<ecosystem>:<binary_name>`, Value: `source_names` |

---

## Environment Configuration

### API Server Configuration (`go/cmd/api/main.go`)

The API server reads configuration from **environment variables**:

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `GOOGLE_CLOUD_PROJECT` | ✓ | (none) | GCP project ID for Datastore/GCS access |
| `DATASTORE_DATABASE_ID` | ✗ | `""` (default DB) | Specific Datastore database to use |
| `OSV_VULNERABILITIES_BUCKET` | ✓ | (none) | GCS bucket name for storing `.pb` files |
| `OSV_DB_BATCH_TIMEOUT` | ✗ | (default) | Duration for batching DB queries (e.g., `100ms`) |
| `OSV_DB_BATCH_MAX_SIZE` | ✗ | 100 | Max vulnerability IDs per batch query |
| `FAILED_TASKS_TOPIC` | ✗ | (none) | Pub/Sub topic for recovery tasks |
| `OSV_VERBOSE_LOGGING` | ✗ | `false` | Enable verbose logging |
| `OSV_HEALTH_CHECK_INTERVAL` | ✗ | (default) | Duration between health checks |
| `OSV_HEALTH_CHECK_THRESHOLD` | ✗ | (default) | Consecutive failures before marking unhealthy |
| `PORT` | ✗ | `8000` | Server port |

### Initialization Code (`go/cmd/api/main.go`, lines 52-106)

```go
// Read environment variables
project := os.Getenv("GOOGLE_CLOUD_PROJECT")
datastoreID := os.Getenv("DATASTORE_DATABASE_ID")
vulnBucket := os.Getenv("OSV_VULNERABILITIES_BUCKET")

// Create Datastore client
dbClient, err := datastore.NewClientWithDatabase(ctx, project, datastoreID, 
    datastore.WithIgnoreFieldMismatch())

// Create GCS client
gcsClient, err := storage.NewClient(ctx)

// Create store implementations
vulnStore := db.NewVulnerabilityStore(db.VulnStoreConfig{
    Client:           dbClient,
    GCS:              clients.NewGCSClient(gcsClient, vulnBucket),
    BatchTimeout:     batchTimeout,
    BatchMaxElements: batchMaxElements,
})
relationsStore := db.NewRelationsStore(dbClient)
importFindingsStore := db.NewImportFindingsStore(dbClient, nil, "", "")
repoIndexStore := db.NewRepoIndexStore(dbClient)
ubuntuPackageMappingStore := db.NewUbuntuPackageMappingStore(dbClient)