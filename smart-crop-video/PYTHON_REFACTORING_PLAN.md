# Python Refactoring Plan: smart-crop-video

**Goal**: Increase test coverage from 30% to 70%+ by refactoring into testable, modular components

**Timeline**: 2-3 days (16-24 hours)

**Status**: Planning Complete - Ready for Execution

---

## Executive Summary

This plan refactors `smart-crop-video.py` into a well-architected Python package with:
- **70%+ test coverage** (up from 30%)
- **4-8x performance improvement** via parallelization
- **Maintainable architecture** with clear separation of concerns
- **100% backward compatibility** - existing Docker usage unchanged

## Current State Analysis

### Code Metrics
- **Total lines**: 1,783
- **main() function**: 628 lines (35% of codebase in one function!)
- **Testable functions**: ~15 small utility functions
- **Untestable code**: ~1,100 lines (62%)
- **Functions**: 20 total, but only 8 are pure/testable

### Test Coverage
- **Container integration**: 100% (15/15 tests passing) ✓
- **API endpoints**: 42% (8/19 tests passing) ⚠️
- **Video analysis functions**: 0% coverage ✗
- **Scene detection**: 0% coverage ✗
- **Encoding logic**: 0% coverage ✗

### Architectural Problems
1. **Monolithic main()** - All logic in one 628-line function
2. **No dependency injection** - Direct subprocess calls everywhere
3. **Tight coupling** - Business logic mixed with I/O
4. **Sequential processing** - No parallelization
5. **Blocking I/O** - `input()` calls prevent automated testing
6. **Hard-coded dependencies** - Can't swap FFmpeg for mocks

---

## New File Structure

```
smart-crop-video/
├── smart-crop-video.py          # Main entry point (simplified to ~200 lines)
├── smart_crop/                  # New package
│   ├── __init__.py
│   ├── core/
│   │   ├── __init__.py
│   │   ├── dimensions.py       # Crop dimension calculations
│   │   ├── grid.py             # Position grid generation
│   │   ├── scoring.py          # Scoring strategies
│   │   └── candidates.py       # Candidate selection logic
│   ├── analysis/
│   │   ├── __init__.py
│   │   ├── analyzer.py         # Abstract analyzer interface
│   │   ├── ffmpeg.py           # FFmpeg implementation
│   │   ├── parallel.py         # Parallel analysis orchestration
│   │   └── metrics.py          # Metric extraction
│   ├── scene/
│   │   ├── __init__.py
│   │   ├── detector.py         # Scene detection
│   │   ├── segmentation.py     # Time-based segments
│   │   └── analysis.py         # Scene metric analysis
│   ├── encoding/
│   │   ├── __init__.py
│   │   ├── encoder.py          # Abstract encoder interface
│   │   ├── ffmpeg_encoder.py   # FFmpeg encoder implementation
│   │   └── variable_speed.py   # Variable speed encoding
│   ├── ui/
│   │   ├── __init__.py
│   │   ├── web_server.py       # Flask app (from create_app)
│   │   ├── state.py            # AppState class
│   │   └── selection.py        # User selection logic (web + CLI)
│   └── utils/
│       ├── __init__.py
│       ├── video_info.py       # Video metadata utilities
│       └── filesystem.py       # File operations
└── tests/
    ├── unit/                    # NEW: Unit tests
    │   ├── test_dimensions.py
    │   ├── test_grid.py
    │   ├── test_scoring.py
    │   ├── test_candidates.py
    │   ├── test_metrics.py
    │   ├── test_scene_detector.py
    │   └── test_selection_logic.py
    ├── integration/             # Existing tests moved here
    │   ├── test_container.py
    │   ├── test_api.py
    │   └── test_web_ui_focused.py
    └── mocks/
        ├── __init__.py
        └── mock_analyzer.py     # Mock video analyzer
```

---

## Phase-by-Phase Refactoring Plan

### Phase 1: Extract Pure Functions (4 hours)

**Goal**: Extract all pure, stateless functions into testable modules

#### 1.1: Create `smart_crop/core/dimensions.py`

**Extract from**: Lines 1204-1234 in main()

**New module**:
```python
# smart_crop/core/dimensions.py
"""
Crop dimension calculations - pure functions with no side effects.
"""
from typing import Tuple
from dataclasses import dataclass

@dataclass
class CropDimensions:
    """Result of crop dimension calculation"""
    crop_w: int
    crop_h: int
    max_crop_w: int
    max_crop_h: int
    max_x: int  # Movement range
    max_y: int  # Movement range

def parse_aspect_ratio(aspect_str: str) -> Tuple[int, int]:
    """Parse aspect ratio string like '1:1' or '16:9'"""
    parts = aspect_str.split(':')
    if len(parts) != 2:
        raise ValueError(f"Invalid aspect ratio: {aspect_str}")
    return int(parts[0]), int(parts[1])

def calculate_crop_dimensions(
    video_width: int,
    video_height: int,
    aspect_w: int,
    aspect_h: int,
    crop_scale: float = 0.75
) -> CropDimensions:
    """
    Calculate crop dimensions for a given video and aspect ratio.

    This is a PURE function - same inputs always produce same outputs.
    No I/O, no side effects - highly testable!

    Args:
        video_width: Original video width
        video_height: Original video height
        aspect_w: Target aspect ratio width component
        aspect_h: Target aspect ratio height component
        crop_scale: Scale factor (0.0-1.0) to reduce crop size

    Returns:
        CropDimensions with calculated values
    """
    # Calculate maximum possible crop dimensions
    if video_width < video_height:
        max_crop_w = video_width
        max_crop_h = video_width * aspect_h // aspect_w
        if max_crop_h > video_height:
            max_crop_h = video_height
            max_crop_w = video_height * aspect_w // aspect_h
    else:
        max_crop_h = video_height
        max_crop_w = video_height * aspect_w // aspect_h
        if max_crop_w > video_width:
            max_crop_w = video_width
            max_crop_h = video_width * aspect_h // aspect_w

    # Apply scale factor
    crop_w = int(max_crop_w * crop_scale)
    crop_h = int(max_crop_h * crop_scale)

    # Ensure even dimensions (required for H.264)
    crop_w = crop_w - (crop_w % 2)
    crop_h = crop_h - (crop_h % 2)

    # Calculate movement range
    max_x = video_width - crop_w
    max_y = video_height - crop_h

    return CropDimensions(
        crop_w=crop_w,
        crop_h=crop_h,
        max_crop_w=max_crop_w,
        max_crop_h=max_crop_h,
        max_x=max_x,
        max_y=max_y
    )
```

