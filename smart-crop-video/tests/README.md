# smart-crop-video Test Suite

Comprehensive testing for smart-crop-video, covering unit tests, integration tests, and end-to-end validation.

## Quick Start

```bash
# Run fast tests (default, ~5 minutes)
./run-tests.sh

# Run comprehensive tests (~15 minutes)
./run-tests.sh comprehensive

# Run everything
./run-tests.sh all-with-e2e
```

## Test Organization

### Directory Structure

```
tests/
├── unit/                      # Fast unit tests (286 tests)
│   ├── test_dimensions.py     # Crop dimension calculations
│   ├── test_scoring.py        # Scoring strategies
│   ├── test_scenes.py         # Scene detection and filtering
│   ├── test_grid.py           # Grid generation
│   └── ...
├── integration/               # Integration tests with FFmpeg
│   ├── test_parallel_integration.py    # Parallel analysis (8 tests)
│   ├── test_end_to_end_video.py       # E2E crop validation (10 tests) 🆕
│   └── test_acceleration.py           # Acceleration features (9 tests) 🆕
├── helpers/                   # Test utilities
│   ├── video_generator.py     # Synthetic video generation (deprecated - used only for fixture regeneration)
│   ├── frame_analyzer.py      # Frame extraction and analysis 🆕
│   ├── docker_manager.py      # Docker container management
│   └── api_helper.py          # API testing utilities
├── fixtures/                  # Pre-generated test videos 🆕
│   ├── motion_top_right.mov   # Motion in top-right corner
│   ├── motion_center.mov      # Motion in center
│   ├── subject_left.mov       # Subject on left side
│   ├── multi_scene.mov        # Multi-scene with varying motion
│   └── audio_test.mov         # Video with audio track
├── test_container.py          # Docker container tests (15 tests)
├── test_api.py                # Flask API tests (19 tests)
├── test_web_ui_focused.py     # Web UI tests (5 tests)
└── test_diagnostic.py         # Diagnostic tests (1 test)
```

## Test Categories

### Fast Tests (< 5 minutes)

Run by default on every commit. Includes:

- **Unit Tests (286 tests)**: Pure function testing, no I/O
  - Dimension calculations with H.264 constraints
  - Scoring strategy normalization
  - Scene detection and filtering
  - Grid generation

- **Integration Tests (8 tests)**: FFmpeg integration
  - Parallel vs sequential analysis
  - Progress tracking
  - Performance validation

- **Container Tests (15 tests)**: Docker environment
  - Image builds correctly
  - Port mapping works
  - Volume mounts accessible
  - Environment variables passed

- **API Tests (19 tests)**: Flask backend
  - Endpoint responses
  - JSON validation
  - State management
  - Preview image generation

**Run fast tests:**
```bash
./run-tests.sh fast
# or
pytest tests/ -m "not comprehensive" -v
```

### Comprehensive Tests (5-10 minutes) 🆕

Run on pull requests, releases, and weekly. Validates real-world user scenarios using **pre-generated test fixtures** (no FFmpeg required for test execution).

#### End-to-End Video Validation (10 tests)

Tests the most critical user concern: **"Did it crop my video correctly?"**

- **Crop Accuracy Tests**
  - Motion Priority strategy captures high-motion regions
  - Subject Detection strategy finds prominent objects
  - Center-motion videos crop to center
  - Different strategies produce different results

- **Aspect Ratio Precision**
  - 1:1 (square) produces exactly equal width/height
  - 9:16 (vertical) produces correct tall aspect ratio
  - Dimensions are always even (H.264 requirement)

- **Output Quality**
  - Output video is playable (decodable)
  - Can extract frames throughout video
  - Audio stream is preserved
  - Audio duration matches video duration

**Implementation:**
- Uses pre-generated synthetic test videos with known characteristics (committed to `tests/fixtures/`)
- Processes videos through smart-crop-video in Docker container
- Extracts frames and analyzes crop position using template matching
- Validates metadata and visual content
- No FFmpeg required for test execution (only for regenerating fixtures)

**Run end-to-end tests:**
```bash
./run-tests.sh e2e
# or
pytest tests/integration/test_end_to_end_video.py -m comprehensive -v
```

#### Acceleration Feature Tests (9 tests) 🆕

Tests intelligent acceleration features:

- **Basic Functionality**
  - Acceleration runs without errors
  - Output is shorter than input
  - Works with auto-detection

- **Duration Calculations**
  - Total duration matches expected for scene speeds
  - Mixed rates (1x, 2x, 4x) calculated correctly

- **Audio Synchronization**
  - Audio tempo matches video speed
  - No desynchronization

- **Scene Detection**
  - Boring sections (low motion) identified
  - High-motion content not accelerated
  - Scene boundaries have no glitches

- **Edge Cases**
  - Very short videos handled gracefully
  - All high-motion videos don't over-accelerate
  - No acceleration mode works correctly

