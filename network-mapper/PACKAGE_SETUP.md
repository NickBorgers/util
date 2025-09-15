# Package Manager Setup Guide

This document explains how to set up and publish Network Mapper to Homebrew and Chocolatey package managers.

## Prerequisites

### For Homebrew

1. **Create a Homebrew Tap Repository**
   - Create a new GitHub repository named `homebrew-tap`
   - The repository MUST be named with the `homebrew-` prefix
   - Repository URL will be: `https://github.com/NickBorgers/homebrew-tap`

2. **GitHub Personal Access Token**
   - Create a token with `repo` scope for pushing to the tap repository
   - Add as secret: `HOMEBREW_TAP_TOKEN` in the main repository settings

### For Chocolatey

1. **Chocolatey Account**
   - Register at https://community.chocolatey.org/account/Register
   - Verify your email and obtain your API key

2. **API Key Setup**
   - Get your API key from https://community.chocolatey.org/account
   - Add as secret: `CHOCOLATEY_API_KEY` in the main repository settings

## Repository Structure

```
network-mapper/
├── homebrew/
│   └── network-mapper.rb        # Homebrew formula template
├── chocolatey/
│   ├── network-mapper.nuspec    # Chocolatey package specification
│   └── tools/
│       ├── chocolateyInstall.ps1    # Installation script
│       └── chocolateyUninstall.ps1  # Uninstallation script
└── PACKAGE_SETUP.md             # This file
```

## How It Works

### Automated Publishing Workflow

The `.github/workflows/package-managers.yml` workflow automatically:

1. **On Release Creation**:
   - Triggers when a new release is published
   - Downloads release artifacts and checksums
   - Updates package configurations with version and checksums
   - Publishes to both package managers

2. **Homebrew Process**:
   - Updates the formula with the new version and SHA256 checksums
   - Pushes the updated formula to your `homebrew-tap` repository
   - Users can then install via: `brew tap nickborgers/tap && brew install network-mapper`

3. **Chocolatey Process**:
   - Updates the nuspec and install script with version and checksum
   - Packs the Chocolatey package
   - Pushes to the Chocolatey Community Repository
   - Users can then install via: `choco install network-mapper`

## Manual Testing

### Test Homebrew Formula Locally

```bash
# Test the formula locally before publishing
cd network-mapper/homebrew
brew install --build-from-source ./network-mapper.rb
brew test network-mapper
brew uninstall network-mapper
```

### Test Chocolatey Package Locally

```powershell
# Pack the package
cd network-mapper\chocolatey
choco pack

# Test install locally
choco install network-mapper -source . -y
choco uninstall network-mapper -y
```

## Initial Setup Steps

1. **Create Homebrew Tap Repository**
   ```bash
   # Create a new repository on GitHub named "homebrew-tap"
   # Clone it locally
   git clone https://github.com/NickBorgers/homebrew-tap.git
   cd homebrew-tap
   mkdir Formula
   echo "# Homebrew Tap for Nick Borgers' Tools" > README.md
   git add .
   git commit -m "Initial tap setup"
   git push
   ```

2. **Configure Repository Secrets**
   - Go to Settings → Secrets and variables → Actions
   - Add `HOMEBREW_TAP_TOKEN` with your GitHub PAT
   - Add `CHOCOLATEY_API_KEY` with your Chocolatey API key

3. **First Release**
   - Create a new release with tag `v1.0.0` (or your version)
   - The workflow will automatically run and publish to both package managers

## Updating Package Versions

The process is fully automated:

1. Create and publish a new GitHub release
2. The workflow automatically:
   - Updates Homebrew formula with new version/checksums
   - Creates and publishes Chocolatey package
   - Updates README with installation instructions

## Manual Workflow Trigger

You can also manually trigger the package publishing:

1. Go to Actions → Package Manager Publishing
2. Click "Run workflow"
3. Enter the version tag (e.g., `v1.0.0`)
4. Click "Run workflow"

## Troubleshooting

### Homebrew Issues

- **Formula not found**: Ensure the tap repository exists and is public
- **Checksum mismatch**: The workflow automatically calculates checksums from release artifacts
- **Installation fails**: Check that all platform binaries are included in the release

### Chocolatey Issues

- **API key invalid**: Verify the API key in repository secrets
- **Package validation fails**: Ensure the nuspec file has valid XML and required fields
- **Moderation queue**: New packages go through moderation; check https://community.chocolatey.org/packages

## Verification Commands

After publishing, verify the packages:

```bash
# Homebrew
brew tap nickborgers/tap
brew info network-mapper
brew install network-mapper

# Chocolatey (Windows)
choco search network-mapper
choco info network-mapper
choco install network-mapper
```

## Package Management Best Practices

1. **Version Tags**: Always use semantic versioning (e.g., v1.0.0)
2. **Release Notes**: Include detailed release notes for users
3. **Testing**: Test packages locally before releasing
4. **Checksums**: Always verify checksums match the release artifacts
5. **Documentation**: Keep installation instructions in README up to date

## Support

For issues with:
- **Homebrew formula**: Open an issue in the `homebrew-tap` repository
- **Chocolatey package**: Open an issue in the main repository
- **General packaging**: Open an issue in the main repository with the `packaging` label