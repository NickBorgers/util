"""
Mock video analyzer for testing without FFmpeg.

This module provides a mock implementation of VideoAnalyzer that returns
pre-configured values instead of actually analyzing videos. This enables:

1. Fast unit tests (no FFmpeg subprocess calls)
2. Deterministic tests (same inputs → same outputs)
3. Testing edge cases (can configure any metrics you want)
4. Testing without FFmpeg installed
"""
from typing import Dict, Tuple
from smart_crop.analysis.analyzer import VideoAnalyzer
from smart_crop.core.scoring import PositionMetrics


class MockAnalyzer(VideoAnalyzer):
    """
    Mock analyzer that returns pre-configured values.

    Perfect for testing scoring logic, candidate selection, and other
    business logic without needing actual video files or FFmpeg!

    Args:
        dimensions: Video (width, height). Default: (1920, 1080)
        duration: Video duration in seconds. Default: 30.0
        fps: Frame rate. Default: 30.0
        position_metrics: Pre-configured metrics for specific positions.
                         Dict mapping (x, y) -> PositionMetrics.
                         If a position isn't in the dict, a default metric
                         based on position is generated.

    Examples:
        >>> # Simple mock with defaults
        >>> mock = MockAnalyzer()
        >>> mock.get_dimensions()
        (1920, 1080)

        >>> # Mock with specific position metrics
        >>> metrics_map = {
        ...     (100, 100): PositionMetrics(100, 100, motion=10.0, complexity=8.0, edges=9.0, saturation=7.0),
        ...     (200, 200): PositionMetrics(200, 200, motion=5.0, complexity=9.0, edges=6.0, saturation=8.0),
        ... }
        >>> mock = MockAnalyzer(position_metrics=metrics_map)
        >>> result = mock.analyze_position(100, 100, 640, 640)
        >>> result.motion
        10.0

        >>> # Test that analyze was called
        >>> len(mock.analyze_position_calls)
        1
        >>> mock.analyze_position_calls[0]
        (100, 100)
    """

    def __init__(
        self,
        dimensions: Tuple[int, int] = (1920, 1080),
        duration: float = 30.0,
        fps: float = 30.0,
        position_metrics: Dict[Tuple[int, int], PositionMetrics] = None
    ):
        """
        Initialize mock analyzer with pre-configured values.

        Args:
            dimensions: Video dimensions as (width, height)
            duration: Video duration in seconds
            fps: Frame rate
            position_metrics: Optional dict mapping (x,y) positions to PositionMetrics.
                            If not provided, default deterministic metrics are generated.
        """
        self.dimensions = dimensions
        self.duration = duration
        self.fps = fps
        self.position_metrics = position_metrics or {}

        # Track which methods were called (useful for testing)
        self.get_dimensions_calls = 0
        self.get_duration_calls = 0
        self.get_fps_calls = 0
        self.get_frame_count_calls = 0
        self.analyze_position_calls = []  # List of (x, y) tuples
        self.extract_frame_calls = []     # List of (timestamp, output_path, x, y, crop_w, crop_h)

    def get_dimensions(self) -> Tuple[int, int]:
        """
        Return mock video dimensions.

        Returns:
            Configured dimensions tuple
        """
        self.get_dimensions_calls += 1
        return self.dimensions

    def get_duration(self) -> float:
        """
        Return mock video duration.

        Returns:
            Configured duration in seconds
        """
        self.get_duration_calls += 1
        return self.duration

    def get_fps(self) -> float:
        """
        Return mock video frame rate.

        Returns:
            Configured fps
        """
        self.get_fps_calls += 1
        return self.fps

    def get_frame_count(self) -> int:
        """
        Calculate mock frame count.

        Returns:
            duration × fps (rounded to int, at least 1)
        """
        self.get_frame_count_calls += 1
        return max(int(self.duration * self.fps), 1)

    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """
        Return pre-configured or generated metrics for a position.

        If the position was pre-configured in position_metrics dict,
        returns that. Otherwise, generates deterministic metrics based
        on the position coordinates.

        The default generation formula creates varied but predictable
        metrics that can be used for testing scoring strategies:
        - motion = (x + y) / 10
        - complexity = (x * y) / 100
        - edges = x / 5
        - saturation = y / 5

        Args:
            x: X coordinate
            y: Y coordinate
            crop_w: Crop width (ignored in mock)
            crop_h: Crop height (ignored in mock)
            sample_frames: Number of frames (ignored in mock)

        Returns:
            PositionMetrics for this position
        """
        # Track that this position was analyzed
        self.analyze_position_calls.append((x, y))

        # Return pre-configured metrics if available
        key = (x, y)
        if key in self.position_metrics:
            return self.position_metrics[key]

        # Generate deterministic default metrics based on position
        # This creates a gradient of values that's useful for testing
        return PositionMetrics(
            x=x,
            y=y,
            motion=float(x + y) / 10,
            complexity=float(x * y) / 100,
            edges=float(x) / 5,
            saturation=float(y) / 5
        )

    def extract_frame(
        self,
        timestamp: float,
        output_path: str,
        x: int = 0,
        y: int = 0,
        crop_w: int = None,
        crop_h: int = None
    ) -> None:
        """
        Mock frame extraction - creates a dummy JPEG file.

        Creates a small file with JPEG magic bytes at the output path.
        This is enough to make code that checks for file existence work.

        Args:
            timestamp: Time in seconds (recorded but not used)
            output_path: Path where dummy JPEG should be created
            x: Crop X (recorded but not used)
            y: Crop Y (recorded but not used)
            crop_w: Crop width (recorded but not used)
            crop_h: Crop height (recorded but not used)
        """
        # Track the call
        self.extract_frame_calls.append((timestamp, output_path, x, y, crop_w, crop_h))

        # Create a dummy JPEG file (just the magic bytes)
        # This makes file existence checks pass
        with open(output_path, 'wb') as f:
            # JPEG magic bytes: FF D8 FF E0
            f.write(b'\xFF\xD8\xFF\xE0')

    def reset_call_tracking(self) -> None:
        """
        Reset all call tracking counters.

        Useful when running multiple tests with the same mock instance.
        """
        self.get_dimensions_calls = 0
        self.get_duration_calls = 0
        self.get_fps_calls = 0
        self.get_frame_count_calls = 0
        self.analyze_position_calls = []
        self.extract_frame_calls = []

    def set_position_metric(self, x: int, y: int, metrics: PositionMetrics) -> None:
        """
        Add or update metrics for a specific position.

        Useful for setting up test scenarios after construction.

        Args:
            x: X coordinate
            y: Y coordinate
            metrics: PositionMetrics to return for this position
        """
        self.position_metrics[(x, y)] = metrics

    def was_position_analyzed(self, x: int, y: int) -> bool:
        """
        Check if a specific position was analyzed.

        Args:
            x: X coordinate
            y: Y coordinate

        Returns:
            True if analyze_position was called for this position
        """
        return (x, y) in self.analyze_position_calls

    def get_analysis_count(self) -> int:
        """
        Get total number of positions analyzed.

        Returns:
            Count of analyze_position calls
        """
        return len(self.analyze_position_calls)