**Run acceleration tests:**
```bash
./run-tests.sh acceleration
# or
pytest tests/integration/test_acceleration.py -m comprehensive -v
```

### Slow Tests (Web UI)

Run on pull requests and manual triggers only:

- **Web UI Tests (5 tests)**: Browser automation with Playwright
  - Progress indicators update during analysis
  - Preview images load and display
  - Selection and confirmation flow works
  - Complete workflow completes successfully

**Run UI tests:**
```bash
./run-tests.sh ui
# or
pytest tests/test_web_ui_focused.py -v
```

## Test Markers

Tests are categorized using pytest markers:

| Marker | Description | When to Run |
|--------|-------------|-------------|
| `fast` | Unit tests (< 1s each) | Every commit |
| `slow` | Tests taking several seconds | Every commit |
| `comprehensive` | End-to-end tests (minutes) | PRs, releases, weekly |
| `container` | Requires Docker container | Every commit |
| `api` | Tests API endpoints | Every commit |
| `ui` | Tests Web UI with browser | PRs, manual |
| `integration` | Tests with FFmpeg | Every commit |
| `video` | Video processing validation | Comprehensive runs |

**Run specific categories:**
```bash
# Only fast tests
pytest -m fast

# Everything except comprehensive
pytest -m "not comprehensive"

# Only comprehensive tests
pytest -m comprehensive

# Container and API tests
pytest -m "container or api"
```

## Running Tests

### Using run-tests.sh (Recommended)

```bash
# Default: fast tests only
./run-tests.sh

# Comprehensive tests
./run-tests.sh comprehensive

# All tests including comprehensive
./run-tests.sh all-with-e2e

# Specific test categories
./run-tests.sh e2e
./run-tests.sh acceleration
./run-tests.sh container
./run-tests.sh api
./run-tests.sh ui

# Quick smoke test
./run-tests.sh quick

# Get help
./run-tests.sh help
```

### Using pytest Directly

```bash
# Install dependencies
pip install -r tests/requirements.txt

# Run all fast tests
pytest tests/ -m "not comprehensive" -v

# Run comprehensive tests
pytest tests/integration/test_end_to_end_video.py \
       tests/integration/test_acceleration.py \
       -m comprehensive -v

# Run specific test file
pytest tests/unit/test_dimensions.py -v

# Run specific test
pytest tests/integration/test_end_to_end_video.py::TestEndToEndVideoCropping::test_crop_accuracy_motion_priority -v

# Run with parallel execution (unit tests only)
pytest tests/unit/ -n auto

# Run with coverage
pytest tests/ --cov=smart_crop --cov-report=html
```

## Dependencies

### Required for All Tests
- Python 3.11+
- pytest >= 7.4.0

### Additional for Comprehensive Tests 🆕
- Pillow >= 10.0.0 (image processing)
- numpy >= 1.24.0 (numerical analysis)

### Optional (Only for Regenerating Fixtures)
- FFmpeg (system package) - Only needed if regenerating test fixtures with `tests/generate_fixtures.py`

### Additional for Container Tests
- Docker
- docker-py >= 6.1.0

### Additional for API Tests
- requests >= 2.31.0

### Additional for Web UI Tests
- Playwright >= 1.40.0
- Chromium browser (installed via `playwright install`)

**Install all dependencies:**
```bash
pip install -r tests/requirements.txt
playwright install chromium --with-deps
```

## CI/CD Integration

### Fast Tests Workflow (`.github/workflows/test-smart-crop-video.yml`)

Runs on every push and pull request:
- Unit tests (286 tests) - parallel execution
- Integration tests (8 tests)
- Container tests (15 tests)
- API tests (19 tests)
- Diagnostic tests (1 test)
- Web UI tests (5 tests) - PRs only

**Total: 329-334 tests in ~8 minutes**

### Comprehensive Tests Workflow (`.github/workflows/test-smart-crop-video-comprehensive.yml`) 🆕

Runs on:
- Pull requests (required for merge)
- Releases
- Weekly schedule (Mondays at 6 AM UTC)
- Manual trigger

Tests:
- End-to-end video validation (10 tests)
- Acceleration features (9 tests)

**Total: 19 tests in ~10-15 minutes**

## Writing New Tests

### Unit Tests

Place in `tests/unit/`, use fast pure functions:

```python
def test_crop_dimensions_are_even():
    """Verify crop dimensions meet H.264 requirements."""
    dims = calculate_crop_dimensions(1920, 1080, 1, 1, crop_scale=0.75)
    assert dims.crop_w % 2 == 0
    assert dims.crop_h % 2 == 0
```

### Integration Tests

Place in `tests/integration/`, can use FFmpeg:

```python
@pytest.mark.skipif(not HAS_FFMPEG, reason="FFmpeg not available")
def test_parallel_analysis(test_video_path):
    """Test parallel position analysis."""
    results = analyze_positions_parallel(
        str(test_video_path),
        positions,
        max_workers=4
    )
    assert len(results) == len(positions)
```

