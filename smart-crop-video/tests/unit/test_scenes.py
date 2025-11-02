"""
Unit tests for smart_crop.analysis.scenes module.

Tests scene detection, segmentation, and scene manipulation functions
using pure functions (no FFmpeg required).
"""
import pytest
from smart_crop.analysis.scenes import (
    Scene,
    parse_scene_timestamps,
    create_scenes_from_timestamps,
    create_time_based_segments,
    filter_short_scenes,
    merge_short_scenes,
    get_scene_at_time
)


class TestScene:
    """Tests for Scene dataclass"""

    def test_create_scene(self):
        """Test creating a Scene object"""
        scene = Scene(
            start_time=5.0,
            end_time=10.0,
            start_frame=150,
            end_frame=300,
            metric_value=25.5
        )

        assert scene.start_time == 5.0
        assert scene.end_time == 10.0
        assert scene.start_frame == 150
        assert scene.end_frame == 300
        assert scene.metric_value == 25.5

    def test_scene_duration_property(self):
        """Test duration property calculation"""
        scene = Scene(5.0, 10.0, 150, 300)
        assert scene.duration == 5.0

    def test_scene_frame_count_property(self):
        """Test frame_count property calculation"""
        scene = Scene(5.0, 10.0, 150, 300)
        assert scene.frame_count == 150

    def test_scene_default_values(self):
        """Test Scene with default values"""
        scene = Scene(0.0, 5.0, 0, 150)
        assert scene.metric_value == 0.0
        assert scene.first_frame_path == ""
        assert scene.last_frame_path == ""

    def test_scene_with_frame_paths(self):
        """Test Scene with frame path values"""
        scene = Scene(
            0.0, 5.0, 0, 150,
            first_frame_path="first.jpg",
            last_frame_path="last.jpg"
        )
        assert scene.first_frame_path == "first.jpg"
        assert scene.last_frame_path == "last.jpg"


class TestParseSceneTimestamps:
    """Tests for parse_scene_timestamps function"""

    def test_parse_empty_stderr(self):
        """Test parsing empty stderr"""
        timestamps = parse_scene_timestamps("")
        assert timestamps == []

    def test_parse_single_timestamp(self):
        """Test parsing single timestamp"""
        stderr = "[Parsed_showinfo_1] n:150 pts_time:5.0"
        timestamps = parse_scene_timestamps(stderr)

        assert len(timestamps) == 1
        assert timestamps[0] == (5.0, 150)

    def test_parse_multiple_timestamps(self):
        """Test parsing multiple timestamps"""
        stderr = """
[Parsed_showinfo_1] n:0 pts_time:0.0
[Parsed_showinfo_1] n:150 pts_time:5.0
[Parsed_showinfo_1] n:300 pts_time:10.0
"""
        timestamps = parse_scene_timestamps(stderr)

        assert len(timestamps) == 3
        assert timestamps[0] == (0.0, 0)
        assert timestamps[1] == (5.0, 150)
        assert timestamps[2] == (10.0, 300)

    def test_parse_with_extra_output(self):
        """Test parsing with extra FFmpeg output"""
        stderr = """
ffmpeg version 4.4.2 Copyright (c) 2000-2021
[Parsed_showinfo_1] n:150 pts_time:5.0
Some other output line
[Parsed_showinfo_1] n:300 pts_time:10.0
Duration: 00:00:15.00
"""
        timestamps = parse_scene_timestamps(stderr)

        assert len(timestamps) == 2
        assert timestamps[0] == (5.0, 150)
        assert timestamps[1] == (10.0, 300)

    def test_parse_fractional_timestamps(self):
        """Test parsing fractional timestamps"""
        stderr = "[Parsed_showinfo_1] n:150 pts_time:5.25"
        timestamps = parse_scene_timestamps(stderr)

        assert timestamps[0] == (5.25, 150)

    def test_parse_missing_frame_number(self):
        """Test parsing line with missing frame number"""
        stderr = "pts_time:5.0"  # Missing n:
        timestamps = parse_scene_timestamps(stderr)

        assert timestamps == []

    def test_parse_missing_timestamp(self):
        """Test parsing line with missing timestamp"""
        stderr = "n:150"  # Missing pts_time:
        timestamps = parse_scene_timestamps(stderr)

        assert timestamps == []


