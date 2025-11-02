"""
Scene detection and segmentation for video analysis.

This module provides functionality for dividing videos into scenes or segments
for intelligent acceleration and analysis. Scenes can be detected automatically
using FFmpeg's scene detection, or created as fixed-duration segments.

Scene Detection Workflow:
1. Attempt automatic scene detection using FFmpeg
2. If too few scenes found, fall back to time-based segmentation
3. Extract thumbnails for scene preview
4. Allow user to select scenes for acceleration

Pure functions for testing are separated from FFmpeg-dependent operations.
"""
from typing import List, Tuple, Optional
from dataclasses import dataclass
import re


@dataclass
class Scene:
    """
    Represents a continuous scene or segment in a video.

    A scene has temporal boundaries (start/end time), frame boundaries
    (start/end frame), and optional metric values for ranking.

    Attributes:
        start_time: Scene start timestamp in seconds
        end_time: Scene end timestamp in seconds
        start_frame: Frame number at scene start
        end_frame: Frame number at scene end
        metric_value: Optional metric for ranking scenes (e.g., motion, complexity)
        first_frame_path: Path to extracted first frame thumbnail
        last_frame_path: Path to extracted last frame thumbnail
    """
    start_time: float
    end_time: float
    start_frame: int
    end_frame: int
    metric_value: float = 0.0
    first_frame_path: str = ""
    last_frame_path: str = ""

    @property
    def duration(self) -> float:
        """
        Calculate scene duration in seconds.

        Returns:
            Duration in seconds (end_time - start_time)

        Examples:
            >>> scene = Scene(5.0, 10.0, 150, 300)
            >>> scene.duration
            5.0
        """
        return self.end_time - self.start_time

    @property
    def frame_count(self) -> int:
        """
        Calculate number of frames in scene.

        Returns:
            Number of frames (end_frame - start_frame)

        Examples:
            >>> scene = Scene(5.0, 10.0, 150, 300)
            >>> scene.frame_count
            150
        """
        return self.end_frame - self.start_frame


def parse_scene_timestamps(
    ffmpeg_stderr: str,
    pts_pattern: str = r'pts_time:([0-9.]+)',
    n_pattern: str = r'n:\s*(\d+)'
) -> List[Tuple[float, int]]:
    """
    Parse scene change timestamps from FFmpeg showinfo output.

    FFmpeg's scene detection filter outputs lines like:
    [Parsed_showinfo_1] n:150 pts:5.0 pts_time:5.0 ...

    This function extracts (pts_time, frame_number) tuples.

    Args:
        ffmpeg_stderr: FFmpeg stderr output containing showinfo lines
        pts_pattern: Regex pattern for extracting pts_time (default: r'pts_time:([0-9.]+)')
        n_pattern: Regex pattern for extracting frame number (default: r'n:\s*(\d+)')

    Returns:
        List of (timestamp, frame_number) tuples, sorted by timestamp

    Examples:
        >>> stderr = '''
        ... [Parsed_showinfo_1] n:0 pts_time:0.0
        ... [Parsed_showinfo_1] n:150 pts_time:5.0
        ... [Parsed_showinfo_1] n:300 pts_time:10.0
        ... '''
        >>> timestamps = parse_scene_timestamps(stderr)
        >>> timestamps
        [(0.0, 0), (5.0, 150), (10.0, 300)]

    Notes:
        - Returns empty list if no timestamps found
        - Skips lines that don't match both patterns
        - Results are in the order they appear in the input
    """
    scene_changes = []

    for line in ffmpeg_stderr.split('\n'):
        if 'pts_time' in line:
            pts_match = re.search(pts_pattern, line)
            n_match = re.search(n_pattern, line)

            if pts_match and n_match:
                timestamp = float(pts_match.group(1))
                frame_num = int(n_match.group(1))
                scene_changes.append((timestamp, frame_num))

    return scene_changes


