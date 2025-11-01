# Smart-Crop-Video Integration Test Suite

Comprehensive integration tests for the smart-crop-video utility, covering containerized testing, API endpoints, Web UI interaction, end-to-end workflows, and video processing verification.

## ⚡ Quick Start (Containerized - Recommended)

**Run tests without installing Python, Playwright, or any dependencies!**

Only Docker and Docker Compose required:

```bash
# Fast validation (15 tests, ~40s)
./run-tests.sh container

# Quick smoke test
./run-tests.sh quick

# Full test suite
./run-tests.sh all
```

📚 **See [CONTAINERIZED_TESTING.md](../CONTAINERIZED_TESTING.md) for complete documentation**

---

## Overview

This test suite follows the integration testing philosophy of the util repository:
- **Containerized testing**: Uses Docker for isolated, reproducible test environments
- **Realistic scenarios**: Tests exercise actual interfaces with real video files
- **Production-like environments**: Tests run in the same Docker container as production
- **Comprehensive coverage**: Tests cover all critical paths and edge cases

## Test Categories

### 1. Container Integration Tests (`test_container.py`)
Tests Docker image building, container startup, port mapping, and volume mounts.

**Key tests:**
- Docker image builds successfully with all dependencies (FFmpeg, Python, Flask)
- Container starts and runs with correct port mapping (8765)
- Volume mounts allow file access between host and container
- Environment variables are properly passed and respected
- Network access is available (unlike older utils with `--network=none`)

### 2. API Endpoint Tests (`test_api.py`)
Tests all Flask API endpoints for proper responses and state management.

**Endpoints tested:**
- `GET /` - Main UI HTML
- `GET /api/status` - Status JSON
- `GET /api/preview/<int:index>` - Preview images
- `POST /api/select/<int:index>` - Crop selection
- `POST /api/acceleration/<choice>` - Acceleration choice
- `GET /api/scene_thumbnail/<scene_id>/<frame_type>` - Scene thumbnails
- `POST /api/scene_selections` - Scene speed selections

**Key tests:**
- Status endpoint returns all required fields
- Status transitions through correct states (analyzing → candidates_ready → etc.)
- Candidates have proper structure and validation
- Preview endpoints serve valid JPEG images
- State updates correctly when selections are made
- Concurrent requests are handled properly

### 3. Web UI Tests (`test_web_ui.py`)
Tests browser-based user interface using Python Playwright library.

**Note:** These tests use the Python Playwright library (not the MCP Playwright server), as the MCP server runs in a separate container and cannot access localhost:8765.

**Key tests:**
- UI loads with correct title and structure
- Status section updates dynamically via polling
- Candidate cards appear with images and information
- Clicking candidates selects them visually
- Confirm button enables after selection
- Acceleration section checkbox and controls work
- Scene selection section appears with thumbnails (if enabled)
- Scene checkboxes toggle and speed dropdowns appear
- "Select All" and "Clear All" buttons work
- Encoding progress bar updates
- Completion message appears when done

### 4. End-to-End Workflow Tests (`test_workflow.py`)
Tests complete workflows from video input to encoded output.

**Workflows tested:**
- Complete workflow without acceleration
- Workflow with acceleration enabled but no scenes selected
- Workflow with acceleration and scene selections
- Different aspect ratios (9:16, 4:5, 16:9, 1:1)
- Preview file creation and cleanup
- Scene thumbnail creation and cleanup
- Environment variable effects (PRESET, CROP_SCALE, ANALYSIS_FRAMES)
- Progress updates throughout workflow
- Strategy selection tracking

### 5. Video Processing Tests (`test_video_processing.py`)
Tests video output quality, dimensions, encoding parameters, and file integrity.