**Unit tests** (`tests/unit/test_dimensions.py`):
```python
import pytest
from smart_crop.core.dimensions import (
    calculate_crop_dimensions,
    parse_aspect_ratio,
    CropDimensions
)

def test_parse_aspect_ratio_valid():
    assert parse_aspect_ratio("1:1") == (1, 1)
    assert parse_aspect_ratio("16:9") == (16, 9)
    assert parse_aspect_ratio("4:5") == (4, 5)

def test_parse_aspect_ratio_invalid():
    with pytest.raises(ValueError):
        parse_aspect_ratio("invalid")
    with pytest.raises(ValueError):
        parse_aspect_ratio("1:2:3")

def test_calculate_crop_dimensions_square_video():
    """Test 1:1 crop on square 1000x1000 video"""
    dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=0.75)

    assert dims.crop_w == 750  # 1000 * 0.75, rounded to even
    assert dims.crop_h == 750
    assert dims.max_crop_w == 1000
    assert dims.max_crop_h == 1000
    assert dims.max_x == 250  # 1000 - 750
    assert dims.max_y == 250

def test_calculate_crop_dimensions_landscape_to_portrait():
    """Test 9:16 crop on 1920x1080 landscape video"""
    dims = calculate_crop_dimensions(1920, 1080, 9, 16, crop_scale=1.0)

    # Should use height as constraint
    assert dims.max_crop_h == 1080
    assert dims.max_crop_w == 1080 * 9 // 16  # 607
    assert dims.crop_w % 2 == 0  # Even
    assert dims.crop_h % 2 == 0  # Even

def test_calculate_crop_dimensions_portrait_to_square():
    """Test 1:1 crop on 1080x1920 portrait video"""
    dims = calculate_crop_dimensions(1080, 1920, 1, 1, crop_scale=0.5)

    # Should use width as constraint
    assert dims.max_crop_w == 1080
    assert dims.max_crop_h == 1080
    assert dims.crop_w == 540  # 1080 * 0.5
    assert dims.crop_h == 540

def test_dimensions_always_even():
    """Ensure dimensions are always even (H.264 requirement)"""
    # Test with values that would produce odd numbers
    dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=0.755)
    assert dims.crop_w % 2 == 0
    assert dims.crop_h % 2 == 0
```

**Estimated coverage gain**: +8% (120 lines of pure logic now testable)

---

#### 1.2: Create `smart_crop/core/grid.py`

**Extract from**: Lines 1258-1260

**New module**:
```python
# smart_crop/core/grid.py
"""
Position grid generation for crop analysis.
"""
from typing import List, Tuple
from dataclasses import dataclass

@dataclass
class Position:
    """A single position in the analysis grid"""
    x: int
    y: int

def generate_analysis_grid(
    max_x: int,
    max_y: int,
    grid_size: int = 5
) -> List[Position]:
    """
    Generate a uniform grid of positions to analyze.

    Pure function - easily testable!

    Args:
        max_x: Maximum x coordinate (video_width - crop_width)
        max_y: Maximum y coordinate (video_height - crop_height)
        grid_size: Number of positions per dimension (default: 5x5 = 25 total)

    Returns:
        List of Position objects in row-major order
    """
    if max_x <= 0 and max_y <= 0:
        # Crop fits exactly - only one position
        return [Position(0, 0)]

    # Generate grid positions (start from 1, not 0, to avoid edge artifacts)
    x_positions = [max(1, max_x * i // (grid_size - 1)) for i in range(grid_size)]
    y_positions = [max(1, max_y * i // (grid_size - 1)) for i in range(grid_size)]

    # Create all combinations
    positions = []
    for y in y_positions:
        for x in x_positions:
            positions.append(Position(x, y))

    return positions
```

**Unit tests**:
```python
def test_generate_grid_5x5():
    positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=5)
    assert len(positions) == 25

    # Check corners
    assert Position(1, 1) in positions  # Top-left
    assert Position(400, 300) in positions  # Bottom-right

def test_generate_grid_no_movement_needed():
    """When crop fits exactly, should return single position"""
    positions = generate_analysis_grid(max_x=0, max_y=0)
    assert len(positions) == 1
    assert positions[0] == Position(0, 0)

def test_grid_positions_are_uniform():
    positions = generate_analysis_grid(max_x=400, max_y=400, grid_size=5)
    x_coords = sorted(set(p.x for p in positions))

    # Should have 5 distinct x coordinates
    assert len(x_coords) == 5
    # Should be roughly evenly spaced
    spacing = [x_coords[i+1] - x_coords[i] for i in range(len(x_coords)-1)]
    assert all(abs(s - spacing[0]) <= 1 for s in spacing)  # Within 1 pixel
```

**Estimated coverage gain**: +3% (40 lines)

---

#### 1.3: Create `smart_crop/core/scoring.py`

**Extract from**: Lines 1123-1153 (normalize, score_with_strategy)

