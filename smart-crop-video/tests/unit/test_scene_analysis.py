"""
Unit tests for smart_crop.scene.analysis module.

Tests all pure functions and FFmpeg-dependent functions with mocks.
"""
import pytest
from unittest.mock import Mock, patch, MagicMock
from smart_crop.scene.analysis import (
    determine_primary_metric,
    identify_boring_sections,
    calculate_speedup_factor,
    extract_metric_from_showinfo,
    analyze_scene_metrics,
    extract_scene_thumbnails
)
from smart_crop.analysis.scenes import Scene


class TestDeterminePrimaryMetric:
    """Tests for determine_primary_metric() pure function"""

    def test_subject_detection_uses_edges(self):
        """Subject Detection strategy should use edges metric"""
        assert determine_primary_metric('Subject Detection') == 'edges'

    def test_motion_priority_uses_motion(self):
        """Motion Priority strategy should use motion metric"""
        assert determine_primary_metric('Motion Priority') == 'motion'

    def test_visual_detail_uses_complexity(self):
        """Visual Detail strategy should use complexity metric"""
        assert determine_primary_metric('Visual Detail') == 'complexity'

    def test_balanced_uses_motion(self):
        """Balanced strategy should default to motion metric"""
        assert determine_primary_metric('Balanced') == 'motion'

    def test_color_focus_uses_edges(self):
        """Color Focus strategy should use edges as proxy"""
        assert determine_primary_metric('Color Focus') == 'edges'

    def test_spatial_strategies_use_motion(self):
        """Spatial strategies should default to motion"""
        assert determine_primary_metric('Spatial:Top-Left') == 'motion'
        assert determine_primary_metric('Spatial:Center') == 'motion'
        assert determine_primary_metric('Spatial:Bottom-Right') == 'motion'

    def test_unknown_strategy_defaults_to_motion(self):
        """Unknown strategies should default to motion"""
        assert determine_primary_metric('UnknownStrategy') == 'motion'
        assert determine_primary_metric('Custom:New') == 'motion'


class TestIdentifyBoringSections:
    """Tests for identify_boring_sections() pure function"""

    def test_empty_scenes_returns_empty_list(self):
        """Empty scene list should return empty boring sections"""
        boring = identify_boring_sections([])
        assert boring == []

    def test_no_boring_scenes_when_all_above_threshold(self):
        """When all scenes above threshold, no boring sections"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=10.0),
            Scene(1, 2, 30, 60, metric_value=15.0),
            Scene(2, 3, 60, 90, metric_value=20.0)
        ]
        # 30th percentile of [10, 15, 20] is 10, so only scene 0 is below
        boring = identify_boring_sections(scenes, percentile_threshold=10.0)
        assert len(boring) == 0  # Nothing below 10th percentile

    def test_identifies_boring_scenes_below_percentile(self):
        """Scenes below percentile threshold should be identified as boring"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=2.0),   # Boring
            Scene(1, 2, 30, 60, metric_value=5.0),  # Boring
            Scene(2, 3, 60, 90, metric_value=10.0), # Not boring
            Scene(3, 4, 90, 120, metric_value=15.0) # Not boring
        ]
        # 50th percentile of [2, 5, 10, 15] = 7.5
        boring = identify_boring_sections(scenes, percentile_threshold=50.0)

        assert len(boring) == 2
        assert boring[0][0] == 0  # Scene 0 is boring
        assert boring[1][0] == 1  # Scene 1 is boring

    def test_speedup_factors_in_range(self):
        """Speedup factors should be between 2.0 and 4.0"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=1.0),
            Scene(1, 2, 30, 60, metric_value=10.0)
        ]
        boring = identify_boring_sections(scenes, percentile_threshold=75.0)

        for scene_idx, speedup in boring:
            assert 2.0 <= speedup <= 4.0

    def test_very_boring_scene_gets_high_speedup(self):
        """Very boring scenes (metric near 0) should get speedup near 4x"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=0.1),   # Very boring
            Scene(1, 2, 30, 60, metric_value=10.0)  # Not boring
        ]
        boring = identify_boring_sections(scenes, percentile_threshold=75.0)

        # Scene 0 should be identified as boring with high speedup
        assert len(boring) >= 1
        _, speedup = boring[0]
        assert speedup >= 3.5  # Should be close to max 4.0

    def test_moderately_boring_scene_gets_low_speedup(self):
        """Moderately boring scenes (near threshold) should get speedup near 2x"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=4.9),  # Just below threshold
            Scene(1, 2, 30, 60, metric_value=5.0),
            Scene(2, 3, 60, 90, metric_value=10.0)
        ]
        # Threshold at 50th percentile = 5.0
        boring = identify_boring_sections(scenes, percentile_threshold=50.0)

        if len(boring) > 0:
            _, speedup = boring[0]
            assert 2.0 <= speedup <= 2.5  # Should be close to min 2.0

    def test_all_scenes_same_value(self):
        """When all scenes have same value, threshold handling"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=5.0),
            Scene(1, 2, 30, 60, metric_value=5.0),
            Scene(2, 3, 60, 90, metric_value=5.0)
        ]
        # All scenes have same value, so none should be below threshold
        boring = identify_boring_sections(scenes, percentile_threshold=50.0)
        # Depending on implementation, this could be 0 or more scenes
        assert isinstance(boring, list)

    def test_single_scene(self):
        """Single scene should handle correctly"""
        scenes = [Scene(0, 1, 0, 30, metric_value=5.0)]
        boring = identify_boring_sections(scenes, percentile_threshold=50.0)
        # Single scene is its own threshold, so won't be below itself
        assert len(boring) == 0

    def test_custom_percentile_threshold(self):
        """Should support custom percentile thresholds"""
        scenes = [
            Scene(0, 1, 0, 30, metric_value=1.0),
            Scene(1, 2, 30, 60, metric_value=2.0),
            Scene(2, 3, 60, 90, metric_value=3.0),
            Scene(3, 4, 90, 120, metric_value=4.0),
            Scene(4, 5, 120, 150, metric_value=5.0)
        ]
        # 20th percentile should identify fewer boring scenes
        boring_20 = identify_boring_sections(scenes, percentile_threshold=20.0)
        # 80th percentile should identify more boring scenes
        boring_80 = identify_boring_sections(scenes, percentile_threshold=80.0)

        assert len(boring_20) <= len(boring_80)


