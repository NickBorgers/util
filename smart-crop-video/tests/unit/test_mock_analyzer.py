"""
Unit tests for MockAnalyzer.

These tests verify that the MockAnalyzer works correctly and demonstrate
how to use it for testing business logic without FFmpeg.
"""
import pytest
import tempfile
import os
from pathlib import Path
from tests.mocks.mock_analyzer import MockAnalyzer
from smart_crop.core.scoring import PositionMetrics, NormalizationBounds, score_position
from smart_crop.core.grid import generate_analysis_grid


class TestMockAnalyzerBasics:
    """Tests for basic MockAnalyzer functionality"""

    def test_create_with_defaults(self):
        """Test creating MockAnalyzer with default values"""
        mock = MockAnalyzer()

        assert mock.get_dimensions() == (1920, 1080)
        assert mock.get_duration() == 30.0
        assert mock.get_fps() == 30.0
        assert mock.get_frame_count() == 900  # 30 * 30

    def test_create_with_custom_values(self):
        """Test creating MockAnalyzer with custom values"""
        mock = MockAnalyzer(
            dimensions=(3840, 2160),
            duration=60.0,
            fps=24.0
        )

        assert mock.get_dimensions() == (3840, 2160)
        assert mock.get_duration() == 60.0
        assert mock.get_fps() == 24.0
        assert mock.get_frame_count() == 1440  # 60 * 24

    def test_get_video_info(self):
        """Test get_video_info convenience method"""
        mock = MockAnalyzer(
            dimensions=(1920, 1080),
            duration=45.0,
            fps=30.0
        )

        info = mock.get_video_info()

        assert info['width'] == 1920
        assert info['height'] == 1080
        assert info['duration'] == 45.0
        assert info['fps'] == 30.0
        assert info['frame_count'] == 1350


class TestMockAnalyzerCallTracking:
    """Tests for call tracking functionality"""

    def test_tracks_get_dimensions_calls(self):
        """Test that get_dimensions calls are tracked"""
        mock = MockAnalyzer()

        assert mock.get_dimensions_calls == 0
        mock.get_dimensions()
        assert mock.get_dimensions_calls == 1
        mock.get_dimensions()
        mock.get_dimensions()
        assert mock.get_dimensions_calls == 3

    def test_tracks_analyze_position_calls(self):
        """Test that analyze_position calls are tracked"""
        mock = MockAnalyzer()

        assert len(mock.analyze_position_calls) == 0

        mock.analyze_position(100, 100, 640, 640)
        assert len(mock.analyze_position_calls) == 1
        assert mock.analyze_position_calls[0] == (100, 100)

        mock.analyze_position(200, 200, 640, 640)
        assert len(mock.analyze_position_calls) == 2
        assert mock.analyze_position_calls[1] == (200, 200)

    def test_was_position_analyzed(self):
        """Test was_position_analyzed helper method"""
        mock = MockAnalyzer()

        assert not mock.was_position_analyzed(100, 100)

        mock.analyze_position(100, 100, 640, 640)

        assert mock.was_position_analyzed(100, 100)
        assert not mock.was_position_analyzed(200, 200)

    def test_get_analysis_count(self):
        """Test get_analysis_count helper method"""
        mock = MockAnalyzer()

        assert mock.get_analysis_count() == 0

        mock.analyze_position(100, 100, 640, 640)
        mock.analyze_position(200, 200, 640, 640)
        mock.analyze_position(300, 300, 640, 640)

        assert mock.get_analysis_count() == 3

    def test_reset_call_tracking(self):
        """Test that reset_call_tracking clears all counters"""
        mock = MockAnalyzer()

        mock.get_dimensions()
        mock.get_duration()
        mock.analyze_position(100, 100, 640, 640)

        assert mock.get_dimensions_calls > 0
        assert mock.get_duration_calls > 0
        assert len(mock.analyze_position_calls) > 0

        mock.reset_call_tracking()

        assert mock.get_dimensions_calls == 0
        assert mock.get_duration_calls == 0
        assert len(mock.analyze_position_calls) == 0


class TestMockAnalyzerPositionMetrics:
    """Tests for position metrics generation and configuration"""

    def test_default_metrics_are_deterministic(self):
        """Test that default metrics are consistent for same position"""
        mock = MockAnalyzer()

        metrics1 = mock.analyze_position(100, 100, 640, 640)
        metrics2 = mock.analyze_position(100, 100, 640, 640)

        assert metrics1.x == metrics2.x
        assert metrics1.y == metrics2.y
        assert metrics1.motion == metrics2.motion
        assert metrics1.complexity == metrics2.complexity
        assert metrics1.edges == metrics2.edges
        assert metrics1.saturation == metrics2.saturation

    def test_default_metrics_vary_by_position(self):
        """Test that different positions get different default metrics"""
        mock = MockAnalyzer()

        metrics1 = mock.analyze_position(100, 100, 640, 640)
        metrics2 = mock.analyze_position(200, 200, 640, 640)

        # Should have different scores due to position
        assert metrics1.motion != metrics2.motion
        assert metrics1.complexity != metrics2.complexity

    def test_preconfigured_metrics_override_defaults(self):
        """Test that pre-configured metrics override defaults"""
        custom_metrics = PositionMetrics(
            x=100, y=100,
            motion=99.9,
            complexity=88.8,
            edges=77.7,
            saturation=66.6
        )

        mock = MockAnalyzer(
            position_metrics={(100, 100): custom_metrics}
        )

        result = mock.analyze_position(100, 100, 640, 640)

        assert result.motion == 99.9
        assert result.complexity == 88.8
        assert result.edges == 77.7
        assert result.saturation == 66.6

    def test_set_position_metric_after_creation(self):
        """Test setting metrics after mock creation"""
        mock = MockAnalyzer()

        # Default metrics initially
        result1 = mock.analyze_position(100, 100, 640, 640)
        default_motion = result1.motion

        # Set custom metrics
        custom = PositionMetrics(100, 100, motion=50.0, complexity=40.0, edges=30.0, saturation=20.0)
        mock.set_position_metric(100, 100, custom)

        # Should now return custom metrics
        result2 = mock.analyze_position(100, 100, 640, 640)
        assert result2.motion == 50.0
        assert result2.motion != default_motion