def create_scenes_from_timestamps(
    timestamps: List[Tuple[float, int]],
    video_duration: float,
    total_frames: int
) -> List[Scene]:
    """
    Create Scene objects from timestamp boundaries.

    Given a list of scene change timestamps, creates Scene objects
    representing the segments between each pair of timestamps. Automatically
    adds start (0.0, 0) and end (duration, total_frames) if not present.

    Args:
        timestamps: List of (timestamp, frame_number) tuples marking scene boundaries
        video_duration: Total video duration in seconds
        total_frames: Total number of frames in video

    Returns:
        List of Scene objects, one for each segment between timestamps

    Raises:
        ValueError: If video_duration or total_frames are invalid (<= 0)

    Examples:
        >>> timestamps = [(5.0, 150), (10.0, 300)]  # One scene change at 5s
        >>> scenes = create_scenes_from_timestamps(timestamps, 15.0, 450)
        >>> len(scenes)
        3
        >>> scenes[0].start_time, scenes[0].end_time
        (0.0, 5.0)
        >>> scenes[1].start_time, scenes[1].end_time
        (5.0, 10.0)
        >>> scenes[2].start_time, scenes[2].end_time
        (10.0, 15.0)

    Notes:
        - First scene always starts at (0.0, 0)
        - Last scene always ends at (video_duration, total_frames)
        - Empty timestamps list returns single scene spanning entire video
    """
    if video_duration <= 0:
        raise ValueError(f"video_duration must be > 0, got {video_duration}")

    if total_frames <= 0:
        raise ValueError(f"total_frames must be > 0, got {total_frames}")

    # Ensure we have start and end boundaries
    all_timestamps = [(0.0, 0)] + timestamps + [(video_duration, total_frames)]

    # Remove duplicates while preserving order
    seen = set()
    unique_timestamps = []
    for ts in all_timestamps:
        if ts not in seen:
            seen.add(ts)
            unique_timestamps.append(ts)

    # Sort by timestamp (should already be sorted, but ensure it)
    unique_timestamps.sort(key=lambda x: x[0])

    # Create Scene objects for each segment
    scenes = []
    for i in range(len(unique_timestamps) - 1):
        start_time, start_frame = unique_timestamps[i]
        end_time, end_frame = unique_timestamps[i + 1]

        scene = Scene(
            start_time=start_time,
            end_time=end_time,
            start_frame=start_frame,
            end_frame=end_frame,
            metric_value=0.0
        )
        scenes.append(scene)

    return scenes


def create_time_based_segments(
    video_duration: float,
    fps: float,
    segment_duration: float = 5.0
) -> List[Scene]:
    """
    Create fixed-duration time-based segments.

    When automatic scene detection fails to find enough scenes, this function
    creates fixed-duration segments. This is useful for videos with few cuts
    or gradual transitions.

    Args:
        video_duration: Total video duration in seconds
        fps: Video frame rate (frames per second)
        segment_duration: Duration of each segment in seconds (default: 5.0)

    Returns:
        List of Scene objects representing equally-sized segments
        Last segment may be shorter if video_duration is not evenly divisible

    Raises:
        ValueError: If video_duration, fps, or segment_duration are invalid (<= 0)

    Examples:
        >>> segments = create_time_based_segments(
        ...     video_duration=12.5,
        ...     fps=30.0,
        ...     segment_duration=5.0
        ... )
        >>> len(segments)
        3
        >>> segments[0].duration
        5.0
        >>> segments[1].duration
        5.0
        >>> segments[2].duration
        2.5

    Notes:
        - Segments are non-overlapping and contiguous
        - Last segment ends exactly at video_duration
        - Frame numbers are calculated as int(time * fps)
    """
    if video_duration <= 0:
        raise ValueError(f"video_duration must be > 0, got {video_duration}")

    if fps <= 0:
        raise ValueError(f"fps must be > 0, got {fps}")

    if segment_duration <= 0:
        raise ValueError(f"segment_duration must be > 0, got {segment_duration}")

    scenes = []
    current_time = 0.0
    current_frame = 0

    while current_time < video_duration:
        # Calculate end of this segment
        end_time = min(current_time + segment_duration, video_duration)
        end_frame = int(end_time * fps)

        # Create scene
        scene = Scene(
            start_time=current_time,
            end_time=end_time,
            start_frame=current_frame,
            end_frame=end_frame,
            metric_value=0.0
        )
        scenes.append(scene)

        # Move to next segment
        current_time = end_time
        current_frame = end_frame

    return scenes


