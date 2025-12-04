# Publishing Guide for flog

## Prerequisites

1. GitHub account with the repository `github.com/ishk9/flog`
2. A separate repository for Homebrew tap: `github.com/ishk9/homebrew-tap`

---

## One-Time Setup

### Step 1: Create the Homebrew Tap Repository

1. Go to GitHub and create a new repository named `homebrew-tap`
2. Initialize it with a README
3. Create a `Formula` directory (GoReleaser will add formulas here)

```bash
# Clone and setup
git clone https://github.com/ishk9/homebrew-tap
cd homebrew-tap
mkdir Formula
touch Formula/.gitkeep
git add .
git commit -m "Initial setup"
git push
```

### Step 2: Create a Personal Access Token (PAT)

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name: `HOMEBREW_TAP_TOKEN`
4. Scopes: Select `repo` (full control of private repositories)
5. Generate and **copy the token**

### Step 3: Add Token to Repository Secrets

1. Go to your `flog` repository on GitHub
2. Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `HOMEBREW_TAP_TOKEN`
5. Value: Paste the token from Step 2

---

## Releasing a New Version

### Step 1: Update Version (if needed)

The version is automatically determined from the git tag.

### Step 2: Commit All Changes

```bash
git add .
git commit -m "Prepare for release v0.1.0"
git push origin main
```

### Step 3: Create and Push a Tag

```bash
git tag v0.1.0
git push origin v0.1.0
```

### Step 4: Watch the Magic ✨

1. GitHub Actions will automatically:
   - Run tests
   - Build binaries for all platforms
   - Create a GitHub Release with binaries
   - Update the Homebrew formula

2. Check progress at: `https://github.com/ishk9/flog/actions`

---

## What Gets Built

| Platform | Architecture | Binary Size |
|----------|--------------|-------------|
| Linux    | amd64        | ~2.4 MB     |
| Linux    | arm64        | ~2.4 MB     |
| macOS    | amd64 (Intel)| ~2.4 MB     |
| macOS    | arm64 (M1/M2)| ~2.4 MB     |
| Windows  | amd64        | ~2.5 MB     |
| Windows  | arm64        | ~2.5 MB     |

---

## Tracking Downloads & Usage

### GitHub Release Downloads

**GitHub Releases Page:**
- Go to `https://github.com/ishk9/flog/releases`
- Each release shows download counts per asset

**GitHub API:**
```bash
# Get download counts for all releases
curl -s https://api.github.com/repos/ishk9/flog/releases | \
  jq '.[] | {tag: .tag_name, downloads: [.assets[].download_count] | add}'
```

### Homebrew Analytics

Homebrew collects anonymous install statistics (users can opt-out).

**View install counts (after ~30 days):**
```bash
# This works for official taps; for custom taps, stats are limited
brew info ishk9/tap/flog
```

**Note:** Custom tap analytics are not publicly available. You'll mainly rely on:
- GitHub release download counts
- GitHub stars/forks
- Issues and discussions

### GitHub Insights

- **Traffic:** Repository → Insights → Traffic (views, clones)
- **Stars:** Track star growth over time
- **Forks:** See who's forking your project

### Add a Download Badge (Optional)

Add to your README:

```markdown
![GitHub Downloads](https://img.shields.io/github/downloads/ishk9/flog/total)
![GitHub Stars](https://img.shields.io/github/stars/ishk9/flog)
```

---

## Manual Release (Without Tags)

If you need to release manually:

```bash
# Install GoReleaser
brew install goreleaser

# Test the release (dry run)
goreleaser release --snapshot --clean

# Check the dist/ folder for built binaries
ls -la dist/
```

---

## Troubleshooting

### Release Failed

1. Check GitHub Actions logs
2. Ensure `HOMEBREW_TAP_TOKEN` is set correctly
3. Verify the tag format is `v*` (e.g., `v0.1.0`)

### Homebrew Formula Not Updated

1. Check if `homebrew-tap` repo exists
2. Verify PAT has `repo` permissions
3. Check GoReleaser logs for errors

### Users Can't Install

```bash
# Users should first add the tap
brew tap ishk9/tap

# Then install
brew install flog
```

---

## Version Bumping Guide

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Bug fix     | Patch        | 0.1.0 → 0.1.1 |
| New feature | Minor        | 0.1.0 → 0.2.0 |
| Breaking change | Major    | 0.1.0 → 1.0.0 |

```bash
# Example: Release v0.2.0
git tag v0.2.0
git push origin v0.2.0
```