**New module**:
```python
# smart_crop/core/scoring.py
"""
Scoring strategies for crop position evaluation.
"""
from typing import Dict, List
from dataclasses import dataclass

@dataclass
class PositionMetrics:
    """Metrics for a single crop position"""
    x: int
    y: int
    motion: float
    complexity: float
    edges: float
    saturation: float

@dataclass
class NormalizationBounds:
    """Min/max values for normalization"""
    motion_min: float
    motion_max: float
    complexity_min: float
    complexity_max: float
    edges_min: float
    edges_max: float
    saturation_min: float
    saturation_max: float

    @classmethod
    def from_positions(cls, positions: List[PositionMetrics]) -> 'NormalizationBounds':
        """Calculate bounds from a list of positions"""
        return cls(
            motion_min=min(p.motion for p in positions),
            motion_max=max(p.motion for p in positions),
            complexity_min=min(p.complexity for p in positions),
            complexity_max=max(p.complexity for p in positions),
            edges_min=min(p.edges for p in positions),
            edges_max=max(p.edges for p in positions),
            saturation_min=min(p.saturation for p in positions),
            saturation_max=max(p.saturation for p in positions),
        )

def normalize(value: float, min_val: float, max_val: float) -> float:
    """
    Normalize value to 0-100 range.

    Pure function - perfect for unit testing!
    """
    if max_val - min_val > 0:
        return ((value - min_val) / (max_val - min_val)) * 100
    return 50.0

# Strategy definitions - configuration, not code!
STRATEGIES = {
    'Subject Detection': {
        'weights': {'motion': 0.05, 'complexity': 0.25, 'edges': 0.40, 'saturation': 0.30},
        'description': 'Finds people/objects (40% edges, 30% saturation)'
    },
    'Motion Priority': {
        'weights': {'motion': 0.50, 'complexity': 0.15, 'edges': 0.25, 'saturation': 0.10},
        'description': 'Tracks movement (50% motion, 25% edges)'
    },
    'Visual Detail': {
        'weights': {'motion': 0.05, 'complexity': 0.50, 'edges': 0.30, 'saturation': 0.15},
        'description': 'Identifies complex areas (50% complexity, 30% edges)'
    },
    'Balanced': {
        'weights': {'motion': 0.25, 'complexity': 0.25, 'edges': 0.25, 'saturation': 0.25},
        'description': 'Equal weights (25% each metric)'
    },
    'Color Focus': {
        'weights': {'motion': 0.05, 'complexity': 0.20, 'edges': 0.30, 'saturation': 0.45},
        'description': 'Colorful subjects (45% saturation, 30% edges)'
    },
}

def score_position(
    metrics: PositionMetrics,
    bounds: NormalizationBounds,
    strategy: str = 'Balanced'
) -> float:
    """
    Score a position using a specific strategy.

    Pure function - testable without FFmpeg!
    """
    if strategy not in STRATEGIES:
        raise ValueError(f"Unknown strategy: {strategy}")

    weights = STRATEGIES[strategy]['weights']

    # Normalize all metrics
    motion_norm = normalize(metrics.motion, bounds.motion_min, bounds.motion_max)
    complexity_norm = normalize(metrics.complexity, bounds.complexity_min, bounds.complexity_max)
    edges_norm = normalize(metrics.edges, bounds.edges_min, bounds.edges_max)
    saturation_norm = normalize(metrics.saturation, bounds.saturation_min, bounds.saturation_max)

    # Weighted sum
    score = (
        motion_norm * weights['motion'] +
        complexity_norm * weights['complexity'] +
        edges_norm * weights['edges'] +
        saturation_norm * weights['saturation']
    )

    return score

def get_available_strategies() -> List[str]:
    """Return list of available strategy names"""
    return list(STRATEGIES.keys())
```

**Unit tests**:
```python
def test_normalize_basic():
    assert normalize(5, 0, 10) == 50.0
    assert normalize(0, 0, 10) == 0.0
    assert normalize(10, 0, 10) == 100.0

def test_normalize_same_min_max():
    # Should return middle value
    assert normalize(5, 5, 5) == 50.0

def test_score_position_balanced():
    metrics = PositionMetrics(
        x=100, y=100,
        motion=5.0,
        complexity=10.0,
        edges=15.0,
        saturation=20.0
    )
    bounds = NormalizationBounds(
        motion_min=0, motion_max=10,
        complexity_min=0, complexity_max=20,
        edges_min=0, edges_max=30,
        saturation_min=0, saturation_max=40
    )

    score = score_position(metrics, bounds, 'Balanced')

    # Balanced: 50% + 50% + 50% + 50% / 4 = 50.0
    assert score == 50.0

def test_score_position_motion_priority():
    """Motion Priority should heavily weight motion metric"""
    metrics = PositionMetrics(
        x=100, y=100,
        motion=10.0,  # Max motion
        complexity=0.0,
        edges=0.0,
        saturation=0.0
    )
    bounds = NormalizationBounds(
        motion_min=0, motion_max=10,
        complexity_min=0, complexity_max=10,
        edges_min=0, edges_max=10,
        saturation_min=0, saturation_max=10
    )

    score = score_position(metrics, bounds, 'Motion Priority')
    # Should be heavily influenced by motion (50% weight)
    assert score == 50.0  # 100 * 0.5 + 0 * 0.15 + 0 * 0.25 + 0 * 0.10

def test_invalid_strategy_raises():
    metrics = PositionMetrics(100, 100, 1, 1, 1, 1)
    bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

    with pytest.raises(ValueError):
        score_position(metrics, bounds, 'NonexistentStrategy')
```

**Estimated coverage gain**: +5% (80 lines)

---

### Phase 2: Create Abstraction Layer for FFmpeg (6 hours)

**Goal**: Create interface for video analysis that can be mocked in tests

#### 2.1: Create `smart_crop/analysis/analyzer.py` (Interface)

```python
# smart_crop/analysis/analyzer.py
"""
Abstract interface for video analysis.

This allows us to swap FFmpeg implementation with mocks for testing!
"""
from abc import ABC, abstractmethod
from typing import Tuple, List
from smart_crop.core.scoring import PositionMetrics

class VideoAnalyzer(ABC):
    """Abstract video analyzer interface"""

    @abstractmethod
    def get_dimensions(self) -> Tuple[int, int]:
        """Get video width and height"""
        pass

    @abstractmethod
    def get_duration(self) -> float:
        """Get video duration in seconds"""
        pass

    @abstractmethod
    def get_fps(self) -> float:
        """Get video frame rate"""
        pass

    @abstractmethod
    def get_frame_count(self) -> int:
        """Get estimated total frame count"""
        pass

    @abstractmethod
    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """
        Analyze metrics for a specific crop position.

        Returns:
            PositionMetrics with motion, complexity, edges, saturation
        """
        pass

    @abstractmethod
    def extract_frame(
        self,
        timestamp: float,
        output_path: str,
        x: int = 0,
        y: int = 0,
        crop_w: int = None,
        crop_h: int = None
    ) -> None:
        """Extract a single frame as JPEG"""
        pass
```