class TestCreateScenesFromTimestamps:
    """Tests for create_scenes_from_timestamps function"""

    def test_create_scenes_basic(self):
        """Test basic scene creation"""
        timestamps = [(5.0, 150), (10.0, 300)]
        scenes = create_scenes_from_timestamps(timestamps, 15.0, 450)

        assert len(scenes) == 3
        # First scene: 0.0 to 5.0
        assert scenes[0].start_time == 0.0
        assert scenes[0].end_time == 5.0
        # Second scene: 5.0 to 10.0
        assert scenes[1].start_time == 5.0
        assert scenes[1].end_time == 10.0
        # Third scene: 10.0 to 15.0
        assert scenes[2].start_time == 10.0
        assert scenes[2].end_time == 15.0

    def test_create_scenes_empty_timestamps(self):
        """Test scene creation with no scene changes"""
        scenes = create_scenes_from_timestamps([], 10.0, 300)

        # Should create single scene spanning entire video
        assert len(scenes) == 1
        assert scenes[0].start_time == 0.0
        assert scenes[0].end_time == 10.0
        assert scenes[0].start_frame == 0
        assert scenes[0].end_frame == 300

    def test_create_scenes_with_start_and_end(self):
        """Test when timestamps already include start and end"""
        timestamps = [(0.0, 0), (5.0, 150), (10.0, 300)]
        scenes = create_scenes_from_timestamps(timestamps, 10.0, 300)

        # Should not duplicate start/end
        assert len(scenes) == 2
        assert scenes[0].start_time == 0.0
        assert scenes[0].end_time == 5.0
        assert scenes[1].start_time == 5.0
        assert scenes[1].end_time == 10.0

    def test_create_scenes_frame_numbers(self):
        """Test that frame numbers are correctly assigned"""
        timestamps = [(5.0, 150)]
        scenes = create_scenes_from_timestamps(timestamps, 10.0, 300)

        assert scenes[0].start_frame == 0
        assert scenes[0].end_frame == 150
        assert scenes[1].start_frame == 150
        assert scenes[1].end_frame == 300

    def test_create_scenes_invalid_duration(self):
        """Test error with invalid duration"""
        with pytest.raises(ValueError, match="video_duration must be > 0"):
            create_scenes_from_timestamps([], 0.0, 300)

        with pytest.raises(ValueError, match="video_duration must be > 0"):
            create_scenes_from_timestamps([], -5.0, 300)

    def test_create_scenes_invalid_frames(self):
        """Test error with invalid total_frames"""
        with pytest.raises(ValueError, match="total_frames must be > 0"):
            create_scenes_from_timestamps([], 10.0, 0)

        with pytest.raises(ValueError, match="total_frames must be > 0"):
            create_scenes_from_timestamps([], 10.0, -100)

    def test_create_scenes_removes_duplicates(self):
        """Test that duplicate timestamps are removed"""
        timestamps = [(5.0, 150), (5.0, 150), (10.0, 300)]
        scenes = create_scenes_from_timestamps(timestamps, 15.0, 450)

        # Should not create duplicate scenes
        assert len(scenes) == 3


class TestCreateTimeBasedSegments:
    """Tests for create_time_based_segments function"""

    def test_create_segments_exact_division(self):
        """Test when duration divides evenly into segments"""
        segments = create_time_based_segments(
            video_duration=15.0,
            fps=30.0,
            segment_duration=5.0
        )

        assert len(segments) == 3
        assert segments[0].duration == 5.0
        assert segments[1].duration == 5.0
        assert segments[2].duration == 5.0

    def test_create_segments_partial_last(self):
        """Test when last segment is partial"""
        segments = create_time_based_segments(
            video_duration=12.5,
            fps=30.0,
            segment_duration=5.0
        )

        assert len(segments) == 3
        assert segments[0].duration == 5.0
        assert segments[1].duration == 5.0
        assert segments[2].duration == 2.5  # Partial

    def test_create_segments_shorter_than_one(self):
        """Test video shorter than one segment"""
        segments = create_time_based_segments(
            video_duration=3.0,
            fps=30.0,
            segment_duration=5.0
        )

        assert len(segments) == 1
        assert segments[0].duration == 3.0

    def test_create_segments_frame_calculation(self):
        """Test frame number calculation"""
        segments = create_time_based_segments(
            video_duration=10.0,
            fps=30.0,
            segment_duration=5.0
        )

        assert segments[0].start_frame == 0
        assert segments[0].end_frame == 150  # 5.0 * 30
        assert segments[1].start_frame == 150
        assert segments[1].end_frame == 300  # 10.0 * 30

    def test_create_segments_contiguous(self):
        """Test that segments are contiguous"""
        segments = create_time_based_segments(15.0, 30.0, 5.0)

        for i in range(len(segments) - 1):
            assert segments[i].end_time == segments[i+1].start_time
            assert segments[i].end_frame == segments[i+1].start_frame

    def test_create_segments_invalid_duration(self):
        """Test error with invalid video duration"""
        with pytest.raises(ValueError, match="video_duration must be > 0"):
            create_time_based_segments(0.0, 30.0, 5.0)

    def test_create_segments_invalid_fps(self):
        """Test error with invalid fps"""
        with pytest.raises(ValueError, match="fps must be > 0"):
            create_time_based_segments(10.0, 0.0, 5.0)

    def test_create_segments_invalid_segment_duration(self):
        """Test error with invalid segment duration"""
        with pytest.raises(ValueError, match="segment_duration must be > 0"):
            create_time_based_segments(10.0, 30.0, 0.0)