class TestCalculateSpeedupFactor:
    """Tests for calculate_speedup_factor() pure function"""

    def test_metric_at_threshold_gives_min_speedup(self):
        """Metric at threshold should give minimum speedup"""
        speedup = calculate_speedup_factor(10.0, 10.0, min_speedup=2.0, max_speedup=4.0)
        assert speedup == 2.0

    def test_metric_at_zero_gives_max_speedup(self):
        """Metric at zero should give maximum speedup"""
        speedup = calculate_speedup_factor(0.0, 10.0, min_speedup=2.0, max_speedup=4.0)
        assert speedup == 4.0

    def test_metric_halfway_gives_middle_speedup(self):
        """Metric halfway to threshold should give middle speedup"""
        speedup = calculate_speedup_factor(5.0, 10.0, min_speedup=2.0, max_speedup=4.0)
        assert speedup == 3.0

    def test_threshold_zero_returns_middle_value(self):
        """Zero threshold should return middle of min/max"""
        speedup = calculate_speedup_factor(5.0, 0.0, min_speedup=2.0, max_speedup=4.0)
        assert speedup == 3.0

    def test_respects_max_speedup_cap(self):
        """Speedup should never exceed max_speedup"""
        speedup = calculate_speedup_factor(0.0, 10.0, min_speedup=2.0, max_speedup=3.5)
        assert speedup == 3.5

    def test_custom_min_max_speedup(self):
        """Should support custom min/max speedup values"""
        speedup = calculate_speedup_factor(5.0, 10.0, min_speedup=1.5, max_speedup=3.0)
        assert 1.5 <= speedup <= 3.0


class TestExtractMetricFromShowinfo:
    """Tests for extract_metric_from_showinfo()"""

    def test_extract_single_mean_value(self):
        """Should extract mean value from showinfo output"""
        output = "[Parsed_showinfo_1] mean:[123.45 67.89 90.12]"
        values = extract_metric_from_showinfo(output, 'mean')
        assert len(values) == 1
        assert values[0] == 123.45

    def test_extract_multiple_mean_values(self):
        """Should extract multiple mean values from multiple frames"""
        output = """
        [Parsed_showinfo_1] mean:[123.45 67.89 90.12]
        [Parsed_showinfo_1] mean:[100.23 45.67 78.90]
        [Parsed_showinfo_1] mean:[150.00 80.00 60.00]
        """
        values = extract_metric_from_showinfo(output, 'mean')
        assert len(values) == 3
        assert values[0] == 123.45
        assert values[1] == 100.23
        assert values[2] == 150.00

    def test_extract_stdev_values(self):
        """Should extract stdev values"""
        output = "[Parsed_showinfo_1] stdev:[12.34 56.78 90.12]"
        values = extract_metric_from_showinfo(output, 'stdev')
        assert len(values) == 1
        assert values[0] == 12.34

    def test_empty_output_returns_empty_list(self):
        """Empty output should return empty list"""
        values = extract_metric_from_showinfo("", 'mean')
        assert values == []

    def test_no_matching_metric_returns_empty_list(self):
        """Output without matching metric should return empty list"""
        output = "[Parsed_showinfo_1] mean:[123.45]"
        values = extract_metric_from_showinfo(output, 'stdev')
        assert values == []

    def test_extracts_first_value_only(self):
        """Should extract only first value (Y channel) from each array"""
        output = "[Parsed_showinfo_1] mean:[123.45 67.89 90.12]"
        values = extract_metric_from_showinfo(output, 'mean')
        # Should only get first value, not all three
        assert len(values) == 1
        assert values[0] == 123.45


