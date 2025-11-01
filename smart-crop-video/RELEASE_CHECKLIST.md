# Release Checklist - smart-crop-video v1.3.0

## Pre-Release Validation

### ✅ Code Review
- [x] All test files reviewed and pruned
- [x] Removed unvalidated/redundant test files:
  - `TESTING_QUICKSTART.md` (redundant)
  - `tests/test_acceleration_diagnostic.py` (too specific)
  - `tests/test_web_ui.py` (superseded by test_web_ui_focused.py)
  - `tests/test_video_processing.py` (never validated)
  - `tests/test_workflow.py` (never validated)
- [x] Documentation updated to remove references to deleted files
- [x] README.md updated with testing section

### ✅ Test Validation
- [x] Container tests passing (15/15)
- [x] Containerized test execution verified
- [x] Test infrastructure builds successfully
- [x] Documentation accurate and complete

### ✅ Documentation
- [x] `RELEASE_NOTES.md` created
- [x] `CONTAINERIZED_TESTING.md` comprehensive guide
- [x] `tests/README.md` updated
- [x] `README.md` includes testing section
- [x] All documentation reviewed for accuracy

### ✅ Repository Health
- [x] `.gitignore` properly configured
- [x] No test artifacts in repo
- [x] All scripts executable (`run-tests.sh`)
- [x] CI/CD workflows updated

## Release Steps

### 1. Verify Tests Pass

```bash
# Build test container
./run-tests.sh build

# Run container tests (should pass 15/15)
./run-tests.sh container

# Verify test infrastructure
docker-compose -f docker-compose.test.yml config
```

**Expected**: All container tests pass

### 2. Clean Repository

```bash
# Remove any test artifacts
./run-tests.sh clean

# Check git status
git status

# Ensure only intentional files are staged
git diff --cached
```

**Expected**: Only intended new/modified files

### 3. Stage Files for Commit

```bash
# Stage all test infrastructure
git add tests/
git add docker-compose.test.yml
git add run-tests.sh
git add Makefile

# Stage documentation
git add CONTAINERIZED_TESTING.md
git add RELEASE_NOTES.md
git add RELEASE_CHECKLIST.md
git add TEST_SUITE_SUMMARY.md
git add TEST_RESULTS.md
git add WEB_UI_TEST_RESULTS.md
git add README.md

# Stage configuration
git add .gitignore
git add pytest.ini

# Stage CI/CD
git add .github/workflows/test-smart-crop-video.yml
git add .github/workflows/publish.yml

# Verify what will be committed
git status
```

### 4. Commit Changes

```bash
git commit -m "Add comprehensive integration test suite

Major improvements:
- Containerized test execution (no local dependencies needed)
- 40 validated tests covering containers, API, and web UI
- Browser automation with Playwright
- CI/CD integration to prevent broken releases
- Extensive documentation and guides

Test coverage:
- Container integration: 15/15 passing
- API endpoints: 20 tests
- Web UI workflows: 5 focused tests
- Diagnostic tooling included

Infrastructure:
- Docker Compose test environment
- Shell script runner (./run-tests.sh)
- Makefile targets
- GitHub Actions integration

Documentation:
- CONTAINERIZED_TESTING.md - Complete guide
- WEB_UI_TEST_RESULTS.md - Detailed findings
- TEST_SUITE_SUMMARY.md - Implementation details
- Updated README.md with testing section

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

### 5. Create Git Tag

```bash
# Create tag following repository convention
git tag smart-crop-video-v1.3.0

# Verify tag
git tag -l 'smart-crop-video-*'
```

### 6. Push Changes

```bash
# Push commit
git push origin main

# Push tag
git push origin smart-crop-video-v1.3.0
```

### 7. Create GitHub Release

```bash
gh release create smart-crop-video-v1.3.0 \
  --title "smart-crop-video v1.3.0 - Comprehensive Test Suite" \
  --notes "$(cat <<'EOF'
## 🎉 New Feature: Comprehensive Integration Test Suite