#### 2.2: Create `smart_crop/analysis/ffmpeg.py` (Implementation)

**Move existing functions here**:
- `run_ffmpeg()` (line 555)
- `get_video_dimensions()` (line 566)
- `get_video_duration()` (line 580)
- `get_video_frame_count()` (line 592)
- `get_video_fps()` (line 622)
- `analyze_position()` (line 1072)
- `extract_metric_from_showinfo()` (line 1059)

```python
# smart_crop/analysis/ffmpeg.py
"""
FFmpeg-based video analyzer implementation.
"""
import subprocess
import re
from typing import List, Tuple
from smart_crop.analysis.analyzer import VideoAnalyzer
from smart_crop.core.scoring import PositionMetrics

class FFmpegAnalyzer(VideoAnalyzer):
    """Video analyzer using FFmpeg"""

    def __init__(self, video_path: str):
        self.video_path = video_path

    def get_dimensions(self) -> Tuple[int, int]:
        """Get video width and height"""
        cmd = [
            'ffprobe', '-v', 'error',
            '-select_streams', 'v:0',
            '-show_entries', 'stream=width,height',
            '-of', 'csv=p=0',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        width, height = map(int, result.stdout.strip().split(','))
        return width, height

    def get_duration(self) -> float:
        """Get video duration in seconds"""
        cmd = [
            'ffprobe', '-v', 'error',
            '-show_entries', 'format=duration',
            '-of', 'default=noprint_wrappers=1:nokey=1',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        return float(result.stdout.strip())

    def get_frame_count(self) -> int:
        """Estimate total number of frames"""
        fps = self.get_fps()
        duration = self.get_duration()
        return max(int(duration * fps), 1)

    def get_fps(self) -> float:
        """Get video frame rate"""
        cmd = [
            'ffprobe', '-v', 'error',
            '-select_streams', 'v:0',
            '-show_entries', 'stream=r_frame_rate',
            '-of', 'csv=p=0',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        fps_str = result.stdout.strip()

        try:
            if '/' in fps_str:
                num, den = map(int, fps_str.split('/'))
                return num / den
            else:
                return float(fps_str)
        except (ValueError, ZeroDivisionError):
            return 24.0  # Default fallback

    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """Analyze a crop position and return its metrics"""
        # Get motion and complexity from showinfo
        cmd = [
            'ffmpeg', '-i', self.video_path,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},showinfo',
            '-frames:v', str(sample_frames),
            '-f', 'null', '-'
        ]
        stats_output = self._run_ffmpeg(cmd)

        # Extract motion (frame-to-frame differences in mean)
        means = self._extract_metric_from_showinfo(stats_output, 'mean')
        motion = 0.0
        if len(means) > 1:
            diffs = [abs(means[i] - means[i-1]) for i in range(1, len(means))]
            motion = sum(diffs) / len(diffs) if diffs else 0.0

        # Extract complexity (standard deviation)
        stdevs = self._extract_metric_from_showinfo(stats_output, 'stdev')
        complexity = sum(stdevs) / len(stdevs) if stdevs else 0.0

        # Get edge detection score
        cmd = [
            'ffmpeg', '-i', self.video_path,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},edgedetect=low=0.3:high=0.4:mode=colormix,showinfo',
            '-frames:v', str(sample_frames),
            '-f', 'null', '-'
        ]
        edge_output = self._run_ffmpeg(cmd)
        edge_means = self._extract_metric_from_showinfo(edge_output, 'mean')
        edges = sum(edge_means) / len(edge_means) if edge_means else 0.0

        # Color variance (sum of RGB channel standard deviations)
        stdev_pattern = r'stdev:\[([0-9. ]+)\]'
        stdev_matches = re.findall(stdev_pattern, stats_output)
        color_variance = 0.0
        if stdev_matches:
            rgb_sums = []
            for match in stdev_matches:
                values = list(map(float, match.split()))
                if len(values) >= 3:
                    rgb_sums.append(sum(values[:3]))  # R, G, B
            color_variance = sum(rgb_sums) / len(rgb_sums) if rgb_sums else 0.0

        return PositionMetrics(x, y, motion, complexity, edges, color_variance)

    def extract_frame(
        self,
        timestamp: float,
        output_path: str,
        x: int = 0,
        y: int = 0,
        crop_w: int = None,
        crop_h: int = None
    ) -> None:
        """Extract a single frame as JPEG"""
        vf_parts = []
        if crop_w and crop_h:
            vf_parts.append(f'crop={crop_w}:{crop_h}:{x}:{y}')

        cmd = [
            'ffmpeg', '-ss', str(timestamp), '-i', self.video_path,
        ]

        if vf_parts:
            cmd.extend(['-vf', ','.join(vf_parts)])

        cmd.extend([
            '-vframes', '1', '-q:v', '2',
            output_path, '-y'
        ])

        subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    def _run_ffmpeg(self, cmd: List[str]) -> str:
        """Run ffmpeg command and return stderr output"""
        result = subprocess.run(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        return result.stderr

    def _extract_metric_from_showinfo(self, output: str, metric: str) -> List[float]:
        """Extract metric values from ffmpeg showinfo filter output"""
        pattern = rf'{metric}:\[([0-9. ]+)\]'
        matches = re.findall(pattern, output)
        values = []
        for match in matches:
            # Take first value from each match (for Y channel in YUV)
            parts = match.split()
            if parts:
                values.append(float(parts[0]))
        return values
```

#### 2.3: Create Mock Analyzer for Tests