class TestFilterShortScenes:
    """Tests for filter_short_scenes function"""

    def test_filter_empty_list(self):
        """Test filtering empty list"""
        filtered = filter_short_scenes([])
        assert filtered == []

    def test_filter_all_long_scenes(self):
        """Test when all scenes are long enough"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300),
            Scene(10.0, 15.0, 300, 450)
        ]
        filtered = filter_short_scenes(scenes, min_duration=0.5)

        assert len(filtered) == 3

    def test_filter_removes_short_scenes(self):
        """Test that short scenes are removed"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),      # Long
            Scene(5.0, 5.2, 150, 156),    # Short (0.2s)
            Scene(5.2, 10.0, 156, 300)    # Long
        ]
        filtered = filter_short_scenes(scenes, min_duration=0.5)

        assert len(filtered) == 2
        assert filtered[0].start_time == 0.0
        assert filtered[1].start_time == 5.2

    def test_filter_all_short_scenes(self):
        """Test when all scenes are short"""
        scenes = [
            Scene(0.0, 0.1, 0, 3),
            Scene(0.1, 0.2, 3, 6),
            Scene(0.2, 0.3, 6, 9)
        ]
        filtered = filter_short_scenes(scenes, min_duration=0.5)

        assert filtered == []

    def test_filter_custom_min_duration(self):
        """Test with custom min_duration"""
        scenes = [
            Scene(0.0, 2.0, 0, 60),   # 2.0s
            Scene(2.0, 3.0, 60, 90),  # 1.0s
            Scene(3.0, 5.0, 90, 150)  # 2.0s
        ]
        filtered = filter_short_scenes(scenes, min_duration=1.5)

        assert len(filtered) == 2
        assert all(s.duration >= 1.5 for s in filtered)

    def test_filter_invalid_min_duration(self):
        """Test error with negative min_duration"""
        scenes = [Scene(0.0, 5.0, 0, 150)]

        with pytest.raises(ValueError, match="min_duration must be >= 0"):
            filter_short_scenes(scenes, min_duration=-1.0)


class TestMergeShortScenes:
    """Tests for merge_short_scenes function"""

    def test_merge_empty_list(self):
        """Test merging empty list"""
        merged = merge_short_scenes([])
        assert merged == []

    def test_merge_single_scene(self):
        """Test merging single scene"""
        scenes = [Scene(0.0, 5.0, 0, 150)]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        assert len(merged) == 1
        assert merged[0].start_time == 0.0

    def test_merge_all_long_scenes(self):
        """Test when all scenes are long enough"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        assert len(merged) == 2

    def test_merge_short_with_next(self):
        """Test merging short scene with next scene"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 5.2, 150, 156),  # Short
            Scene(5.2, 10.0, 156, 300)
        ]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        assert len(merged) == 2
        assert merged[1].start_time == 5.0  # Merged with short
        assert merged[1].end_time == 10.0

    def test_merge_last_short_with_previous(self):
        """Test merging last short scene with previous"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300),
            Scene(10.0, 10.2, 300, 306)  # Short, last
        ]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        assert len(merged) == 2
        assert merged[1].start_time == 5.0
        assert merged[1].end_time == 10.2  # Includes short scene

    def test_merge_consecutive_short_scenes(self):
        """Test merging consecutive short scenes"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 5.2, 150, 156),  # Short
            Scene(5.2, 5.4, 156, 162),  # Short
            Scene(5.4, 10.0, 162, 300)
        ]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        # First short merges with second short
        # Result merges with third long scene
        assert len(merged) <= 3

    def test_merge_preserves_metric_value(self):
        """Test that highest metric value is preserved"""
        scenes = [
            Scene(0.0, 5.0, 0, 150, metric_value=10.0),
            Scene(5.0, 5.2, 150, 156, metric_value=25.0),  # Short, higher metric
            Scene(5.2, 10.0, 156, 300, metric_value=15.0)
        ]
        merged = merge_short_scenes(scenes, min_duration=0.5)

        # Merged scene should have highest metric
        assert merged[1].metric_value == 25.0

    def test_merge_invalid_min_duration(self):
        """Test error with negative min_duration"""
        scenes = [Scene(0.0, 5.0, 0, 150)]

        with pytest.raises(ValueError, match="min_duration must be >= 0"):
            merge_short_scenes(scenes, min_duration=-1.0)


