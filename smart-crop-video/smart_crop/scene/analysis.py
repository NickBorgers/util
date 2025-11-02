"""
Scene analysis utilities for smart-crop-video.

This module provides functions for analyzing scenes in videos:
- Extracting scene thumbnails for preview
- Analyzing scene metrics (motion, complexity, edges)
- Identifying boring sections for intelligent acceleration
- Temporal pattern analysis and scene selection

All functions are designed to be testable with minimal FFmpeg dependencies.
"""
import subprocess
import re
import os
from pathlib import Path
from typing import List, Tuple, Optional, Callable
from smart_crop.analysis.scenes import Scene


# ============================================================================
# Pure Functions (Testable without FFmpeg)
# ============================================================================

def determine_primary_metric(strategy: str) -> str:
    """
    Determine which metric is most important for a given scoring strategy.

    This is a pure function that maps strategies to their primary metrics.
    Used for scene analysis to focus on the most relevant visual feature.

    Args:
        strategy: Name of the scoring strategy (e.g., 'Subject Detection', 'Motion Priority')

    Returns:
        Metric name: 'motion', 'complexity', or 'edges'

    Examples:
        >>> determine_primary_metric('Motion Priority')
        'motion'
        >>> determine_primary_metric('Visual Detail')
        'complexity'
        >>> determine_primary_metric('Subject Detection')
        'edges'
    """
    strategy_metrics = {
        'Subject Detection': 'edges',
        'Motion Priority': 'motion',
        'Visual Detail': 'complexity',
        'Balanced': 'motion',  # Default to motion
        'Color Focus': 'edges',  # Use edges as proxy
    }

    # Handle spatial strategies (e.g., 'Spatial:Top-Left')
    if 'Spatial:' in strategy:
        return 'motion'  # Default for spatial strategies

    return strategy_metrics.get(strategy, 'motion')


def identify_boring_sections(
    scenes: List[Scene],
    percentile_threshold: float = 30.0
) -> List[Tuple[int, float]]:
    """
    Identify boring sections based on scene metric values.

    Boring sections are scenes below a percentile threshold (default 30th percentile).
    More boring scenes get higher speedup factors (2x to 4x).

    Args:
        scenes: List of Scene objects with metric_value populated
        percentile_threshold: Percentile below which scenes are considered boring (0-100)

    Returns:
        List of (scene_index, speedup_factor) tuples
        - scene_index: 0-based index into scenes list
        - speedup_factor: 2.0 to 4.0 (higher = more boring)

    Examples:
        >>> scenes = [Scene(0, 1, 0, 30, metric_value=5.0),
        ...           Scene(1, 2, 30, 60, metric_value=10.0),
        ...           Scene(2, 3, 60, 90, metric_value=2.0)]
        >>> boring = identify_boring_sections(scenes, percentile_threshold=50.0)
        >>> len(boring)  # 2 out of 3 scenes below 50th percentile
        2
        >>> boring[0][0]  # First boring scene is scene 0
        0
        >>> 2.0 <= boring[0][1] <= 4.0  # Speedup factor in range
        True
    """
    if not scenes:
        return []

    # Calculate threshold (Nth percentile)
    metric_values = [s.metric_value for s in scenes]
    metric_values.sort()
    threshold_idx = int(len(metric_values) * (percentile_threshold / 100.0))
    threshold = metric_values[threshold_idx] if threshold_idx < len(metric_values) else metric_values[0]

    # Identify boring sections and calculate speedup factors
    boring_sections = []
    for i, scene in enumerate(scenes):
        if scene.metric_value < threshold:
            # Calculate speedup based on how boring the scene is
            # Very boring (near 0) = 4x speed
            # Slightly boring (near threshold) = 2x speed
            if threshold > 0:
                ratio = scene.metric_value / threshold
                speedup = 2.0 + (2.0 * (1.0 - ratio))  # Range: 2x to 4x
            else:
                speedup = 3.0

            boring_sections.append((i, min(speedup, 4.0)))  # Cap at 4x

    return boring_sections


