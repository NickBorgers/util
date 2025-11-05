# Smart-Crop-Video Testing Guide

Complete guide for running, debugging, and understanding the test suite.

## Table of Contents

- [Quick Start](#quick-start)
- [Test Organization](#test-organization)
- [Running Tests](#running-tests)
- [Test Fixtures](#test-fixtures)
- [Debugging](#debugging)
- [CI/CD Integration](#cicd-integration)
- [Test Coverage](#test-coverage)

## Quick Start

### Prerequisites

**Option 1: Docker only (fastest for local dev)**
- Docker installed and running
- No other dependencies needed

**Option 2: Full test suite (for comprehensive testing)**
- Python 3.9+
- Docker
- FFmpeg
- Playwright (`playwright install chromium`)

### Run Tests Quickly

```bash
# Quick unit tests (~2 seconds) - Docker only
docker-compose -f docker-compose.test.yml run test-quick

# Fast tests without UI (~8 minutes) - Docker only
docker-compose -f docker-compose.test.yml run test-fast

# All tests including Web UI (~18 minutes) - Docker only
docker-compose -f docker-compose.test.yml run test-all

# Skip Web UI tests for faster iteration
pytest tests/ -m "not ui" -v
```

## Test Organization

### Test Suite Structure (334 total tests)

```
tests/
├── unit/                    # 286 tests, pure functions, <1s
│   ├── test_dimensions.py
│   ├── test_scoring.py
│   ├── test_candidates.py
│   ├── test_scenes.py
│   ├── test_scene_analysis.py
│   ├── test_grid.py
│   ├── test_parallel.py
│   └── test_mock_analyzer.py
│
├── integration/             # 8 tests, real FFmpeg, ~2min
│   └── test_parallel_integration.py
│
├── test_container.py        # 15 tests, Docker lifecycle, ~30s
├── test_api.py              # 19 tests, Flask endpoints, ~3min
├── test_web_ui_focused.py   # 5 tests, Playwright, ~10min
├── test_diagnostic.py       # 1 test, monitoring, ~2min
│
├── helpers/                 # Test utilities
│   ├── docker_manager.py    # Docker fixtures
│   └── api_helper.py        # API client fixtures
│
├── mocks/                   # Mock implementations
│   └── mock_analyzer.py     # MockVideoAnalyzer
│
└── conftest.py              # Pytest configuration
```

### Test Categories (Pytest Markers)

```bash
# Run specific categories
pytest -m "unit"          # Unit tests only
pytest -m "integration"   # Integration tests only
pytest -m "api"           # API tests only
pytest -m "ui"            # Web UI tests only
pytest -m "not ui"        # Skip slow Web UI tests (recommended for dev)
```

## Running Tests

### Using Docker Compose (Recommended)

Docker Compose provides isolated test environments with all dependencies:

```bash
# Build test image
docker-compose -f docker-compose.test.yml build

# Available test services:
docker-compose -f docker-compose.test.yml run test-quick       # Unit tests, quiet (~2s)
docker-compose -f docker-compose.test.yml run test-unit        # Unit tests, verbose
docker-compose -f docker-compose.test.yml run test-integration # Integration tests
docker-compose -f docker-compose.test.yml run test-fast        # Skip UI tests (~8min)
docker-compose -f docker-compose.test.yml run test-all         # All tests (~18min)
docker-compose -f docker-compose.test.yml run test-shell       # Debug shell

# Custom pytest command:
docker-compose -f docker-compose.test.yml run tests pytest tests/test_api.py -v
```

### Using Pytest Directly (On Host)

If you have Python, FFmpeg, and Playwright installed:

```bash
# Install dependencies
pip install -r tests/requirements.txt
playwright install chromium

# Run tests
pytest tests/                    # All tests
pytest tests/unit/ -v            # Unit tests only
pytest tests/ -m "not ui" -v     # Skip Web UI tests
pytest tests/test_api.py -v      # Specific test file
pytest -k "test_candidate" -v    # Tests matching pattern

# With coverage
pytest tests/unit/ --cov=smart_crop --cov-report=term-missing
```

### Performance Tips

**Fast iteration during development:**
```bash
# Fastest: Unit tests only (~0.5s)
pytest tests/unit/ -q

# Fast: Skip Web UI (~8min vs ~18min)
pytest tests/ -m "not ui" -v

# Parallel execution for unit tests
pytest tests/unit/ -n auto
```

## Test Fixtures

### Overview

Fixtures are defined in:
- `conftest.py` - Core fixtures (video files, temp directories, FFmpeg helpers)
- `helpers/docker_manager.py` - Docker-related fixtures
- `helpers/api_helper.py` - API client fixtures

### Key Fixtures

#### File Management

```python
def test_example(temp_workdir: Path):
    """temp_workdir provides isolated directory with test video copy"""
    video = temp_workdir / "example_movie.mov"
    output = temp_workdir / "output.mov"
```

#### Docker Container

```python
def test_container(smart_crop_container):
    """smart_crop_container provides running container with:
    - container: Docker container object
    - port: Dynamic port number
    - base_url: API base URL
    - workdir: Temporary work directory
    """
    base_url = smart_crop_container["base_url"]
    response = requests.get(f"{base_url}/api/status")
```

#### API Client

```python
def test_api(api_client):
    """api_client provides helper methods:
    - get_status()
    - wait_for_status(target_status, timeout=60)
    - select_crop(index)
    - choose_acceleration(enable)
    - select_scenes(selections)
    - get_preview_image(index)
    - get_scene_thumbnail(scene_id, frame_type)
    """
    status = api_client["get_status"]()
    api_client["select_crop"](1)
```

#### FFmpeg Helper

```python
def test_video(ffmpeg_helper):
    """ffmpeg_helper provides video utilities:
    - get_video_info(path) -> dict
    - verify_aspect_ratio(path, ratio) -> bool
    - video_exists_and_valid(path) -> bool
    """
    info = ffmpeg_helper["get_video_info"](video_path)
    assert info["width"] == 1920
```

### Fixture Scopes

- **Session**: Built once per test session (Docker client, Docker image)
- **Function**: Fresh for each test (containers, temp directories)

## Debugging

### Common Issues

#### 1. Port Conflicts

**Symptom**: `Container failed to start` or `Port already in use`

**Solution**: Tests use dynamic port allocation. If you see conflicts:
```bash
# Clean up containers
docker ps -a | grep smart-crop-video | awk '{print $1}' | xargs docker rm -f

# Clean up Docker resources
docker system prune -af
```

#### 2. Flask Server Timeout

**Symptom**: `Flask server didn't start within 30s`

**Solution**: Check container logs:
```bash
# Enable verbose logging
PYTEST_VERBOSE=1 pytest tests/test_container.py -v

# Check Docker logs
docker logs <container-id>
```

#### 3. Web UI Tests Fail

**Symptom**: Playwright errors or timeout

**Solution**:
```bash
# Ensure Playwright is installed
playwright install chromium --with-deps

# Run with headed browser for debugging
PWDEBUG=1 pytest tests/test_web_ui_focused.py -v
```

#### 4. Import Errors After Refactoring

**Symptom**: `ModuleNotFoundError` for helper modules

**Solution**: Ensure Python can find test helpers:
```bash
# Run from smart-crop-video directory
cd /path/to/smart-crop-video
pytest tests/ -v
```

### Debug Techniques

**Interactive debugging:**
```bash
# Drop into PDB on failure
pytest tests/test_api.py -v --pdb

# Stop after first failure
pytest tests/ -x

# Show local variables on failure
pytest tests/ -l
```

**Shell access:**
```bash
# Interactive shell in test container
docker-compose -f docker-compose.test.yml run test-shell

# Inside container, you can:
# - Run pytest manually
# - Inspect files
# - Debug FFmpeg commands
```

**Verbose logging:**
```bash
# Enable test logging
pytest tests/ -v --log-cli-level=DEBUG

# Check test log file
cat tests/test_run.log
```

## CI/CD Integration

### GitHub Actions Workflow

Tests run automatically on:
- **Push to main**: Fast tests only (skip Web UI, ~8min)
- **Pull requests**: All tests including Web UI (~18min)
- **Manual trigger**: All tests

### Test Execution Strategy

**Fast Path (Push to main):**
1. Unit tests (286 tests, parallel, ~10s)
2. Integration tests (8 tests, ~2min)
3. Container tests (15 tests, ~30s)
4. API tests (19 tests, ~3min)
5. Diagnostic tests (1 test, ~2min)
**Total: ~8 minutes, 329 tests**

**Slow Path (Pull Requests):**
All fast tests + Web UI tests (5 tests, ~10min)
**Total: ~18 minutes, 334 tests**

### CI Configuration

The workflow is split for efficiency:
```yaml
# Fast tests always run
- name: Run unit tests (fast)
  run: pytest tests/unit/ -v -n auto

# Web UI tests only on PRs
- name: Run Web UI tests
  if: github.event_name == 'pull_request' || github.event_name == 'workflow_dispatch'
  run: pytest tests/test_web_ui_focused.py -v
```

### Reproducing CI Failures Locally

```bash
# Run the same tests as CI fast path
docker-compose -f docker-compose.test.yml run test-fast

# Run full CI suite
docker-compose -f docker-compose.test.yml run test-all

# Or on host (requires all dependencies)
pytest tests/ -m "not ui" -v  # Fast path
pytest tests/ -v              # Slow path
```

## Test Coverage

### Current Coverage: 73%

**Fully tested modules (100%):**
- ✅ Dimension calculations
- ✅ Grid generation
- ✅ Scoring strategies
- ✅ Candidate selection
- ✅ Scene detection
- ✅ Scene analysis
- ✅ Parallel processing

**Partially tested:**
- ⚠️ Video analysis (90% via integration tests)
- ⚠️ Flask API endpoints (covered by API tests)

**Lower coverage:**
- ❌ UI/state management (~10%)
- ❌ Encoding logic (~8%)
- ❌ Main orchestration (~7%)

### Running Coverage Reports

```bash
# Generate HTML coverage report
pytest tests/unit/ --cov=smart_crop --cov-report=html
open htmlcov/index.html

# Terminal coverage report
pytest tests/unit/ --cov=smart_crop --cov-report=term-missing

# Coverage for specific module
pytest tests/unit/test_scoring.py --cov=smart_crop.scoring -v
```

### Coverage Goals

- **Target**: Maintain 70%+ overall coverage
- **Unit tests**: Aim for 100% on core logic modules
- **Integration tests**: Validate FFmpeg integration
- **UI tests**: Cover critical user workflows

## Best Practices

### Writing New Tests

**1. Use appropriate test level:**
```python
# Unit test: Fast, isolated, no external dependencies
def test_dimension_calculation():
    assert calculate_dimensions(1920, 1080, "9:16") == (607, 1080)

# Integration test: Real dependencies (FFmpeg)
def test_video_analysis_integration():
    analyzer = VideoAnalyzer("example_movie.mov")
    result = analyzer.analyze()

# API test: Full container lifecycle
def test_crop_selection_api(api_client):
    api_client["select_crop"](1)
    status = api_client["wait_for_status"]("crop_selected")
```

**2. Use fixtures for common setup:**
```python
# Good: Reuse fixtures
def test_with_video(temp_workdir):
    video = temp_workdir / "example_movie.mov"
    assert video.exists()

# Bad: Manual setup
def test_with_video():
    tmpdir = tempfile.mkdtemp()
    shutil.copy("example_movie.mov", tmpdir)
    # Forgot to clean up!
```

**3. Add markers:**
```python
@pytest.mark.ui
@pytest.mark.timeout(300)
def test_web_interface():
    # Slow test, will be skipped in fast runs
    pass
```

### Performance Guidelines

- **Unit tests**: Should run in <0.1s each
- **Integration tests**: <30s each
- **Container tests**: <60s each
- **API tests**: <180s each
- **UI tests**: <600s each

### Test Independence

Each test should:
- ✅ Run in any order
- ✅ Clean up after itself (fixtures handle this)
- ✅ Not depend on other tests
- ✅ Use isolated temp directories

## Troubleshooting Reference

| Symptom | Likely Cause | Solution |
|---------|--------------|----------|
| Import errors | Wrong directory | Run from `smart-crop-video/` |
| Port conflicts | Old containers | `docker system prune -af` |
| Flask timeout | Build failure | Check `docker logs <container>` |
| Playwright errors | Browser not installed | `playwright install chromium --with-deps` |
| Slow tests | Running all UI tests | Use `pytest -m "not ui"` |
| Coverage low | Only testing subset | Run `pytest tests/unit/ --cov` |

## Additional Resources

- **Test logs**: `tests/test_run.log` (created automatically)
- **Coverage reports**: `htmlcov/` (after running with `--cov-report=html`)
- **CI logs**: GitHub Actions workflow runs
- **Coverage history**: `TEST_COVERAGE_SUMMARY.md`
- **CI strategy**: `CI_TEST_COVERAGE.md`

## Summary

**For daily development:**
```bash
# Fastest feedback (~2s)
docker-compose -f docker-compose.test.yml run test-quick

# Before committing (~8min)
docker-compose -f docker-compose.test.yml run test-fast
```

**Before creating PR:**
```bash
# Full test suite (~18min)
docker-compose -f docker-compose.test.yml run test-all
```

**When debugging:**
```bash
# Interactive shell
docker-compose -f docker-compose.test.yml run test-shell

# Or use PDB
pytest tests/test_api.py -v --pdb
```