def filter_short_scenes(
    scenes: List[Scene],
    min_duration: float = 0.5
) -> List[Scene]:
    """
    Filter out scenes shorter than a minimum duration.

    Very short scenes (< 0.5s) are often false positives from scene detection
    or transitions. This function removes them.

    Args:
        scenes: List of Scene objects to filter
        min_duration: Minimum scene duration in seconds (default: 0.5)

    Returns:
        List of Scene objects with duration >= min_duration

    Examples:
        >>> scenes = [
        ...     Scene(0.0, 0.2, 0, 6),      # Too short
        ...     Scene(0.2, 5.0, 6, 150),    # OK
        ...     Scene(5.0, 5.3, 150, 159),  # Too short
        ...     Scene(5.3, 10.0, 159, 300)  # OK
        ... ]
        >>> filtered = filter_short_scenes(scenes, min_duration=0.5)
        >>> len(filtered)
        2
        >>> [s.duration for s in filtered]
        [4.8, 4.7]

    Notes:
        - Empty list returns empty list
        - If all scenes are too short, returns empty list
        - Does not modify original scenes list
    """
    if min_duration < 0:
        raise ValueError(f"min_duration must be >= 0, got {min_duration}")

    return [scene for scene in scenes if scene.duration >= min_duration]


def merge_short_scenes(
    scenes: List[Scene],
    min_duration: float = 0.5
) -> List[Scene]:
    """
    Merge scenes shorter than min_duration with adjacent scenes.

    Instead of removing short scenes, this function merges them with
    their neighbors to preserve video coverage.

    Args:
        scenes: List of Scene objects to process
        min_duration: Minimum scene duration in seconds (default: 0.5)

    Returns:
        List of Scene objects with all scenes >= min_duration
        Short scenes are merged with the following scene

    Examples:
        >>> scenes = [
        ...     Scene(0.0, 5.0, 0, 150),
        ...     Scene(5.0, 5.2, 150, 156),  # Short
        ...     Scene(5.2, 10.0, 156, 300)
        ... ]
        >>> merged = merge_short_scenes(scenes, min_duration=0.5)
        >>> len(merged)
        2
        >>> merged[1].start_time
        5.0
        >>> merged[1].end_time
        10.0

    Notes:
        - Last short scene is merged with previous scene
        - Consecutive short scenes are merged together
        - Empty list returns empty list
        - Single scene always returns as-is
    """
    if min_duration < 0:
        raise ValueError(f"min_duration must be >= 0, got {min_duration}")

    if not scenes:
        return []

    if len(scenes) == 1:
        return scenes.copy()

    merged = []
    i = 0

    while i < len(scenes):
        current = scenes[i]

        # If current scene is long enough, add it
        if current.duration >= min_duration:
            merged.append(current)
            i += 1
        else:
            # Merge with next scene if available
            if i + 1 < len(scenes):
                next_scene = scenes[i + 1]
                # Create merged scene
                merged_scene = Scene(
                    start_time=current.start_time,
                    end_time=next_scene.end_time,
                    start_frame=current.start_frame,
                    end_frame=next_scene.end_frame,
                    metric_value=max(current.metric_value, next_scene.metric_value)
                )
                merged.append(merged_scene)
                i += 2  # Skip both scenes
            else:
                # Last scene is short, merge with previous
                if merged:
                    prev = merged.pop()
                    merged_scene = Scene(
                        start_time=prev.start_time,
                        end_time=current.end_time,
                        start_frame=prev.start_frame,
                        end_frame=current.end_frame,
                        metric_value=max(prev.metric_value, current.metric_value)
                    )
                    merged.append(merged_scene)
                else:
                    # Only scene, keep it even if short
                    merged.append(current)
                i += 1

    return merged


def get_scene_at_time(scenes: List[Scene], timestamp: float) -> Optional[Scene]:
    """
    Find the scene containing a specific timestamp.

    Args:
        scenes: List of Scene objects to search
        timestamp: Timestamp in seconds

    Returns:
        Scene object containing the timestamp, or None if not found

    Examples:
        >>> scenes = [
        ...     Scene(0.0, 5.0, 0, 150),
        ...     Scene(5.0, 10.0, 150, 300),
        ...     Scene(10.0, 15.0, 300, 450)
        ... ]
        >>> scene = get_scene_at_time(scenes, 7.5)
        >>> scene.start_time
        5.0
        >>> get_scene_at_time(scenes, 20.0) is None
        True

    Notes:
        - Returns None if scenes list is empty
        - Returns None if timestamp is outside all scenes
        - Uses inclusive start, exclusive end (start <= t < end)
    """
    for scene in scenes:
        if scene.start_time <= timestamp < scene.end_time:
            return scene

    # Check if timestamp exactly matches last scene's end time
    if scenes and abs(scenes[-1].end_time - timestamp) < 0.001:
        return scenes[-1]

    return None