**Key tests:**
- Output video has correct aspect ratio dimensions
- Output uses H.264 codec as specified
- Audio stream is preserved
- Duration is reasonable compared to input
- Output is playable and decodable
- File size is reasonable for quality settings
- Crop position is correctly applied
- Encoding preset affects encoding speed
- Preview images are created with correct dimensions
- Scene thumbnails are created (when acceleration enabled)
- Variable speed encoding produces valid output
- CRF quality parameter results in decent bitrate
- Frame rate is preserved from input to output

## Prerequisites

### Required Software
- Python 3.9+
- Docker (with Docker daemon running)
- FFmpeg and FFprobe (installed locally for verification tests)

### Python Dependencies
Install test dependencies:
```bash
cd /Users/nborgers/code/util/smart-crop-video
pip install -r tests/requirements.txt
```

### Playwright Browser Installation
After installing Playwright, install the browser:
```bash
playwright install chromium
```

### Test Video
The test suite requires `example_movie.mov` in the `smart-crop-video/` directory. This file should already exist in the repository.

## Running Tests

### Run All Tests
```bash
cd /Users/nborgers/code/util/smart-crop-video
pytest tests/
```

### Run Specific Test Categories
```bash
# Container tests only
pytest tests/test_container.py

# API tests only
pytest tests/test_api.py

# Web UI tests only
pytest tests/test_web_ui.py

# Workflow tests only
pytest tests/test_workflow.py

# Video processing tests only
pytest tests/test_video_processing.py
```

### Run Tests with Markers
```bash
# Run only fast tests (exclude slow tests)
pytest tests/ -m "not slow"

# Run only API tests
pytest tests/ -m api

# Run only integration tests
pytest tests/ -m integration
```

### Run Tests in Parallel
```bash
# Run tests across 4 workers (faster execution)
pytest tests/ -n 4
```

### Run with Verbose Output
```bash
pytest tests/ -v
```

### Run with Container Logs
```bash
# Enable verbose container logging
PYTEST_VERBOSE=1 pytest tests/ -v
```

## Test Execution Times

- **Container tests**: ~2-3 minutes
- **API tests**: ~3-5 minutes (includes analysis phase)
- **Web UI tests**: ~5-10 minutes (includes full workflows)
- **Workflow tests**: ~10-20 minutes (multiple complete workflows)
- **Video processing tests**: ~5-10 minutes

**Total suite**: ~30-60 minutes depending on hardware and parallel execution.

## Test Architecture

### Fixtures (`conftest.py`)

The test suite uses pytest fixtures for setup and teardown:

- **`docker_client`**: Provides a Docker client for the test session
- **`docker_image`**: Builds the Docker image once per session
- **`test_video_path`**: Path to the example test video
- **`temp_workdir`**: Temporary directory with a copy of test video
- **`smart_crop_container`**: Runs container and yields connection info
- **`api_client`**: Provides helper methods for API testing
- **`ffmpeg_helper`**: Helper functions for video verification

### Temporary Directories

Each test gets a fresh temporary directory with a copy of the test video. This ensures:
- Tests don't interfere with each other
- Output files are isolated
- Cleanup happens automatically

### Container Management

Containers are automatically started and stopped for each test that needs them. Cleanup includes:
- Stopping the container gracefully
- Removing the container
- Capturing logs for debugging (if `PYTEST_VERBOSE=1`)

## Debugging Tests

### View Container Logs
Set the `PYTEST_VERBOSE` environment variable:
```bash
PYTEST_VERBOSE=1 pytest tests/test_api.py -v
```

### Keep Containers Running
Modify the fixture in `conftest.py` temporarily:
```python
# In smart_crop_container fixture
finally:
    if container:
        import pdb; pdb.set_trace()  # Debug breakpoint
        # ... cleanup code
```

### Run Single Test
```bash
pytest tests/test_api.py::test_status_endpoint_returns_json -v
```

### Increase Timeouts
Edit `pytest.ini` to increase the global timeout:
```ini
timeout = 1800  # 30 minutes
```

Or use the command line:
```bash
pytest tests/ --timeout=1800
```