def calculate_speedup_factor(
    metric_value: float,
    threshold: float,
    min_speedup: float = 2.0,
    max_speedup: float = 4.0
) -> float:
    """
    Calculate speedup factor for a scene based on its metric value.

    Pure function for testability.

    Args:
        metric_value: Scene's metric value
        threshold: Threshold below which scene is considered boring
        min_speedup: Minimum speedup factor (for scenes near threshold)
        max_speedup: Maximum speedup factor (for very boring scenes)

    Returns:
        Speedup factor between min_speedup and max_speedup

    Examples:
        >>> calculate_speedup_factor(5.0, 10.0)  # Half of threshold
        3.0
        >>> calculate_speedup_factor(0.0, 10.0)  # Very boring
        4.0
        >>> calculate_speedup_factor(9.0, 10.0)  # Near threshold
        2.2
    """
    if threshold <= 0:
        return (min_speedup + max_speedup) / 2.0

    ratio = metric_value / threshold
    speedup = min_speedup + ((max_speedup - min_speedup) * (1.0 - ratio))
    return min(speedup, max_speedup)


# ============================================================================
# FFmpeg-Dependent Functions (Require video file and subprocess)
# ============================================================================

def extract_metric_from_showinfo(output: str, metric: str) -> List[float]:
    """
    Extract metric values from FFmpeg showinfo filter output.

    Parses FFmpeg stderr output to extract metric arrays like:
    - mean:[123.45 67.89 ...] → motion/brightness analysis
    - stdev:[12.34 56.78 ...] → complexity/variance analysis

    Args:
        output: FFmpeg stderr output containing showinfo data
        metric: Metric name to extract ('mean', 'stdev', etc.)

    Returns:
        List of float values (first value from each array, typically Y channel)

    Examples:
        >>> output = "[Parsed_showinfo_1] mean:[123.45 67.89 90.12]"
        >>> extract_metric_from_showinfo(output, 'mean')
        [123.45]
    """
    pattern = rf'{metric}:\[([0-9. ]+)\]'
    matches = re.findall(pattern, output)
    values = []
    for match in matches:
        # Take first value from each match (for Y channel in YUV)
        parts = match.split()
        if parts:
            values.append(float(parts[0]))
    return values


def run_ffmpeg(cmd: List[str]) -> str:
    """
    Run FFmpeg command and return stderr output.

    Wrapper function for subprocess calls to FFmpeg. Returns stderr
    because FFmpeg outputs analysis data (like showinfo) to stderr.

    Args:
        cmd: FFmpeg command as list of strings

    Returns:
        FFmpeg stderr output as string

    Examples:
        >>> cmd = ['ffmpeg', '-i', 'video.mp4', '-f', 'null', '-']
        >>> output = run_ffmpeg(cmd)
        >>> 'ffmpeg version' in output
        True
    """
    result = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    return result.stderr


def analyze_scene_metrics(
    input_file: str,
    scene: Scene,
    x: int,
    y: int,
    crop_w: int,
    crop_h: int,
    metric_type: str,
    sample_frames: int = 10
) -> float:
    """
    Analyze a specific metric for a scene.

    Samples frames from the scene and calculates the specified metric
    (motion, complexity, or edges) for the cropped region.

    Args:
        input_file: Path to video file
        scene: Scene object with start/end time and frames
        x, y: Crop position coordinates
        crop_w, crop_h: Crop dimensions
        metric_type: Type of metric to analyze ('motion', 'complexity', 'edges')
        sample_frames: Number of frames to sample from scene (default: 10)

    Returns:
        Average metric value for the scene (higher = more interesting)
        Returns 0.0 if scene is too short or analysis fails

    Examples:
        >>> scene = Scene(5.0, 10.0, 150, 300)
        >>> score = analyze_scene_metrics(
        ...     'video.mp4', scene, 100, 100, 640, 640, 'motion', sample_frames=5
        ... )
        >>> score >= 0.0
        True
    """
    # Validate scene duration
    duration = scene.duration
    if duration < 0.1:  # Scene too short
        return 0.0

    # Calculate frame sampling
    total_scene_frames = int(scene.end_frame - scene.start_frame)
    if total_scene_frames < 1:
        return 0.0

    # Sample up to N frames evenly distributed across the scene
    sample_count = min(sample_frames, max(1, total_scene_frames))

    # Analyze based on metric type
    if metric_type == 'motion':
        # Analyze motion in this scene
        cmd = [
            'ffmpeg', '-ss', str(scene.start_time), '-t', str(duration),
            '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},showinfo',
            '-frames:v', str(sample_count),
            '-f', 'null', '-'
        ]
        output = run_ffmpeg(cmd)

        means = extract_metric_from_showinfo(output, 'mean')
        if len(means) > 1:
            diffs = [abs(means[i] - means[i-1]) for i in range(1, len(means))]
            return sum(diffs) / len(diffs) if diffs else 0.0
        return 0.0

    elif metric_type == 'complexity':
        # Analyze visual complexity (standard deviation)
        cmd = [
            'ffmpeg', '-ss', str(scene.start_time), '-t', str(duration),
            '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},showinfo',
            '-frames:v', str(sample_count),
            '-f', 'null', '-'
        ]
        output = run_ffmpeg(cmd)

        stdevs = extract_metric_from_showinfo(output, 'stdev')
        return sum(stdevs) / len(stdevs) if stdevs else 0.0

    elif metric_type == 'edges':
        # Analyze edge content
        cmd = [
            'ffmpeg', '-ss', str(scene.start_time), '-t', str(duration),
            '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},edgedetect=low=0.3:high=0.4:mode=colormix,showinfo',
            '-frames:v', str(sample_count),
            '-f', 'null', '-'
        ]
        output = run_ffmpeg(cmd)

        means = extract_metric_from_showinfo(output, 'mean')
        return sum(means) / len(means) if means else 0.0

    return 0.0