class TestMockAnalyzerFrameExtraction:
    """Tests for frame extraction functionality"""

    def test_extract_frame_creates_file(self):
        """Test that extract_frame creates a file"""
        mock = MockAnalyzer()

        with tempfile.TemporaryDirectory() as tmpdir:
            output_path = os.path.join(tmpdir, "frame.jpg")

            mock.extract_frame(5.0, output_path)

            assert os.path.exists(output_path)
            assert os.path.getsize(output_path) > 0

    def test_extract_frame_creates_jpeg_magic_bytes(self):
        """Test that extracted frame has JPEG magic bytes"""
        mock = MockAnalyzer()

        with tempfile.TemporaryDirectory() as tmpdir:
            output_path = os.path.join(tmpdir, "frame.jpg")

            mock.extract_frame(5.0, output_path)

            with open(output_path, 'rb') as f:
                magic = f.read(4)
                # JPEG magic bytes: FF D8 FF E0
                assert magic == b'\xFF\xD8\xFF\xE0'

    def test_extract_frame_tracks_calls(self):
        """Test that frame extraction calls are tracked"""
        mock = MockAnalyzer()

        with tempfile.TemporaryDirectory() as tmpdir:
            output_path = os.path.join(tmpdir, "frame.jpg")

            assert len(mock.extract_frame_calls) == 0

            mock.extract_frame(5.0, output_path, x=100, y=100, crop_w=640, crop_h=480)

            assert len(mock.extract_frame_calls) == 1
            call = mock.extract_frame_calls[0]
            assert call == (5.0, output_path, 100, 100, 640, 480)


class TestUsingMockForBusinessLogic:
    """Tests demonstrating how to use MockAnalyzer for business logic testing"""

    def test_scoring_with_mock_analyzer(self):
        """Test scoring positions using MockAnalyzer"""
        # Set up mock with specific metrics for testing
        mock = MockAnalyzer(
            position_metrics={
                (100, 100): PositionMetrics(100, 100, motion=10.0, complexity=5.0, edges=8.0, saturation=6.0),
                (200, 200): PositionMetrics(200, 200, motion=5.0, complexity=10.0, edges=6.0, saturation=8.0),
                (300, 300): PositionMetrics(300, 300, motion=2.0, complexity=3.0, edges=9.0, saturation=7.0),
            }
        )

        # Analyze positions
        pos1 = mock.analyze_position(100, 100, 640, 640)
        pos2 = mock.analyze_position(200, 200, 640, 640)
        pos3 = mock.analyze_position(300, 300, 640, 640)

        # Calculate normalization bounds
        positions = [pos1, pos2, pos3]
        bounds = NormalizationBounds.from_positions(positions)

        # Score each position with Motion Priority strategy
        score1 = score_position(pos1, bounds, 'Motion Priority')
        score2 = score_position(pos2, bounds, 'Motion Priority')
        score3 = score_position(pos3, bounds, 'Motion Priority')

        # Position 1 should score highest (has highest motion)
        assert score1 > score2
        assert score1 > score3

    def test_grid_analysis_with_mock(self):
        """Test analyzing a full grid using MockAnalyzer"""
        mock = MockAnalyzer()

        # Generate a 3x3 grid
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=3)

        # Analyze all positions
        metrics = []
        for pos in positions:
            result = mock.analyze_position(pos.x, pos.y, 640, 640)
            metrics.append(result)

        # Should have analyzed 9 positions
        assert len(metrics) == 9
        assert mock.get_analysis_count() == 9

        # All positions should have been analyzed
        for pos in positions:
            assert mock.was_position_analyzed(pos.x, pos.y)

    def test_find_best_position_with_mock(self):
        """Test finding best position using mock data"""
        # Configure mock with specific "interesting" positions
        mock = MockAnalyzer(
            position_metrics={
                (0, 0): PositionMetrics(0, 0, motion=1.0, complexity=1.0, edges=1.0, saturation=1.0),
                (100, 100): PositionMetrics(100, 100, motion=10.0, complexity=8.0, edges=9.0, saturation=7.0),
                (200, 200): PositionMetrics(200, 200, motion=5.0, complexity=5.0, edges=5.0, saturation=5.0),
            }
        )

        # Analyze all positions
        positions_to_analyze = [(0, 0), (100, 100), (200, 200)]
        metrics = [mock.analyze_position(x, y, 640, 640) for x, y in positions_to_analyze]

        # Calculate bounds and score
        bounds = NormalizationBounds.from_positions(metrics)
        scores = [(m, score_position(m, bounds, 'Balanced')) for m in metrics]

        # Find best
        best_metrics, best_score = max(scores, key=lambda x: x[1])

        # Position (100, 100) should win (highest values overall)
        assert best_metrics.x == 100
        assert best_metrics.y == 100

    def test_mock_enables_fast_tests(self):
        """Test that MockAnalyzer is fast (no subprocess calls)"""
        import time

        mock = MockAnalyzer()

        # Analyze 100 positions - should be instant
        start = time.time()
        for i in range(100):
            mock.analyze_position(i, i, 640, 640)
        duration = time.time() - start

        # Should complete in well under 1 second
        # (real FFmpeg would take minutes)
        assert duration < 0.1
        assert mock.get_analysis_count() == 100