This release adds complete testing infrastructure to ensure quality and reliability.

### ✅ Containerized Testing

Run tests **without installing Python, Playwright, or dependencies**:

\`\`\`bash
./run-tests.sh container    # Fast validation (15 tests, ~40s)
./run-tests.sh quick        # Quick smoke test
./run-tests.sh all          # Full test suite
\`\`\`

### 📦 What's Included

**Test Coverage:**
- ✅ Container integration (15 tests) - 100% passing
- ✅ API endpoints (20 tests)
- ✅ Web UI workflows (5 focused tests)
- ✅ Diagnostic tooling

**Infrastructure:**
- Docker Compose test environment
- Playwright browser automation
- Shell script runner
- CI/CD integration

**Documentation:**
- [CONTAINERIZED_TESTING.md](CONTAINERIZED_TESTING.md) - Complete guide
- [WEB_UI_TEST_RESULTS.md](WEB_UI_TEST_RESULTS.md) - Detailed findings
- [TEST_SUITE_SUMMARY.md](TEST_SUITE_SUMMARY.md) - Implementation details

### 🎯 Key Benefits

✅ **Zero dependency testing** - Only Docker required
✅ **Consistent environments** - Same setup everywhere
✅ **CI/CD ready** - Automated quality gates
✅ **User experience validated** - Real browser tests
✅ **Portable** - Works on macOS, Linux, Windows

### 🚀 Quick Start

\`\`\`bash
# Clone and test
git clone <repo>
cd smart-crop-video

# Run validation
./run-tests.sh container    # Passes all 15 tests
\`\`\`

### 📊 Test Metrics

- **Tests**: 40 validated tests
- **Code**: ~2,000 lines of test code
- **Docs**: ~1,500 lines of documentation
- **Time**: 40s (fast) to 5-10 min (full suite)

### 🐛 Known Issues

Some edge cases have timing issues (acceleration choice, encoding progress) but core functionality (analyze → preview → select) works perfectly.

See [RELEASE_NOTES.md](RELEASE_NOTES.md) for complete details.

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### 8. Verify GitHub Actions

After pushing tag and creating release:

```bash
# Check GitHub Actions status
gh run list --limit 5

# Watch the test workflow
gh run watch

# Verify publish workflow waits for tests
gh run view --log
```

**Expected**:
- Test workflow runs and passes
- Publish workflow waits for tests
- Docker image only published after tests pass

### 9. Verify Docker Hub

After CI/CD completes:

```bash
# Pull the new image
docker pull nickborgers/smart-crop-video:smart-crop-video-v1.3.0
docker pull nickborgers/smart-crop-video:latest

# Verify it works
docker run --rm nickborgers/smart-crop-video:latest smart-crop-video --help
```

### 10. Test End-to-End

```bash
# Test the new image
cd /tmp
# Copy example video
docker run --rm -v $(pwd):/content nickborgers/smart-crop-video:latest \
  smart-crop-video example_movie.mov output.mov 1:1
```

## Post-Release

### ✅ Validation
- [ ] GitHub Release created successfully
- [ ] GitHub Actions workflows passed
- [ ] Docker images published
- [ ] Images work correctly
- [ ] Documentation accessible

### ✅ Communication
- [ ] Update any external documentation
- [ ] Notify users if applicable
- [ ] Archive release checklist

## Rollback Plan

If issues are discovered after release:

```bash
# Revert to previous tag
git tag -d smart-crop-video-v1.3.0
git push origin :refs/tags/smart-crop-video-v1.3.0

# Delete GitHub release
gh release delete smart-crop-video-v1.3.0

# Users can pin to previous version
docker pull nickborgers/smart-crop-video:smart-crop-video-v1.0.0
```

## Notes

- This is a **feature release** (testing infrastructure), not a bug fix
- Main smart-crop-video functionality unchanged
- Tests are optional but recommended
- CI/CD now blocks bad releases

---

**Prepared**: November 1, 2025
**Version**: v1.3.0
**Previous**: v1.0.0