### Comprehensive Tests 🆕

Place in `tests/integration/`, mark with `@pytest.mark.comprehensive`:

```python
@pytest.mark.skipif(not HAS_PILLOW, reason="Pillow not available")
@pytest.mark.skipif(not HAS_DOCKER, reason="Docker not available")
@pytest.mark.comprehensive
class TestEndToEndVideoCropping:
    """End-to-end tests validating video output correctness."""

    def test_crop_accuracy(self, motion_top_right_video, test_videos_dir):
        """Verify crop position matches expected region."""
        # Load pre-generated test video from fixtures
        # Process through smart-crop-video
        # Analyze output frames
        # Verify crop position
```

### Using Test Helpers 🆕

**Preferred approach (use pre-generated fixtures):**
```python
from pathlib import Path

# Load pre-generated test video
FIXTURES_DIR = Path(__file__).parent.parent / "fixtures"
test_video = FIXTURES_DIR / "motion_top_right.mov"

# Analyze output using frame_analyzer
from tests.helpers import frame_analyzer as fa
metadata = fa.get_video_metadata(video_path)
crop_x, crop_y = fa.get_crop_position_from_video(original, cropped)
frame = fa.extract_frame(video_path, timestamp=2.0)
```

**Advanced use only (regenerating fixtures):**
```python
# Note: video_generator is deprecated for normal test use
# Only use for regenerating fixtures or edge cases requiring dynamic generation
from tests.helpers import video_generator as vg

motion = vg.MotionRegion(x=1400, y=200, size=100, color="red", speed=100)
vg.create_video_with_motion_in_region(output_path, motion, config)
```

## Test Coverage Goals

| Component | Current Coverage | Goal |
|-----------|------------------|------|
| Core logic (dimensions, scoring) | ~95% | 95%+ |
| Analysis (FFmpeg integration) | ~80% | 85%+ |
| API endpoints | ~90% | 90%+ |
| Web UI workflow | ~70% | 75%+ |
| **End-to-end correctness** | **40% → 85%** 🆕 | **85%+** |
| **Acceleration features** | **30% → 80%** 🆕 | **80%+** |

## Troubleshooting

### Tests Fail Locally But Pass in CI

- **Docker environment**: Tests run on host in CI
- **Playwright browsers**: Run `playwright install chromium --with-deps`
- **Missing fixtures**: Ensure `tests/fixtures/` directory exists with test videos

### Comprehensive Tests Are Slower

- **Expected**: These tests process videos and analyze frames (5-10 minutes)
- **Optimize**: Run only affected tests during development with `pytest tests/integration/test_end_to_end_video.py::TestEndToEndVideoCropping::test_crop_accuracy_motion_priority -v`
- **CI**: These only run on PRs and releases, not every push

### Frame Analysis Errors

- **Missing Pillow**: `pip install Pillow numpy`
- **Missing fixtures**: Clone repository with `git lfs pull` or regenerate with `python3 tests/generate_fixtures.py`
- **Permission errors**: Check temp directory permissions

### Docker Tests Fail

- **Docker not running**: Start Docker daemon
- **Port conflicts**: Tests use dynamic ports, should auto-resolve
- **Image not built**: Run `docker build -t smart-crop-video:test .`

## Performance Benchmarks

| Test Suite | Test Count | Duration | Parallelization |
|------------|------------|----------|-----------------|
| Unit tests | 286 | ~30s | Yes (`-n auto`) |
| Integration | 8 | ~2min | No (FFmpeg) |
| Container | 15 | ~40s | No (Docker) |
| API | 19 | ~3min | No (Flask state) |
| Web UI | 5 | ~5min | No (Browser) |
| **End-to-end** 🆕 | **10** | **~8min** | **No (Video processing)** |
| **Acceleration** 🆕 | **9** | **~7min** | **No (Video processing)** |
| **Total** | **352** | **~25min** | **Mixed** |

**Fast tests only**: ~8 minutes
**Comprehensive only**: ~15 minutes
**All tests**: ~25 minutes

## Continuous Improvement

We continuously improve test coverage based on:
1. **User-reported bugs** → Add regression tests
2. **Edge cases discovered** → Add edge case tests
3. **New features** → Add feature tests
4. **Performance issues** → Add performance tests

Recent improvements:
- ✅ Added end-to-end crop accuracy validation (Nov 2024)
- ✅ Added acceleration feature comprehensive tests (Nov 2024)
- ✅ Added synthetic video generation for precise testing (Nov 2024)
- ✅ Added frame-level analysis utilities (Nov 2024)
- ✅ Migrated to pre-generated test fixtures, eliminating FFmpeg dependency for test execution (Nov 2024)

## Questions?

- **File an issue**: GitHub Issues
- **Check docs**: See main README.md
- **Run help**: `./run-tests.sh help`