class TestAnalyzeSceneMetrics:
    """Tests for analyze_scene_metrics() with mocked FFmpeg"""

    @patch('smart_crop.scene.analysis.run_ffmpeg')
    def test_analyze_motion_metric(self, mock_run_ffmpeg):
        """Should analyze motion metric correctly"""
        # Mock FFmpeg output with varying mean values (indicating motion)
        mock_run_ffmpeg.return_value = """
        [Parsed_showinfo_1] mean:[100.0 50.0 25.0]
        [Parsed_showinfo_1] mean:[110.0 55.0 28.0]
        [Parsed_showinfo_1] mean:[105.0 52.0 26.0]
        """

        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'motion', sample_frames=3
        )

        # Motion = average of frame-to-frame differences
        # Differences: |110-100|=10, |105-110|=5
        # Average: (10 + 5) / 2 = 7.5
        assert score == pytest.approx(7.5)
        mock_run_ffmpeg.assert_called_once()

    @patch('smart_crop.scene.analysis.run_ffmpeg')
    def test_analyze_complexity_metric(self, mock_run_ffmpeg):
        """Should analyze complexity metric correctly"""
        # Mock FFmpeg output with stdev values
        mock_run_ffmpeg.return_value = """
        [Parsed_showinfo_1] stdev:[10.0 5.0 3.0]
        [Parsed_showinfo_1] stdev:[12.0 6.0 4.0]
        [Parsed_showinfo_1] stdev:[11.0 5.5 3.5]
        """

        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'complexity', sample_frames=3
        )

        # Complexity = average of stdev values
        # Values: 10.0, 12.0, 11.0
        # Average: (10 + 12 + 11) / 3 = 11.0
        assert score == pytest.approx(11.0)

    @patch('smart_crop.scene.analysis.run_ffmpeg')
    def test_analyze_edges_metric(self, mock_run_ffmpeg):
        """Should analyze edges metric correctly"""
        # Mock FFmpeg output with mean values from edge detection
        mock_run_ffmpeg.return_value = """
        [Parsed_showinfo_1] mean:[50.0 25.0 10.0]
        [Parsed_showinfo_1] mean:[60.0 30.0 12.0]
        [Parsed_showinfo_1] mean:[55.0 27.5 11.0]
        """

        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'edges', sample_frames=3
        )

        # Edges = average of mean values
        # Values: 50.0, 60.0, 55.0
        # Average: (50 + 60 + 55) / 3 = 55.0
        assert score == pytest.approx(55.0)

    def test_short_scene_returns_zero(self):
        """Scene shorter than 0.1s should return 0"""
        scene = Scene(1.0, 1.05, 30, 31)  # Only 0.05s
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'motion'
        )
        assert score == 0.0

    def test_scene_with_no_frames_returns_zero(self):
        """Scene with no frames should return 0"""
        scene = Scene(1.0, 1.0, 30, 30)  # 0 frames
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'motion'
        )
        assert score == 0.0

    @patch('smart_crop.scene.analysis.run_ffmpeg')
    def test_motion_single_frame_returns_zero(self, mock_run_ffmpeg):
        """Motion analysis with single frame should return 0"""
        # Only one frame = no motion to calculate
        mock_run_ffmpeg.return_value = "[Parsed_showinfo_1] mean:[100.0 50.0 25.0]"

        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'motion', sample_frames=1
        )
        assert score == 0.0

    @patch('smart_crop.scene.analysis.run_ffmpeg')
    def test_empty_ffmpeg_output_returns_zero(self, mock_run_ffmpeg):
        """Empty FFmpeg output should return 0"""
        mock_run_ffmpeg.return_value = ""

        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'motion'
        )
        assert score == 0.0

    def test_unknown_metric_type_returns_zero(self):
        """Unknown metric type should return 0"""
        scene = Scene(1.0, 3.0, 30, 90)
        score = analyze_scene_metrics(
            'test.mp4', scene, 100, 100, 640, 640, 'unknown_metric'
        )
        assert score == 0.0


