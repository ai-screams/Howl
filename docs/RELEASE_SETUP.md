# Release Automation Setup

## Overview

Howl uses automated releases via GitHub Actions. The auto-release workflow requires a Personal Access Token (PAT) to trigger the release pipeline.

## Why PAT is Required

GitHub's GITHUB_TOKEN cannot trigger other workflows to prevent infinite loops. Our release flow needs:

1. auto-release creates tag → 2. tag push triggers release workflow

Reference: [GitHub Docs](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow)

## PAT Setup (One-time)

### 1. Create Fine-grained PAT

- Go to: https://github.com/settings/tokens?type=beta
- Click "Generate new token" → "Fine-grained token"
- **Token name:** `howl-auto-release`
- **Expiration:** 90 days
- **Repository access:** Only select repositories → `ai-screams/Howl`
- **Permissions:**
  - Repository permissions → Contents: **Read and write**
  - (All others: No access)
- Click "Generate token"
- **Copy the token** (shown only once)

### 2. Add Secret to Repository

- Go to: https://github.com/ai-screams/Howl/settings/secrets/actions
- Click "New repository secret"
- **Name:** `AUTO_RELEASE_PAT`
- **Value:** [paste token from step 1]
- Click "Add secret"

### 3. Verify Setup

After next merge to main:

- Check: https://github.com/ai-screams/Howl/actions/workflows/auto-release.yaml
- Auto-release should succeed AND trigger release workflow
- Verify tag push triggers release.yaml

## Maintenance

### Token Expiration (Every 90 days)

1. GitHub will email 7 days before expiration
2. Create new PAT (same steps as above)
3. Update `AUTO_RELEASE_PAT` secret with new token
4. Old token auto-expires

### If Token Expires

**Symptoms:**

- auto-release fails with "Authentication failed"
- Release workflow not triggered after tag push

**Quick fix:**

1. Create new PAT (steps above)
2. Update secret
3. Re-run failed auto-release workflow

## Testing

Test PAT without releasing:

```bash
# Create test tag locally
git tag v1.2.1-test
git push origin v1.2.1-test

# Verify release workflow triggered
gh run list --workflow=release.yaml

# Delete test tag
git tag -d v1.2.1-test
git push origin :refs/tags/v1.2.1-test
```

## Security Notes

- PAT has write access to repository
- Store securely, never commit to code
- Rotate every 90 days
- Use fine-grained token (not classic) for minimal permissions