## Common Issues

### Issue: "Flask server didn't start within timeout"
**Cause:** Container startup is slow or Docker daemon is busy.
**Solution:** Increase timeout in `conftest.py` or ensure Docker has adequate resources.

### Issue: "Playwright not installed"
**Cause:** Playwright Python library not installed or browser not installed.
**Solution:**
```bash
pip install playwright
playwright install chromium
```

### Issue: "Test video not found"
**Cause:** `example_movie.mov` is missing.
**Solution:** Ensure the test video exists in `/Users/nborgers/code/util/smart-crop-video/example_movie.mov`.

### Issue: Tests timeout during encoding
**Cause:** Video encoding can take several minutes, especially with slower CPUs.
**Solution:** Tests use `ultrafast` preset and reduced analysis frames. If still timing out, increase the timeout in `pytest.ini`.

### Issue: Port 8765 already in use
**Cause:** Another instance of the container or test is running.
**Solution:**
```bash
# Kill any running containers on that port
docker ps -a | grep 8765
docker stop <container_id>
```

### Issue: Permission denied for Docker socket
**Cause:** User doesn't have permission to access Docker.
**Solution:**
```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
# Then log out and back in

# On macOS, ensure Docker Desktop is running
```

## CI/CD Integration

The test suite is designed for GitHub Actions CI/CD:

### GitHub Actions Workflow
See `.github/workflows/test-smart-crop-video.yml` for the CI configuration.

**Key features:**
- Runs on multiple platforms (Linux, macOS)
- Installs all dependencies automatically
- Runs tests in parallel where possible
- Uploads test artifacts (logs, videos)
- Reports test results as GitHub checks

### Running in CI
Tests automatically run on:
- Pull requests to main branch
- Pushes to main branch
- Manual workflow dispatch

## Test Coverage

The test suite aims for high coverage of critical functionality:

- ✅ Docker image building and container lifecycle
- ✅ All API endpoints and state transitions
- ✅ Web UI interaction and visual feedback
- ✅ Complete user workflows (with and without acceleration)
- ✅ Video processing and output verification
- ✅ File artifact creation and cleanup
- ✅ Environment variable configuration
- ✅ Error handling and edge cases

## Best Practices

### Writing New Tests

1. **Use appropriate fixtures**: Leverage existing fixtures in `conftest.py`
2. **Test isolation**: Each test should be independent
3. **Realistic scenarios**: Use the actual test video, not mocks
4. **Proper assertions**: Verify both success and quality
5. **Cleanup**: Ensure resources are properly released
6. **Timeouts**: Set reasonable timeouts for long operations
7. **Skip appropriately**: Use `pytest.skip()` for conditional tests

### Example Test Template
```python
def test_new_feature(api_client: dict, smart_crop_container: dict):
    """Test description."""
    wait_for_status = api_client["wait_for_status"]
    workdir = smart_crop_container["workdir"]

    # Wait for appropriate state
    status = wait_for_status("candidates_ready", timeout=120)

    # Perform actions
    # ...

    # Verify results
    assert expected_condition, "Failure message"
```

## Performance Optimization

### Parallel Execution
Use pytest-xdist for parallel test execution:
```bash
pytest tests/ -n 4
```

### Test Ordering
Fast tests run first by default. Slow tests are marked with `@pytest.mark.slow`.

### Resource Limits
Tests use `ultrafast` preset and reduced analysis frames to speed up encoding.

## Contributing

When adding new features to smart-crop-video:

1. Add corresponding integration tests
2. Follow existing test patterns in this suite
3. Update this README if adding new test categories
4. Ensure tests pass in CI before merging

## Troubleshooting

For additional help:
1. Check container logs: `docker logs <container_id>`
2. Review test logs: `tests/test_run.log`
3. Run tests with `-vv` for extra verbosity
4. Check the smart-crop-video README for application-specific issues

## License

These tests are part of the util repository and follow the same license.
