"""
Unit tests for smart_crop.analysis.parallel module.

Tests parallel position analysis functionality using MockAnalyzer
(no real video files or FFmpeg needed).
"""
import pytest
from smart_crop.core.grid import Position, generate_analysis_grid
from smart_crop.analysis.parallel import (
    analyze_positions_parallel_with_analyzer,
    get_optimal_worker_count,
    ProgressTracker
)
from tests.mocks.mock_analyzer import MockAnalyzer


class TestAnalyzePositionsWithAnalyzer:
    """Tests for analyze_positions_parallel_with_analyzer function"""

    def test_analyze_empty_list(self):
        """Test analyzing empty position list returns empty result"""
        mock = MockAnalyzer()
        results = analyze_positions_parallel_with_analyzer(
            mock, [], crop_w=640, crop_h=640
        )
        assert results == []
        assert mock.get_analysis_count() == 0

    def test_analyze_single_position(self):
        """Test analyzing single position"""
        mock = MockAnalyzer()
        positions = [Position(100, 100)]

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640, sample_frames=30
        )

        assert len(results) == 1
        assert results[0].x == 100
        assert results[0].y == 100
        assert mock.get_analysis_count() == 1

    def test_analyze_multiple_positions(self):
        """Test analyzing multiple positions"""
        mock = MockAnalyzer()
        positions = [
            Position(100, 100),
            Position(200, 200),
            Position(300, 300)
        ]

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        assert len(results) == 3
        assert results[0].x == 100
        assert results[1].x == 200
        assert results[2].x == 300
        assert mock.get_analysis_count() == 3

    def test_results_in_same_order_as_input(self):
        """Test that results maintain input order"""
        mock = MockAnalyzer()
        positions = [Position(x, y) for x in range(0, 500, 100) for y in range(0, 500, 100)]

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        for i, pos in enumerate(positions):
            assert results[i].x == pos.x
            assert results[i].y == pos.y

    def test_progress_callback_called(self):
        """Test that progress callback is called for each position"""
        mock = MockAnalyzer()
        positions = [Position(x, x) for x in range(100, 400, 100)]

        progress_calls = []

        def callback(current, total):
            progress_calls.append((current, total))

        analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640,
            progress_callback=callback
        )

        # Should have 3 calls (one per position)
        assert len(progress_calls) == 3
        assert progress_calls[0] == (1, 3)
        assert progress_calls[1] == (2, 3)
        assert progress_calls[2] == (3, 3)

    def test_progress_callback_optional(self):
        """Test that progress callback is optional"""
        mock = MockAnalyzer()
        positions = [Position(100, 100)]

        # Should not raise even without callback
        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640,
            progress_callback=None
        )

        assert len(results) == 1

    def test_uses_preconfigured_metrics(self):
        """Test that analyzer's pre-configured metrics are used"""
        from smart_crop.core.scoring import PositionMetrics

        custom_metrics = {
            (100, 100): PositionMetrics(100, 100, motion=99.0, complexity=88.0, edges=77.0, saturation=66.0)
        }

        mock = MockAnalyzer(position_metrics=custom_metrics)
        positions = [Position(100, 100)]

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        assert results[0].motion == 99.0
        assert results[0].complexity == 88.0

    def test_sample_frames_parameter_passed(self):
        """Test that sample_frames parameter is passed to analyzer"""
        mock = MockAnalyzer()
        positions = [Position(100, 100)]

        # MockAnalyzer doesn't use sample_frames, but it should be accepted
        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640, sample_frames=25
        )

        assert len(results) == 1


class TestGetOptimalWorkerCount:
    """Tests for get_optimal_worker_count function"""

    def test_zero_positions_returns_one(self):
        """Test that zero positions returns 1 worker"""
        assert get_optimal_worker_count(0) == 1

    def test_negative_positions_returns_one(self):
        """Test that negative positions returns 1 worker"""
        assert get_optimal_worker_count(-5) == 1

    def test_single_position_returns_one(self):
        """Test that single position returns 1 worker (no parallelization)"""
        assert get_optimal_worker_count(1) == 1

    def test_two_positions_returns_two(self):
        """Test that 2-3 positions use 2 workers"""
        workers = get_optimal_worker_count(2)
        assert workers >= 1
        assert workers <= 2

    def test_many_positions_uses_multiple_workers(self):
        """Test that many positions use multiple workers"""
        workers = get_optimal_worker_count(100)
        assert workers > 1  # Should definitely use parallelization

    def test_max_workers_limit_respected(self):
        """Test that max_workers limit is respected"""
        workers = get_optimal_worker_count(100, max_workers=2)
        assert workers == 2

    def test_max_workers_one_forces_sequential(self):
        """Test that max_workers=1 forces sequential execution"""
        workers = get_optimal_worker_count(100, max_workers=1)
        assert workers == 1

    def test_worker_count_never_exceeds_position_count(self):
        """Test that we don't create more workers than positions"""
        # Even with many CPUs, shouldn't exceed position count
        workers = get_optimal_worker_count(3, max_workers=100)
        assert workers <= 3

    def test_worker_count_always_positive(self):
        """Test that worker count is always at least 1"""
        for pos_count in [0, 1, 5, 10, 100]:
            workers = get_optimal_worker_count(pos_count)
            assert workers >= 1


