# GitHub Actions CI Test Coverage

## Summary

The GitHub Actions workflow has been updated to run **all 334 tests** instead of just 39 tests, increasing CI coverage from **12% to 100%**.

## Before (Incomplete Coverage)

The workflow `.github/workflows/test-smart-crop-video.yml` was only running:

| Test Suite | Tests | Coverage |
|------------|-------|----------|
| Container tests | 15 | ✅ |
| API tests | 19 | ✅ |
| Web UI tests | 5 | ✅ |
| **TOTAL** | **39** | **12%** |

**Missing from CI:**
- ❌ Unit tests (286 tests) - Core business logic
- ❌ Integration tests (8 tests) - FFmpeg integration
- ❌ Diagnostic tests (1 test) - Application monitoring

## After (Complete Coverage)

The updated workflow now runs all tests:

| Test Suite | Tests | Speed | Coverage |
|------------|-------|-------|----------|
| Unit tests | 286 | ~10s | ✅ |
| Integration tests | 8 | ~2min | ✅ |
| Container tests | 15 | ~30s | ✅ |
| API tests | 19 | ~3min | ✅ |
| Web UI tests | 5 | ~10min | ✅ |
| Diagnostic tests | 1 | ~2min | ✅ |
| **TOTAL** | **334** | **~18min** | **100%** |

## What Changed

### Added Test Steps

```yaml
- name: Run unit tests (fast)
  run: pytest tests/unit/ -v --timeout=60 -n auto

- name: Run integration tests
  run: pytest tests/integration/ -v --timeout=300

- name: Run diagnostic tests
  run: pytest tests/test_diagnostic.py -v --timeout=300
```

### Test Execution Strategy

**CI/CD (GitHub Actions):**
- Runs on Ubuntu with all dependencies installed
- FFmpeg, Docker, and Playwright available
- All 334 tests execute with full coverage
- Tests run in parallel where possible (`-n auto`)

**Local Development:**
- Use `./run-tests-docker.sh all` for 294 tests in ~2 seconds
- Use `pytest tests/ -v` for all 334 tests (requires local setup)

## Benefits of Complete CI Coverage

1. **Catches regressions early** - Unit tests validate all business logic
2. **Validates FFmpeg integration** - Integration tests ensure video processing works
3. **Prevents broken releases** - All critical paths tested before merge
4. **Fast feedback** - Unit tests run in parallel (~10 seconds)
5. **Comprehensive validation** - 100% of test suite runs on every PR

## Continuous Integration Flow

```
PR/Push to main
  ↓
Setup (install dependencies, build Docker image)
  ↓
Run unit tests (286 tests, parallel, ~10s)
  ↓
Run integration tests (8 tests, FFmpeg, ~2min)
  ↓
Run container tests (15 tests, ~30s)
  ↓
Run API tests (19 tests, ~3min)
  ↓
Run Web UI tests (5 tests, Playwright, ~10min)
  ↓
Run diagnostic tests (1 test, ~2min)
  ↓
Upload artifacts & logs
  ↓
✅ All 334 tests passed - Ready to merge/release
```

## Workflow Trigger Conditions

The test workflow runs on:
- **Push to main** (validates commits on main branch)
- **Pull requests to main** (validates before merge)
- **Manual dispatch** (can trigger manually from GitHub UI)
- **Only when smart-crop-video files change** (path filtering for efficiency)

## Recommendations for Contributors

**Before submitting a PR:**
1. Run fast local tests: `./run-tests-docker.sh all` (294 tests in 2s)
2. Verify core functionality works
3. Push to PR - CI will run all 334 tests automatically
4. Fix any failures shown in GitHub Actions

**Before releasing:**
1. Ensure CI passes (all 334 tests)
2. Review test artifacts/logs if needed
3. Tag release following repository conventions

## Monitoring CI Health

- Check GitHub Actions tab for test results
- Review uploaded test logs if tests fail
- Test artifacts saved for 3-7 days for debugging
- Test summary displayed at end of workflow run
