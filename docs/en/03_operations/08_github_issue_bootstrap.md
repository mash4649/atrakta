# 08. GitHub Issue Bootstrap (Parity / Extension / Brownfield)

This operation document turns the backlog drafts into runnable GitHub issue creation.

## Source files

- `.github/issue_drafts/parity_extension_brownfield.json`
- `.github/issue_drafts/parity_extension_brownfield_epic_map.json`
- `scripts/dev/create_parity_issue_pack.sh`

## What the script does

1. Reads draft issues (1-28), labels, milestones, dependencies.
2. Merges Epic/Story alignment and extra tasks from the epic map.
3. Creates missing labels and milestones (unless skipped).
4. Creates issues in order.
5. Adds `Depends on:` comments using created issue numbers.

## Dry run

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo> \
  --dry-run
```

## Create all issues

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo>
```

## Create subset

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo> \
  --start-id 9 \
  --limit 8
```

## Useful flags

- `--skip-label-setup`: use existing labels as-is
- `--skip-milestone-setup`: use existing milestones as-is
- `--draft <path>`: custom draft file
- `--epic-map <path>`: custom epic/story map

## Output artifacts

Created issue ID mapping is stored in:

- `.github/issue_drafts/out/issue-map-YYYYMMDD-HHMMSS.json`

Use it to map draft IDs to real GitHub issue numbers.

## Add issues to GitHub Project

Create or reuse the backlog project and add all draft issues:

```bash
./scripts/dev/populate_parity_project.sh \
  --repo <owner/repo> \
  --owner <owner> \
  --project-title "Atrakta Parity / Extension / Brownfield Backlog" \
  --issue-map .github/issue_drafts/out/issue-map-YYYYMMDD-HHMMSS.json
```

Dry run:

```bash
./scripts/dev/populate_parity_project.sh \
  --repo <owner/repo> \
  --dry-run
```

## Sync Project Fields

Sync `priority:P0/P1/P2` labels into the Project `Priority` single-select field:

```bash
./scripts/dev/sync_parity_project_fields.sh \
  --owner <owner> \
  --project-number <n>
```

Dry run:

```bash
./scripts/dev/sync_parity_project_fields.sh \
  --owner <owner> \
  --project-number <n> \
  --dry-run
```

## Recommended Milestone Views

Create saved views in the Project UI with these filters:

- `milestone:"Milestone 1: Schema & Docs Foundation"`
- `milestone:"Milestone 2: Core Projection"`
- `milestone:"Milestone 3: Brownfield & Doctor"`
- `milestone:"Milestone 4: Extensions & Runtime"`
- `milestone:"Milestone 5: Operations & Quality"`