class TestProgressTracker:
    """Tests for ProgressTracker class"""

    def test_create_tracker(self):
        """Test creating a ProgressTracker"""
        tracker = ProgressTracker(total=25)
        assert tracker.total == 25
        assert tracker.current == 0
        assert tracker.percent == 0

    def test_update_progress(self):
        """Test updating progress"""
        tracker = ProgressTracker(total=100)

        tracker.update(25)
        assert tracker.current == 25
        assert tracker.percent == 25

        tracker.update(50)
        assert tracker.current == 50
        assert tracker.percent == 50

        tracker.update(100)
        assert tracker.current == 100
        assert tracker.percent == 100

    def test_update_with_new_total(self):
        """Test updating total count"""
        tracker = ProgressTracker(total=50)
        tracker.update(25, total=100)

        assert tracker.total == 100
        assert tracker.current == 25
        assert tracker.percent == 25

    def test_is_complete_false_initially(self):
        """Test that is_complete is False initially"""
        tracker = ProgressTracker(total=10)
        assert not tracker.is_complete()

    def test_is_complete_true_when_done(self):
        """Test that is_complete is True when current >= total"""
        tracker = ProgressTracker(total=10)
        tracker.update(10)
        assert tracker.is_complete()

    def test_is_complete_true_when_exceeded(self):
        """Test that is_complete is True even when exceeded"""
        tracker = ProgressTracker(total=10)
        tracker.update(15)
        assert tracker.is_complete()

    def test_percent_calculation(self):
        """Test percent calculation accuracy"""
        tracker = ProgressTracker(total=100)

        tracker.update(0)
        assert tracker.percent == 0

        tracker.update(25)
        assert tracker.percent == 25

        tracker.update(50)
        assert tracker.percent == 50

        tracker.update(75)
        assert tracker.percent == 75

        tracker.update(100)
        assert tracker.percent == 100

    def test_percent_with_non_round_numbers(self):
        """Test percent with non-round total"""
        tracker = ProgressTracker(total=7)

        tracker.update(3)
        # 3/7 = 0.428... = 42%
        assert tracker.percent == 42

        tracker.update(5)
        # 5/7 = 0.714... = 71%
        assert tracker.percent == 71

    def test_zero_total_doesnt_crash(self):
        """Test that zero total doesn't cause division by zero"""
        tracker = ProgressTracker(total=0)
        tracker.update(5)
        assert tracker.percent == 0  # Should handle gracefully

    def test_string_representation(self):
        """Test string representation"""
        tracker = ProgressTracker(total=10)
        tracker.update(3)

        string_repr = str(tracker)
        assert "3" in string_repr
        assert "10" in string_repr
        assert "30%" in string_repr

    def test_use_in_callback(self):
        """Test using ProgressTracker in a callback"""
        mock = MockAnalyzer()
        positions = [Position(x, x) for x in range(100, 600, 100)]

        tracker = ProgressTracker(total=len(positions))

        def callback(current, total):
            tracker.update(current, total)

        analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640,
            progress_callback=callback
        )

        assert tracker.is_complete()
        assert tracker.current == 5
        assert tracker.percent == 100


class TestIntegrationWithGrid:
    """Integration tests combining grid generation with parallel analysis"""

    def test_analyze_full_grid(self):
        """Test analyzing a complete grid"""
        mock = MockAnalyzer(dimensions=(1920, 1080))

        # Generate 5x5 grid
        positions = generate_analysis_grid(max_x=1280, max_y=440, grid_size=5)

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        # Should have 25 results
        assert len(results) == 25
        assert mock.get_analysis_count() == 25

        # All positions should be analyzed
        for pos in positions:
            assert mock.was_position_analyzed(pos.x, pos.y)

    def test_analyze_small_grid_with_progress(self):
        """Test analyzing small grid with progress tracking"""
        mock = MockAnalyzer()
        positions = generate_analysis_grid(max_x=200, max_y=200, grid_size=3)

        tracker = ProgressTracker(total=len(positions))
        progress_updates = []

        def callback(current, total):
            tracker.update(current, total)
            progress_updates.append(tracker.percent)

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640,
            progress_callback=callback
        )

        # Should have 9 results (3x3)
        assert len(results) == 9

        # Should have 9 progress updates
        assert len(progress_updates) == 9

        # Progress should increase
        assert progress_updates[-1] == 100  # Final update should be 100%
        assert all(progress_updates[i] <= progress_updates[i+1]
                  for i in range(len(progress_updates)-1))

    def test_large_grid_analysis(self):
        """Test analyzing larger grid (performance test)"""
        import time

        mock = MockAnalyzer()
        # Generate 10x10 grid = 100 positions
        positions = generate_analysis_grid(max_x=1000, max_y=1000, grid_size=10)

        start = time.time()
        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )
        duration = time.time() - start

        # 100 positions should still be very fast with mock
        assert len(results) == 100
        assert duration < 1.0  # Should complete in under 1 second

    def test_edge_case_no_movement_needed(self):
        """Test grid when crop fits exactly (no movement)"""
        mock = MockAnalyzer()
        # No movement possible
        positions = generate_analysis_grid(max_x=0, max_y=0)

        results = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        # Should have 1 result at (0, 0)
        assert len(results) == 1
        assert results[0].x == 0
        assert results[0].y == 0


class TestDeterminism:
    """Tests for deterministic behavior"""

    def test_same_positions_give_same_results(self):
        """Test that analyzing same positions gives consistent results"""
        mock = MockAnalyzer()
        positions = [Position(100, 100), Position(200, 200), Position(300, 300)]

        results1 = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        # Reset mock
        mock.reset_call_tracking()

        results2 = analyze_positions_parallel_with_analyzer(
            mock, positions, crop_w=640, crop_h=640
        )

        # Results should be identical
        assert len(results1) == len(results2)
        for r1, r2 in zip(results1, results2):
            assert r1.x == r2.x
            assert r1.y == r2.y
            assert r1.motion == r2.motion
            assert r1.complexity == r2.complexity
            assert r1.edges == r2.edges
            assert r1.saturation == r2.saturation