```python
# tests/mocks/mock_analyzer.py
"""
Mock video analyzer for testing without FFmpeg.
"""
from smart_crop.analysis.analyzer import VideoAnalyzer
from smart_crop.core.scoring import PositionMetrics
from typing import Dict, Tuple

class MockAnalyzer(VideoAnalyzer):
    """
    Mock analyzer that returns pre-configured values.

    Perfect for testing scoring logic without video files!
    """

    def __init__(
        self,
        dimensions: Tuple[int, int] = (1920, 1080),
        duration: float = 30.0,
        fps: float = 30.0,
        position_metrics: Dict[Tuple[int, int], PositionMetrics] = None
    ):
        self.dimensions = dimensions
        self.duration = duration
        self.fps = fps
        self.position_metrics = position_metrics or {}
        self.analyze_position_calls = []  # Track what was analyzed

    def get_dimensions(self) -> Tuple[int, int]:
        return self.dimensions

    def get_duration(self) -> float:
        return self.duration

    def get_fps(self) -> float:
        return self.fps

    def get_frame_count(self) -> int:
        return int(self.duration * self.fps)

    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """Return pre-configured metrics or defaults"""
        self.analyze_position_calls.append((x, y))

        key = (x, y)
        if key in self.position_metrics:
            return self.position_metrics[key]

        # Return default metrics based on position
        # (makes testing deterministic)
        return PositionMetrics(
            x=x,
            y=y,
            motion=float(x + y) / 10,
            complexity=float(x * y) / 100,
            edges=float(x) / 5,
            saturation=float(y) / 5
        )

    def extract_frame(self, timestamp, output_path, x=0, y=0, crop_w=None, crop_h=None):
        # Create a dummy JPEG file for testing
        with open(output_path, 'wb') as f:
            f.write(b'\xFF\xD8\xFF\xE0')  # JPEG magic bytes
```

**Now you can test scoring without FFmpeg!**:
```python
def test_candidate_selection_with_mock():
    """Test candidate selection logic without running FFmpeg"""
    from smart_crop.core.candidates import select_top_candidates
    from tests.mocks.mock_analyzer import MockAnalyzer

    # Create mock with predefined "interesting" positions
    analyzer = MockAnalyzer(
        position_metrics={
            (100, 100): PositionMetrics(100, 100, motion=10.0, complexity=8.0, edges=9.0, saturation=7.0),
            (200, 200): PositionMetrics(200, 200, motion=5.0, complexity=9.0, edges=6.0, saturation=8.0),
            (300, 300): PositionMetrics(300, 300, motion=3.0, complexity=5.0, edges=7.0, saturation=9.0),
        }
    )

    candidates = select_top_candidates(analyzer, grid_size=3, crop_w=100, crop_h=100)

    # Should select (100, 100) for Motion Priority due to high motion
    motion_candidate = next(c for c in candidates if c.strategy == 'Motion Priority')
    assert motion_candidate.x == 100
    assert motion_candidate.y == 100
```

**Estimated coverage gain**: +15% (interface abstraction enables ~250 lines of business logic testing)

---

### Phase 3: Add Parallelization (3 hours)

**Goal**: Speed up analysis 4-8x with multiprocessing

#### 3.1: Create `smart_crop/analysis/parallel.py`

```python
# smart_crop/analysis/parallel.py
"""
Parallel video analysis using multiprocessing.
"""
from multiprocessing import Pool, cpu_count
from typing import List, Callable
from smart_crop.core.grid import Position
from smart_crop.core.scoring import PositionMetrics

def _analyze_single_position(args):
    """
    Worker function for multiprocessing.

    Must be top-level function (not lambda) for pickling.
    """
    video_path, x, y, crop_w, crop_h, sample_frames = args

    # Import here to avoid pickling issues
    from smart_crop.analysis.ffmpeg import FFmpegAnalyzer

    analyzer = FFmpegAnalyzer(video_path)
    return analyzer.analyze_position(x, y, crop_w, crop_h, sample_frames)

def analyze_positions_parallel(
    video_path: str,
    positions: List[Position],
    crop_w: int,
    crop_h: int,
    sample_frames: int = 50,
    max_workers: int = None,
    progress_callback: Callable[[int, int], None] = None
) -> List[PositionMetrics]:
    """
    Analyze multiple positions in parallel.

    Args:
        video_path: Path to video file
        positions: List of positions to analyze
        crop_w, crop_h: Crop dimensions
        sample_frames: Frames to sample per position
        max_workers: Number of worker processes (default: cpu_count())
        progress_callback: Optional callback(current, total) for progress updates

    Returns:
        List of PositionMetrics in same order as positions
    """
    if max_workers is None:
        max_workers = min(cpu_count(), len(positions))

    # Prepare arguments for each worker
    args_list = [
        (video_path, pos.x, pos.y, crop_w, crop_h, sample_frames)
        for pos in positions
    ]

    if max_workers == 1 or len(positions) == 1:
        # Sequential for debugging or single position
        results = []
        for i, args in enumerate(args_list):
            results.append(_analyze_single_position(args))
            if progress_callback:
                progress_callback(i + 1, len(args_list))
        return results

    # Parallel execution
    with Pool(processes=max_workers) as pool:
        if progress_callback:
            # Use imap for progress updates
            results = []
            for i, result in enumerate(pool.imap(_analyze_single_position, args_list)):
                results.append(result)
                progress_callback(i + 1, len(args_list))
            return results
        else:
            # Use map for simplicity
            return pool.map(_analyze_single_position, args_list)
```

**Usage in main script**:
```python
# OLD (sequential - slow):
for y in y_positions:
    for x in x_positions:
        pos = analyze_position(input_file, x, y, crop_w, crop_h, analysis_frames)
        positions.append(pos)

# NEW (parallel - 4-8x faster):
from smart_crop.analysis.parallel import analyze_positions_parallel
from smart_crop.core.grid import generate_analysis_grid

grid_positions = generate_analysis_grid(max_x, max_y, grid_size=5)
position_metrics = analyze_positions_parallel(
    video_path=input_file,
    positions=grid_positions,
    crop_w=crop_w,
    crop_h=crop_h,
    sample_frames=analysis_frames,
    progress_callback=lambda current, total: state.update(
        current_position=current,
        progress=int(current * 100 / total),
        message=f"Analyzing position {current}/{total}"
    )
)
```

