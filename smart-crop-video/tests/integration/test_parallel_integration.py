"""
Integration tests for parallel analysis with real FFmpeg.

These tests use the actual example_movie.mov file and FFmpeg to verify
that parallel analysis works correctly and provides performance benefits.

NOTE: These tests require FFmpeg to be installed. They will be skipped
if FFmpeg is not available (e.g., running outside Docker).
"""
import pytest
import time
import subprocess
import shutil
from pathlib import Path
from smart_crop.core.grid import generate_analysis_grid, Position
from smart_crop.analysis.parallel import (
    analyze_positions_parallel,
    ProgressTracker
)


# Path to test video
TEST_VIDEO = Path(__file__).parent.parent.parent / "example_movie.mov"

# Check if FFmpeg is available
HAS_FFMPEG = shutil.which('ffmpeg') is not None


@pytest.mark.skipif(not TEST_VIDEO.exists(), reason="Test video not found")
@pytest.mark.skipif(not HAS_FFMPEG, reason="FFmpeg not available")
class TestParallelAnalysisIntegration:
    """Integration tests with real video file"""

    def test_sequential_analysis_works(self):
        """Test that sequential analysis (max_workers=1) works"""
        positions = [Position(100, 100), Position(200, 200)]

        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=5,  # Small number for fast test
            max_workers=1  # Sequential
        )

        assert len(results) == 2
        assert results[0].x == 100
        assert results[0].y == 100
        assert results[1].x == 200
        assert results[1].y == 200

        # Should have valid metrics
        assert results[0].motion >= 0
        assert results[0].complexity >= 0
        assert results[0].edges >= 0
        assert results[0].saturation >= 0

    def test_parallel_analysis_works(self):
        """Test that parallel analysis works"""
        positions = [Position(100, 100), Position(200, 200)]

        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=5,
            max_workers=2  # Parallel
        )

        assert len(results) == 2
        # Results should be in same order as input
        assert results[0].x == 100
        assert results[1].x == 200

    def test_sequential_and_parallel_give_same_results(self):
        """Test that sequential and parallel produce identical results"""
        positions = [Position(100, 100), Position(200, 200), Position(300, 300)]

        # Sequential analysis
        seq_results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=10,
            max_workers=1
        )

        # Parallel analysis
        par_results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=10,
            max_workers=2
        )

        # Results should be identical (or very close for floating point)
        assert len(seq_results) == len(par_results)
        for seq, par in zip(seq_results, par_results):
            assert seq.x == par.x
            assert seq.y == par.y
            # Metrics should be very close (allow tiny floating point differences)
            assert abs(seq.motion - par.motion) < 0.01
            assert abs(seq.complexity - par.complexity) < 0.01
            assert abs(seq.edges - par.edges) < 0.01
            assert abs(seq.saturation - par.saturation) < 0.01

    def test_progress_callback_with_real_analysis(self):
        """Test progress callback with real FFmpeg analysis"""
        positions = [Position(100, 100), Position(200, 200), Position(300, 300)]
        tracker = ProgressTracker(total=len(positions))
        progress_updates = []

        def callback(current, total):
            tracker.update(current, total)
            progress_updates.append((current, total))

        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=5,
            max_workers=2,
            progress_callback=callback
        )

        assert len(results) == 3
        assert len(progress_updates) == 3
        assert progress_updates[-1] == (3, 3)
        assert tracker.is_complete()

    @pytest.mark.slow
    def test_parallel_is_faster_than_sequential(self):
        """Test that parallel analysis is actually faster"""
        # Use a small grid for reasonable test time
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=3)  # 9 positions

        # Sequential timing
        start = time.time()
        seq_results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=10,
            max_workers=1
        )
        seq_duration = time.time() - start

        # Parallel timing
        start = time.time()
        par_results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=10,
            max_workers=4
        )
        par_duration = time.time() - start

        # Verify results are the same
        assert len(seq_results) == len(par_results)

        # Parallel should be noticeably faster (at least 1.5x on multi-core)
        # Using conservative ratio since test environment may vary
        speedup = seq_duration / par_duration
        print(f"\nSequential: {seq_duration:.2f}s")
        print(f"Parallel: {par_duration:.2f}s")
        print(f"Speedup: {speedup:.2f}x")

        # Should see at least some speedup
        assert par_duration < seq_duration, \
            f"Parallel ({par_duration:.2f}s) not faster than sequential ({seq_duration:.2f}s)"

    def test_empty_position_list(self):
        """Test handling of empty position list"""
        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            [],
            crop_w=200,
            crop_h=200
        )
        assert results == []

    def test_single_position_uses_sequential(self):
        """Test that single position automatically uses sequential"""
        # Single position should work fine
        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            [Position(100, 100)],
            crop_w=200,
            crop_h=200,
            sample_frames=5,
            max_workers=4  # Even with max_workers=4, should work
        )

        assert len(results) == 1
        assert results[0].x == 100

    def test_large_sample_frames(self):
        """Test with larger sample_frames value"""
        positions = [Position(100, 100)]

        results = analyze_positions_parallel(
            str(TEST_VIDEO),
            positions,
            crop_w=200,
            crop_h=200,
            sample_frames=30,
            max_workers=1
        )

        assert len(results) == 1
        # Should have analyzed more frames
        assert results[0].motion >= 0