def extract_scene_thumbnails(
    input_file: str,
    scenes: List[Scene],
    x: int,
    y: int,
    crop_w: int,
    crop_h: int,
    base_name: str,
    progress_callback: Optional[Callable[[int, str], None]] = None,
    progress_offset: int = 0
) -> None:
    """
    Extract first and last frame thumbnails for each scene.

    Creates JPEG thumbnails for scene preview in the web UI. Mutates
    Scene objects by setting first_frame_path and last_frame_path.

    Args:
        input_file: Path to video file
        scenes: List of Scene objects to extract thumbnails for
        x, y: Crop position coordinates
        crop_w, crop_h: Crop dimensions
        base_name: Base name for output files (e.g., 'video')
        progress_callback: Optional callback(progress_pct, message) for progress updates
        progress_offset: Starting progress percentage (e.g., 40 means progress goes from 40-100%)

    Examples:
        >>> scenes = [Scene(0.0, 5.0, 0, 150), Scene(5.0, 10.0, 150, 300)]
        >>> extract_scene_thumbnails(
        ...     'video.mp4', scenes, 100, 100, 640, 640, 'video',
        ...     progress_callback=lambda pct, msg: print(f"{pct}%: {msg}")
        ... )
        >>> scenes[0].first_frame_path
        '.video_scene_1_first.jpg'
    """
    print("Extracting scene thumbnails for preview...")

    # Clean up old scene thumbnail files from previous runs
    for old_thumbnail in Path('.').glob(f".{base_name}_scene_*_first.jpg"):
        old_thumbnail.unlink()
    for old_thumbnail in Path('.').glob(f".{base_name}_scene_*_last.jpg"):
        old_thumbnail.unlink()

    print()

    total_extractions = len(scenes) * 2  # First + last frame for each scene
    current = 0
    progress_range = 100 - progress_offset  # Available progress range

    for i, scene in enumerate(scenes):
        # Extract first frame
        current += 1
        extraction_progress = (current / total_extractions) * progress_range
        progress_pct = int(progress_offset + extraction_progress)
        progress_msg = f"Extracting thumbnails for scene {i+1}/{len(scenes)} (first frame)"
        print(f"\r[{current:3d}/{total_extractions}] {progress_msg}...  ", end='', flush=True)

        if progress_callback:
            progress_callback(progress_pct, progress_msg)

        first_frame_path = f".{base_name}_scene_{i+1}_first.jpg"
        cmd = [
            'ffmpeg', '-ss', str(scene.start_time), '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y}',
            '-vframes', '1', '-q:v', '2',
            first_frame_path, '-y'
        ]
        subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        scene.first_frame_path = first_frame_path

        # Extract last frame (slightly before end to avoid black frames)
        current += 1
        extraction_progress = (current / total_extractions) * progress_range
        progress_pct = int(progress_offset + extraction_progress)
        progress_msg = f"Extracting thumbnails for scene {i+1}/{len(scenes)} (last frame)"
        print(f"\r[{current:3d}/{total_extractions}] {progress_msg}...  ", end='', flush=True)

        if progress_callback:
            progress_callback(progress_pct, progress_msg)

        last_frame_time = max(scene.start_time, scene.end_time - 0.1)
        last_frame_path = f".{base_name}_scene_{i+1}_last.jpg"
        cmd = [
            'ffmpeg', '-ss', str(last_frame_time), '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y}',
            '-vframes', '1', '-q:v', '2',
            last_frame_path, '-y'
        ]
        subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        scene.last_frame_path = last_frame_path

    print(f"\r{' '*80}\r✓ Extracted thumbnails for all {len(scenes)} scenes")
    print()
