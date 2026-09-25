# Release Automation Setup

## Overview

Howl uses automated releases via GitHub Actions. The auto-release workflow uses a GitHub App to trigger the release pipeline.

## Why GitHub App is Used

GitHub's GITHUB_TOKEN cannot trigger other workflows to prevent infinite loops. Our release flow needs:

1. auto-release creates tag → 2. tag push triggers release workflow

GitHub Apps provide more secure token generation (1-hour expiry) vs Personal Access Tokens (90-day expiry).

Reference: [GitHub Docs](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication#using-the-github_token-in-a-workflow)

## GitHub App Setup (One-time)

### 1. Create GitHub App (Organization Admin)

- Go to: https://github.com/organizations/ai-screams/settings/apps/new
- **GitHub App name:** `howl-release-automation`
- **Homepage URL:** `https://github.com/ai-screams/howl`
- **Webhook:** Uncheck "Active" (not needed)
- **Repository permissions:**
  - Contents: **Read and write**
  - Metadata: Read only (automatic)
- **Where can this GitHub App be installed?** Only on this account
- Click "Create GitHub App"

### 2. Generate Private Key

After app creation:

- Scroll down to **Private keys** section
- Click "Generate a private key"
- Download `.pem` file (save securely!)
- Copy **App ID** from page top (e.g., `2833108`)

### 3. Install App to Repository

- Left menu → "Install App"
- Click "Install" next to ai-screams
- **Only select repositories** → Check `Howl`
- Click "Install"

### 4. Add Repository Secrets

Go to: https://github.com/ai-screams/howl/settings/secrets/actions

**Secret 1: APP_ID**

- Click "New repository secret"
- Name: `APP_ID`
- Value: `2833108` (your App ID number)
- Click "Add secret"

**Secret 2: APP_PRIVATE_KEY**

- Click "New repository secret" again
- Name: `APP_PRIVATE_KEY`
- Value: (entire .pem file contents)
  ```
  -----BEGIN RSA PRIVATE KEY-----
  MIIEpAIBAAKCAQEA...
  ...
  -----END RSA PRIVATE KEY-----
  ```
- Click "Add secret"

### 5. Verify Setup

Test token generation:

```bash
gh workflow run test-github-app.yaml
gh run list --workflow=test-github-app.yaml
```

After next merge to main, verify auto-release triggers release workflow

## Maintenance

### Private Key Rotation (If Compromised)

GitHub App tokens auto-expire after 1 hour, but private key is persistent.

**If private key is compromised:**

1. Go to: https://github.com/organizations/ai-screams/settings/apps/howl-release-automation
2. Scroll to **Private keys** section
3. Click "Revoke" on old key
4. Click "Generate a private key" for new key
5. Download new `.pem` file
6. Update `APP_PRIVATE_KEY` secret with new key contents
7. Test with: `gh workflow run test-github-app.yaml`

### If Authentication Fails

**Symptoms:**

- auto-release fails with "Bad credentials" or "Authentication failed"
- Release workflow not triggered after tag push

**Check:**

1. Verify `APP_ID` secret exists and matches GitHub App ID
2. Verify `APP_PRIVATE_KEY` secret contains full .pem file
3. Verify GitHub App is installed on Howl repository
4. Check App permissions: Contents = Read and write

**Quick fix:**

1. Regenerate private key (steps above)
2. Update `APP_PRIVATE_KEY` secret
3. Re-run failed auto-release workflow

## Testing

Test GitHub App authentication:

```bash
# Test token generation (no release)
gh workflow run test-github-app.yaml
gh run list --workflow=test-github-app.yaml

# Test full flow with test tag
git tag v1.2.1-test
git push origin v1.2.1-test

# Verify release workflow triggered
gh run list --workflow=release.yaml

# Delete test tag
git tag -d v1.2.1-test
git push origin :refs/tags/v1.2.1-test
```

## Security Notes

- GitHub App has write access to repository Contents
- Private key stored in GitHub Secrets (encrypted at rest)
- Tokens auto-expire after 1 hour (vs 90 days for PAT)
- App scoped to single repository only
- Never commit `.pem` file or expose `APP_PRIVATE_KEY` secret