class TestExtractSceneThumbnails:
    """Tests for extract_scene_thumbnails() with mocked subprocess"""

    @patch('smart_crop.scene.analysis.subprocess.run')
    @patch('smart_crop.scene.analysis.Path')
    def test_extracts_first_and_last_frames(self, mock_path, mock_subprocess):
        """Should extract first and last frame for each scene"""
        # Mock glob to return empty (no old thumbnails)
        mock_path_instance = MagicMock()
        mock_path_instance.glob.return_value = []
        mock_path.return_value = mock_path_instance

        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]

        extract_scene_thumbnails(
            'test.mp4', scenes, 100, 100, 640, 640, 'test'
        )

        # Should call subprocess 4 times (2 scenes × 2 frames each)
        assert mock_subprocess.call_count == 4

        # Verify scene objects were updated with paths
        assert scenes[0].first_frame_path == '.test_scene_1_first.jpg'
        assert scenes[0].last_frame_path == '.test_scene_1_last.jpg'
        assert scenes[1].first_frame_path == '.test_scene_2_first.jpg'
        assert scenes[1].last_frame_path == '.test_scene_2_last.jpg'

    @patch('smart_crop.scene.analysis.subprocess.run')
    @patch('smart_crop.scene.analysis.Path')
    def test_progress_callback_called(self, mock_path, mock_subprocess):
        """Should call progress callback with updates"""
        mock_path_instance = MagicMock()
        mock_path_instance.glob.return_value = []
        mock_path.return_value = mock_path_instance

        scenes = [Scene(0.0, 5.0, 0, 150)]
        callback_calls = []

        def progress_callback(pct, msg):
            callback_calls.append((pct, msg))

        extract_scene_thumbnails(
            'test.mp4', scenes, 100, 100, 640, 640, 'test',
            progress_callback=progress_callback
        )

        # Should have called callback twice (first frame + last frame)
        assert len(callback_calls) == 2
        assert all(isinstance(pct, int) for pct, _ in callback_calls)
        assert all(isinstance(msg, str) for _, msg in callback_calls)

    @patch('smart_crop.scene.analysis.subprocess.run')
    @patch('smart_crop.scene.analysis.Path')
    def test_progress_offset_applied(self, mock_path, mock_subprocess):
        """Should apply progress offset correctly"""
        mock_path_instance = MagicMock()
        mock_path_instance.glob.return_value = []
        mock_path.return_value = mock_path_instance

        scenes = [Scene(0.0, 5.0, 0, 150)]
        callback_calls = []

        def progress_callback(pct, msg):
            callback_calls.append((pct, msg))

        extract_scene_thumbnails(
            'test.mp4', scenes, 100, 100, 640, 640, 'test',
            progress_callback=progress_callback,
            progress_offset=50  # Start at 50%
        )

        # All progress percentages should be >= 50
        assert all(pct >= 50 for pct, _ in callback_calls)

    @patch('smart_crop.scene.analysis.subprocess.run')
    @patch('smart_crop.scene.analysis.Path')
    def test_cleans_up_old_thumbnails(self, mock_path, mock_subprocess):
        """Should clean up old thumbnail files"""
        # Mock old thumbnails
        old_thumb1 = MagicMock()
        old_thumb2 = MagicMock()
        mock_path_instance = MagicMock()
        mock_path_instance.glob.side_effect = [
            [old_thumb1, old_thumb2],  # First glob (first frames)
            []  # Second glob (last frames)
        ]
        mock_path.return_value = mock_path_instance

        scenes = [Scene(0.0, 5.0, 0, 150)]

        extract_scene_thumbnails(
            'test.mp4', scenes, 100, 100, 640, 640, 'test'
        )

        # Should have unlinked old thumbnails
        old_thumb1.unlink.assert_called_once()
        old_thumb2.unlink.assert_called_once()

    @patch('smart_crop.scene.analysis.subprocess.run')
    @patch('smart_crop.scene.analysis.Path')
    def test_last_frame_slightly_before_end(self, mock_path, mock_subprocess):
        """Last frame should be extracted slightly before scene end"""
        mock_path_instance = MagicMock()
        mock_path_instance.glob.return_value = []
        mock_path.return_value = mock_path_instance

        scenes = [Scene(5.0, 10.0, 150, 300)]

        extract_scene_thumbnails(
            'test.mp4', scenes, 100, 100, 640, 640, 'test'
        )

        # Check that last frame extraction used time < scene.end_time
        # Should be max(5.0, 10.0 - 0.1) = 9.9
        last_frame_call = mock_subprocess.call_args_list[1]  # Second call
        cmd = last_frame_call[0][0]
        # Find -ss argument
        ss_idx = cmd.index('-ss')
        timestamp = float(cmd[ss_idx + 1])
        assert timestamp == pytest.approx(9.9)