**Unit tests**:
```python
def test_parallel_analysis_returns_correct_count():
    """Ensure parallel analysis returns same number of results as positions"""
    from smart_crop.analysis.parallel import analyze_positions_parallel
    from smart_crop.core.grid import Position

    positions = [Position(x, y) for x in range(3) for y in range(3)]

    # Use test video
    results = analyze_positions_parallel(
        video_path="example_movie.mov",
        positions=positions,
        crop_w=100,
        crop_h=100,
        sample_frames=5,
        max_workers=2
    )

    assert len(results) == len(positions)
    assert all(isinstance(r, PositionMetrics) for r in results)

def test_sequential_and_parallel_produce_same_results():
    """Verify parallel gives same results as sequential"""
    positions = [Position(100, 100), Position(200, 200)]

    # Sequential
    seq_results = analyze_positions_parallel(
        video_path="example_movie.mov",
        positions=positions,
        crop_w=100,
        crop_h=100,
        sample_frames=5,
        max_workers=1  # Force sequential
    )

    # Parallel
    par_results = analyze_positions_parallel(
        video_path="example_movie.mov",
        positions=positions,
        crop_w=100,
        crop_h=100,
        sample_frames=5,
        max_workers=2
    )

    # Results should be identical (or very close for floating point)
    for seq, par in zip(seq_results, par_results):
        assert abs(seq.motion - par.motion) < 0.01
        assert abs(seq.complexity - par.complexity) < 0.01
```

**Performance impact**: 4-8x faster analysis (25 positions in ~15-30s instead of 60-180s)

**Estimated coverage gain**: +5% (100 lines, highly testable)

---

### Phase 4: Extract Candidate Selection Logic (2 hours)

**Goal**: Make candidate selection testable

#### 4.1: Create `smart_crop/core/candidates.py`

**Extract from**: Lines 1304-1358

```python
# smart_crop/core/candidates.py
"""
Candidate crop selection logic.
"""
from typing import List
from dataclasses import dataclass
from smart_crop.core.scoring import (
    PositionMetrics,
    NormalizationBounds,
    score_position,
    get_available_strategies
)

@dataclass
class ScoredCandidate:
    """A candidate crop with its score and strategy"""
    x: int
    y: int
    score: float
    strategy: str

def generate_candidates_from_strategies(
    positions: List[PositionMetrics],
    strategies: List[str] = None,
    top_n_per_strategy: int = 5
) -> List[ScoredCandidate]:
    """
    Generate candidate crops using multiple strategies.

    Pure function - testable!
    """
    if strategies is None:
        strategies = get_available_strategies()

    bounds = NormalizationBounds.from_positions(positions)
    candidates = []

    for strategy in strategies:
        # Score all positions with this strategy
        scored = [
            (pos, score_position(pos, bounds, strategy))
            for pos in positions
        ]
        scored.sort(key=lambda x: x[1], reverse=True)

        # Take top N
        for pos, score in scored[:top_n_per_strategy]:
            candidates.append(ScoredCandidate(
                x=pos.x,
                y=pos.y,
                score=score,
                strategy=strategy
            ))

    return candidates

def add_spatial_diversity_candidates(
    positions: List[PositionMetrics],
    video_width: int,
    video_height: int
) -> List[ScoredCandidate]:
    """
    Add candidates from different spatial regions.

    Ensures coverage across different areas of the frame.
    """
    bounds = NormalizationBounds.from_positions(positions)
    center_x = video_width // 2
    center_y = video_height // 2

    quadrants = {
        'Top-Left': lambda p: p.x < center_x and p.y < center_y,
        'Top-Right': lambda p: p.x >= center_x and p.y < center_y,
        'Bottom-Left': lambda p: p.x < center_x and p.y >= center_y,
        'Bottom-Right': lambda p: p.x >= center_x and p.y >= center_y,
        'Center': lambda p: abs(p.x - center_x) < video_width//4 and abs(p.y - center_y) < video_height//4,
    }

    candidates = []
    for quad_name, condition in quadrants.items():
        scored = [
            (p, score_position(p, bounds, 'Balanced'))
            for p in positions if condition(p)
        ]
        if scored:
            scored.sort(key=lambda x: x[1], reverse=True)
            pos, score = scored[0]
            candidates.append(ScoredCandidate(
                x=pos.x,
                y=pos.y,
                score=score,
                strategy=f"Spatial:{quad_name}"
            ))

    return candidates

def deduplicate_candidates(
    candidates: List[ScoredCandidate],
    max_candidates: int = 10
) -> List[ScoredCandidate]:
    """
    Remove duplicate positions, keeping highest-scoring strategy.
    """
    seen = set()
    unique = []

    for cand in sorted(candidates, key=lambda c: c.score, reverse=True):
        key = (cand.x, cand.y)
        if key not in seen and cand.x > 0 and cand.y > 0:
            seen.add(key)
            unique.append(cand)
            if len(unique) >= max_candidates:
                break

    return unique
```