class TestGetSceneAtTime:
    """Tests for get_scene_at_time function"""

    def test_get_scene_empty_list(self):
        """Test with empty scene list"""
        scene = get_scene_at_time([], 5.0)
        assert scene is None

    def test_get_scene_at_start(self):
        """Test getting scene at start time"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]
        scene = get_scene_at_time(scenes, 0.0)

        assert scene is not None
        assert scene.start_time == 0.0

    def test_get_scene_in_middle(self):
        """Test getting scene in middle"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300),
            Scene(10.0, 15.0, 300, 450)
        ]
        scene = get_scene_at_time(scenes, 7.5)

        assert scene is not None
        assert scene.start_time == 5.0
        assert scene.end_time == 10.0

    def test_get_scene_at_boundary(self):
        """Test getting scene at scene boundary"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]
        # At boundary, should return first scene (start <= t < end)
        scene = get_scene_at_time(scenes, 5.0)

        assert scene is not None
        assert scene.start_time == 5.0

    def test_get_scene_at_end(self):
        """Test getting scene at video end"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]
        scene = get_scene_at_time(scenes, 10.0)

        # Should match last scene's end time
        assert scene is not None
        assert scene.end_time == 10.0

    def test_get_scene_before_start(self):
        """Test with timestamp before first scene"""
        scenes = [Scene(5.0, 10.0, 150, 300)]
        scene = get_scene_at_time(scenes, 2.0)

        assert scene is None

    def test_get_scene_after_end(self):
        """Test with timestamp after last scene"""
        scenes = [Scene(0.0, 5.0, 0, 150)]
        scene = get_scene_at_time(scenes, 10.0)

        assert scene is None

    def test_get_scene_fractional_timestamp(self):
        """Test with fractional timestamp"""
        scenes = [
            Scene(0.0, 5.0, 0, 150),
            Scene(5.0, 10.0, 150, 300)
        ]
        scene = get_scene_at_time(scenes, 2.75)

        assert scene is not None
        assert scene.start_time == 0.0


class TestSceneIntegration:
    """Integration tests combining multiple scene functions"""

    def test_create_and_filter_pipeline(self):
        """Test creating time-based segments and filtering short ones"""
        # Create 5s segments for 12s video
        segments = create_time_based_segments(12.0, 30.0, 5.0)

        # Last segment is 2s, filter it out
        filtered = filter_short_scenes(segments, min_duration=3.0)

        assert len(segments) == 3  # 0-5, 5-10, 10-12
        assert len(filtered) == 2  # 0-5, 5-10 (12-15 filtered)

    def test_create_and_merge_pipeline(self):
        """Test creating segments and merging short ones"""
        segments = create_time_based_segments(12.0, 30.0, 5.0)

        # Merge last short segment with previous
        merged = merge_short_scenes(segments, min_duration=3.0)

        assert len(merged) == 2
        assert merged[1].end_time == 12.0  # Includes merged segment

    def test_parse_create_filter_pipeline(self):
        """Test full pipeline from parsing to filtering"""
        # Simulate FFmpeg scene detection output
        stderr = """
[Parsed_showinfo_1] n:150 pts_time:5.0
[Parsed_showinfo_1] n:156 pts_time:5.2
[Parsed_showinfo_1] n:300 pts_time:10.0
"""
        # Parse timestamps
        timestamps = parse_scene_timestamps(stderr)

        # Create scenes
        scenes = create_scenes_from_timestamps(timestamps, 15.0, 450)

        # Filter short scenes (5.0-5.2 is only 0.2s)
        filtered = filter_short_scenes(scenes, min_duration=0.5)

        # Should remove the 0.2s scene
        assert len(scenes) == 4  # 0-5, 5-5.2, 5.2-10, 10-15
        assert len(filtered) == 3  # 0-5, 5.2-10, 10-15