**Unit tests**:
```python
def test_generate_candidates_returns_correct_count():
    """Should return top_n_per_strategy × num_strategies candidates"""
    positions = [
        PositionMetrics(i*10, i*10, i, i, i, i)
        for i in range(10)
    ]

    candidates = generate_candidates_from_strategies(
        positions,
        strategies=['Balanced', 'Motion Priority'],
        top_n_per_strategy=3
    )

    assert len(candidates) == 6  # 2 strategies × 3 per strategy

def test_spatial_diversity_covers_all_quadrants():
    """Should generate candidates from all spatial regions"""
    # Create grid covering all quadrants
    positions = [
        PositionMetrics(100, 100, 5, 5, 5, 5),  # Top-left
        PositionMetrics(900, 100, 5, 5, 5, 5),  # Top-right
        PositionMetrics(100, 500, 5, 5, 5, 5),  # Bottom-left
        PositionMetrics(900, 500, 5, 5, 5, 5),  # Bottom-right
        PositionMetrics(500, 300, 5, 5, 5, 5),  # Center
    ]

    candidates = add_spatial_diversity_candidates(
        positions,
        video_width=1000,
        video_height=600
    )

    strategies = [c.strategy for c in candidates]
    assert 'Spatial:Top-Left' in strategies
    assert 'Spatial:Top-Right' in strategies
    assert 'Spatial:Center' in strategies

def test_deduplicate_removes_duplicates():
    """Should remove duplicate positions"""
    candidates = [
        ScoredCandidate(100, 100, 90, 'Strategy1'),
        ScoredCandidate(100, 100, 80, 'Strategy2'),  # Duplicate position
        ScoredCandidate(200, 200, 85, 'Strategy3'),
    ]

    unique = deduplicate_candidates(candidates, max_candidates=10)

    assert len(unique) == 2
    # Should keep higher score for duplicate position
    assert unique[0].score == 90
```

**Estimated coverage gain**: +10% (150 lines of pure logic)

---

### Phase 5: Extract Scene Detection & User Selection (3 hours)

#### 5.1: Create `smart_crop/scene/detector.py`

**Move from main script**:
- Lines 660-733 (`detect_scenes`, `create_time_based_segments`)
- Scene dataclass (line 645)

```python
# smart_crop/scene/detector.py
"""
Scene detection and segmentation.
"""
from typing import List
from dataclasses import dataclass
import subprocess
import re

@dataclass
class Scene:
    """Represents a scene in the video"""
    start_time: float
    end_time: float
    start_frame: int
    end_frame: int
    metric_value: float = 0.0
    first_frame_path: str = ""
    last_frame_path: str = ""

    @property
    def duration(self) -> float:
        return self.end_time - self.start_time

def detect_scenes(input_file: str, threshold: float = 0.3) -> List[Scene]:
    """Detect scene changes in video using FFmpeg's scene detection"""
    # Implementation from lines 660-697
    pass

def create_time_based_segments(input_file: str, segment_duration: float = 5.0) -> List[Scene]:
    """Create fixed-duration segments when scene detection doesn't find enough scenes"""
    # Implementation from lines 701-733
    pass
```

#### 5.2: Create `smart_crop/ui/selection.py`

**Extract from**: Lines 1412-1632 (user selection logic)

```python
# smart_crop/ui/selection.py
"""
User selection handling - web UI and CLI.
"""
from typing import Optional, List
import sys
import time
import select
import os

class SelectionManager:
    """Manages user selection from multiple sources (web UI, CLI, auto)"""

    def __init__(self, state, timeout: int = 300):
        self.state = state
        self.timeout = timeout
        self.is_automated = self._check_automated_mode()

    def _check_automated_mode(self) -> bool:
        """Check if running in automated mode (no TTY or env var set)"""
        if os.getenv('AUTOMATED_MODE') == 'true':
            return True
        return not sys.stdin.isatty()

    def wait_for_selection(
        self,
        candidates: List,
        default_index: int = 0
    ) -> int:
        """
        Wait for user to select a candidate.

        Returns: selected index (0-based)
        """
        if self.is_automated:
            return default_index

        # Check if web UI already made selection
        web_selection = self.state.get('selected_index')
        if web_selection is not None:
            return web_selection - 1  # Convert from 1-based to 0-based

        # Interactive polling (extract from lines 1430-1475)
        # ...
```

**This makes selection logic testable!**:
```python
def test_selection_manager_automated_mode():
    """In automated mode, should return default without waiting"""
    os.environ['AUTOMATED_MODE'] = 'true'

    state = AppState()
    manager = SelectionManager(state, timeout=1)

    candidates = [Mock(), Mock(), Mock()]
    selected_idx = manager.wait_for_selection(candidates, default_index=1)

    assert selected_idx == 1  # Should use default

def test_selection_manager_uses_web_ui_choice():
    """Should use web UI selection if available"""
    state = AppState()
    state.update(selected_index=3)  # Web UI selected option 3

    manager = SelectionManager(state)
    candidates = [Mock()] * 5
    selected_idx = manager.wait_for_selection(candidates)

    assert selected_idx == 2  # 3-1 (convert to 0-based)
```

**Estimated coverage gain**: +8% (120 lines)

---

### Phase 6: Extract Encoding Logic (2 hours)

#### 6.1: Create `smart_crop/encoding/encoder.py` (Interface)

```python
# smart_crop/encoding/encoder.py
"""
Abstract encoder interface.
"""
from abc import ABC, abstractmethod
from typing import List, Callable
from smart_crop.scene.detector import Scene

class VideoEncoder(ABC):
    """Abstract video encoder"""

    @abstractmethod
    def encode_simple(
        self,
        input_path: str,
        output_path: str,
        crop_x: int,
        crop_y: int,
        crop_w: int,
        crop_h: int,
        preset: str = 'medium',
        progress_callback: Callable[[int], None] = None
    ) -> None:
        """Encode video with simple crop"""
        pass

    @abstractmethod
    def encode_variable_speed(
        self,
        input_path: str,
        output_path: str,
        crop_x: int,
        crop_y: int,
        crop_w: int,
        crop_h: int,
        scenes: List[Scene],
        scene_speedups: dict,
        preset: str = 'medium'
    ) -> None:
        """Encode video with variable speed for scenes"""
        pass
```

#### 6.2: Create `smart_crop/encoding/ffmpeg_encoder.py`

**Move from main script**:
- Lines 915-997 (`encode_with_variable_speed`)
- Lines 1642-1743 (normal encoding logic from main)

**Estimated coverage gain**: +5% (can test encoding logic with mock encoder)

---

## Summary of Coverage Improvements

| Phase | Module | Lines | Tests | Coverage Gain |
|-------|--------|-------|-------|---------------|
| 1.1 | dimensions.py | 120 | 6 tests | +8% |
| 1.2 | grid.py | 40 | 3 tests | +3% |
| 1.3 | scoring.py | 80 | 5 tests | +5% |
| 2 | analyzer.py + ffmpeg.py + mocks | 350 | 8 tests | +15% |
| 3 | parallel.py | 100 | 3 tests | +5% |
| 4 | candidates.py | 150 | 4 tests | +10% |
| 5 | selection.py + detector.py | 200 | 6 tests | +8% |
| 6 | encoder.py + ffmpeg_encoder.py | 150 | 4 tests | +5% |
| **Total** | **8 new modules** | **~1,190 lines** | **39 unit tests** | **+59%** |

**Final Coverage**: 30% (current) + 59% = **89% total coverage**

---

## Testing Strategy

### Unit Tests (NEW - Phase by Phase)
```
tests/unit/
├── test_dimensions.py         # Phase 1.1 - Pure dimension calculations
├── test_grid.py                # Phase 1.2 - Grid generation
├── test_scoring.py             # Phase 1.3 - Scoring strategies
├── test_metrics.py             # Phase 2 - Metric extraction
├── test_candidates.py          # Phase 4 - Candidate selection
├── test_selection_logic.py     # Phase 5 - User selection
└── test_encoder_logic.py       # Phase 6 - Encoding coordination
```

### Integration Tests (EXISTING - Keep As-Is)
```
tests/integration/
├── test_container.py    # ✓ Already passing (100%)
├── test_api.py          # Will pass after Phase 5 (automated mode)
└── test_web_ui_focused.py
```

### Mock Infrastructure
```
tests/mocks/
├── mock_analyzer.py     # Mock video analyzer (no FFmpeg needed)
└── mock_encoder.py      # Mock encoder (no actual encoding)
```

---

## Execution Timeline

### Week 1: Refactoring

**Monday (8 hours)**
- Phase 1.1: Extract dimensions.py (2 hours)
- Phase 1.2: Extract grid.py (1 hour)
- Phase 1.3: Extract scoring.py (2 hours)
- Write unit tests for Phase 1 (3 hours)

**Tuesday (8 hours)**
- Phase 2: Create analyzer abstraction (4 hours)
- Create mock analyzer (1 hour)
- Write unit tests for Phase 2 (3 hours)

**Wednesday (8 hours)**
- Phase 3: Add parallelization (3 hours)
- Phase 4: Extract candidates (2 hours)
- Write unit tests for Phases 3-4 (3 hours)

**Thursday (8 hours)**
- Phase 5: Scene detection & selection (3 hours)
- Phase 6: Encoding abstraction (2 hours)
- Write unit tests for Phases 5-6 (3 hours)

**Friday (8 hours)**
- Integration testing (3 hours)
- Fix integration test failures (2 hours)
- Performance benchmarking (2 hours)
- Documentation updates (1 hour)

**Total**: 40 hours over 5 days

---

## Risk Mitigation

### 1. Backward Compatibility
**Risk**: Refactoring breaks existing functionality
**Mitigation**:
- Keep existing `smart-crop-video.py` working throughout
- Each phase is independently testable
- Run integration tests after each phase
- Use git branches for each phase

### 2. Test Failures
**Risk**: New tests reveal existing bugs
**Mitigation**:
- Fix bugs as discovered (they're real issues!)
- Document any intentional behavior changes
- Get user feedback on edge cases

### 3. Performance Regression
**Risk**: Abstraction layers add overhead
**Mitigation**:
- Benchmark before/after each phase
- Parallelization should make it faster overall
- Profile if needed

### 4. Merge Conflicts
**Risk**: Ongoing development conflicts with refactoring
**Mitigation**:
- Complete refactoring in dedicated branch
- Minimize feature additions during refactoring
- Small, frequent commits

---

## Success Metrics

### Code Quality
- [ ] Test coverage ≥ 70%
- [ ] No function > 50 lines
- [ ] main() function < 200 lines
- [ ] All business logic has unit tests

### Performance
- [ ] Analysis time ≤ 30s (from 60-180s)
- [ ] Test suite runs in < 5 minutes
- [ ] Integration tests pass 100%

### Testability
- [ ] 39+ unit tests added
- [ ] All pure functions have tests
- [ ] Mock analyzer works correctly
- [ ] AUTOMATED_MODE enables non-interactive tests

---

## Post-Refactoring Benefits

1. **Faster Development**: Pure functions are easy to modify and test
2. **Faster Execution**: 4-8x speedup from parallelization
3. **Better Reliability**: 70%+ test coverage catches regressions
4. **Easier Debugging**: Small, focused functions
5. **Future Go Migration**: Clean architecture makes porting easier

---

## Environment Variables for Testing

Add support for these environment variables to enable automated testing:

```bash
# Enable non-interactive mode (no input() prompts)
AUTOMATED_MODE=true

# Use default selections
AUTO_SELECT_CROP=1  # Auto-select first candidate
AUTO_ENABLE_ACCELERATION=false  # Disable acceleration by default

# Fast testing
PRESET=ultrafast
ANALYSIS_FRAMES=10
CROP_SCALE=0.75
```

---

## Next Steps After Refactoring

1. **Add More Strategies**: Easy to add new scoring strategies
2. **Machine Learning**: Could replace scoring with ML model (same interface!)
3. **GPU Acceleration**: Could create GPU analyzer implementation
4. **Go Port**: Clean architecture makes it straightforward
5. **Cloud Deployment**: Easier to containerize and scale

---

## Getting Started

To begin execution:

1. **Create feature branch**:
   ```bash
   git checkout -b refactor/python-architecture
   ```

2. **Create package structure**:
   ```bash
   mkdir -p smart_crop/{core,analysis,scene,encoding,ui,utils}
   touch smart_crop/__init__.py
   touch smart_crop/core/__init__.py
   # ... etc
   ```

3. **Start with Phase 1.1**:
   ```bash
   # Create dimensions module
   touch smart_crop/core/dimensions.py

   # Create test file
   mkdir -p tests/unit
   touch tests/unit/test_dimensions.py
   ```

4. **Run tests frequently**:
   ```bash
   pytest tests/unit/test_dimensions.py -v
   ```

---

**Ready to begin? Start with Phase 1.1!**
