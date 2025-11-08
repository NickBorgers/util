"""Synthetic video generation utilities for testing smart-crop-video.

This module provides functions to programmatically generate test videos with
known characteristics (motion in specific regions, static backgrounds, color
patterns) using FFmpeg. This enables precise testing of crop positioning and
acceleration features.
"""

import subprocess
import tempfile
from pathlib import Path
from typing import Tuple, Optional, List, Dict, Any
from dataclasses import dataclass


@dataclass
class VideoConfig:
    """Configuration for generating a test video."""
    width: int = 1920
    height: int = 1080
    duration: float = 5.0
    fps: int = 30
    codec: str = "libx264"
    preset: str = "ultrafast"
    pix_fmt: str = "yuv420p"


@dataclass
class MotionRegion:
    """Defines a region with motion in the video."""
    x: int  # X position (0-1 normalized, or pixel coordinates)
    y: int  # Y position (0-1 normalized, or pixel coordinates)
    size: int  # Size of moving object in pixels
    color: str  # FFmpeg color name or hex code
    speed: int  # Movement speed (pixels per second)
    direction: str = "horizontal"  # "horizontal", "vertical", or "circular"


def create_video_with_motion_in_region(
    output_path: Path,
    motion_region: MotionRegion,
    config: Optional[VideoConfig] = None,
    background_color: str = "black"
) -> None:
    """
    Generate a test video with motion in a specific region.

    Creates a video with a static background and a moving object in the
    specified region. Useful for testing Motion Priority crop strategy.

    Args:
        output_path: Where to save the generated video
        motion_region: Configuration for the moving object
        config: Video configuration (resolution, duration, etc.)
        background_color: FFmpeg color name for background

    Example:
        >>> motion = MotionRegion(x=100, y=100, size=50, color="red", speed=100)
        >>> create_video_with_motion_in_region(Path("test.mov"), motion)
    """
    if config is None:
        config = VideoConfig()

    # Calculate object position based on region coordinates
    # If x/y are between 0-1, treat as normalized; otherwise as pixels
    if motion_region.x <= 1:
        x_pos = int(motion_region.x * config.width)
    else:
        x_pos = motion_region.x

    if motion_region.y <= 1:
        y_pos = int(motion_region.y * config.height)
    else:
        y_pos = motion_region.y

    # Calculate movement pattern based on direction
    if motion_region.direction == "horizontal":
        # Oscillate horizontally
        move_distance = min(200, config.width // 4)  # Move 200px or 1/4 width
        x_expr = f"{x_pos}+{move_distance}*sin(2*PI*t/{config.duration})"
        y_expr = str(y_pos)
    elif motion_region.direction == "vertical":
        # Oscillate vertically
        move_distance = min(200, config.height // 4)
        x_expr = str(x_pos)
        y_expr = f"{y_pos}+{move_distance}*sin(2*PI*t/{config.duration})"
    else:  # circular
        # Circular motion
        radius = min(100, config.width // 8)
        x_expr = f"{x_pos}+{radius}*cos(2*PI*t/{config.duration})"
        y_expr = f"{y_pos}+{radius}*sin(2*PI*t/{config.duration})"

    # Build FFmpeg command with drawbox filter for moving rectangle
    # Use nullsrc to generate blank video, then draw colored background and moving box
    cmd = [
        "ffmpeg", "-y",  # Overwrite output
        "-f", "lavfi",
        "-i", f"color=c={background_color}:s={config.width}x{config.height}:d={config.duration}:r={config.fps}",
        "-vf", f"drawbox=x={x_expr}:y={y_expr}:w={motion_region.size}:h={motion_region.size}:color={motion_region.color}:t=fill",
        "-c:v", config.codec,
        "-preset", config.preset,
        "-pix_fmt", config.pix_fmt,
        str(output_path)
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"FFmpeg failed to generate video: {result.stderr}")


def create_video_with_gradient(
    output_path: Path,
    gradient_direction: str = "horizontal",
    config: Optional[VideoConfig] = None,
    colors: Tuple[str, str] = ("blue", "red")
) -> None:
    """
    Generate a video with a color gradient.

    Useful for testing crop position by analyzing which colors appear in
    the output video.

    Args:
        output_path: Where to save the generated video
        gradient_direction: "horizontal" or "vertical"
        config: Video configuration
        colors: Tuple of (start_color, end_color) for gradient
    """
    if config is None:
        config = VideoConfig()

    # FFmpeg gradients filter
    if gradient_direction == "horizontal":
        gradient_filter = f"haldclutsrc=0:linear,format=rgb24,scale={config.width}x{config.height}"
    else:
        gradient_filter = f"haldclutsrc=0:linear,format=rgb24,scale={config.width}x{config.height},transpose=1"

    # Simpler approach: use color and fade filters
    cmd = [
        "ffmpeg", "-y",
        "-f", "lavfi",
        "-i", f"color=c={colors[0]}:s={config.width}x{config.height}:d={config.duration}:r={config.fps}",
        "-f", "lavfi",
        "-i", f"color=c={colors[1]}:s={config.width}x{config.height}:d={config.duration}:r={config.fps}",
        "-filter_complex", f"[0][1]blend=all_expr='A*(1-X/W)+B*(X/W)'",
        "-c:v", config.codec,
        "-preset", config.preset,
        "-pix_fmt", config.pix_fmt,
        str(output_path)
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"FFmpeg failed to generate gradient video: {result.stderr}")


def create_video_with_subject(
    output_path: Path,
    subject_position: Tuple[float, float],  # (x, y) normalized 0-1
    subject_size: int = 200,
    config: Optional[VideoConfig] = None,
    subject_shape: str = "circle",
    subject_color: str = "white",
    background_color: str = "black"
) -> None:
    """
    Generate a video with a prominent subject at a specific position.

    Useful for testing Subject Detection crop strategy.

    Args:
        output_path: Where to save the generated video
        subject_position: (x, y) position normalized 0-1
        subject_size: Size of subject in pixels
        config: Video configuration
        subject_shape: "circle" or "rectangle"
        subject_color: FFmpeg color for subject
        background_color: FFmpeg color for background
    """
    if config is None:
        config = VideoConfig()

    x_pos = int(subject_position[0] * config.width)
    y_pos = int(subject_position[1] * config.height)

    if subject_shape == "circle":
        # Draw a filled circle using drawbox with rounded corners (approximation)
        # FFmpeg doesn't have native circle, so we use a filled rectangle with edges
        draw_filter = (
            f"drawbox=x={x_pos - subject_size // 2}:y={y_pos - subject_size // 2}:"
            f"w={subject_size}:h={subject_size}:color={subject_color}:t=fill"
        )
        # Add smaller overlapping boxes to approximate circle
        # This is a simple approximation; for more precise circles, use drawtext with a circle character
    else:
        # Rectangle
        draw_filter = (
            f"drawbox=x={x_pos - subject_size // 2}:y={y_pos - subject_size // 2}:"
            f"w={subject_size}:h={subject_size}:color={subject_color}:t=fill"
        )

    # Add edges/contrast around subject for edge detection
    # Draw a border around the subject
    border_size = 10
    border_filter = (
        f"drawbox=x={x_pos - subject_size // 2 - border_size}:"
        f"y={y_pos - subject_size // 2 - border_size}:"
        f"w={subject_size + 2 * border_size}:h={subject_size + 2 * border_size}:"
        f"color={subject_color}:t={border_size}"
    )

    cmd = [
        "ffmpeg", "-y",
        "-f", "lavfi",
        "-i", f"color=c={background_color}:s={config.width}x{config.height}:d={config.duration}:r={config.fps}",
        "-vf", f"{draw_filter},{border_filter}",
        "-c:v", config.codec,
        "-preset", config.preset,
        "-pix_fmt", config.pix_fmt,
        str(output_path)
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"FFmpeg failed to generate subject video: {result.stderr}")


@dataclass
class SceneConfig:
    """Configuration for a scene in a multi-scene video."""
    duration: float
    motion_level: str = "high"  # "high", "medium", "low"
    background_color: str = "black"
    object_color: str = "white"
    object_size: int = 100


def create_video_with_scenes(
    output_path: Path,
    scenes: List[SceneConfig],
    config: Optional[VideoConfig] = None
) -> Dict[str, Any]:
    """
    Generate a multi-scene video with varying motion levels.

    Useful for testing intelligent acceleration features (detecting and
    speeding up boring sections).

    Args:
        output_path: Where to save the generated video
        scenes: List of scene configurations
        config: Base video configuration

    Returns:
        Dictionary with scene timestamps and characteristics:
        {
            "scenes": [
                {"start": 0.0, "end": 5.0, "motion_level": "high"},
                {"start": 5.0, "end": 10.0, "motion_level": "low"},
                ...
            ],
            "total_duration": 15.0
        }
    """
    if config is None:
        config = VideoConfig()

    # Create temporary files for each scene
    scene_files = []
    scene_info = []
    current_time = 0.0

    try:
        for i, scene in enumerate(scenes):
            scene_file = Path(tempfile.gettempdir()) / f"scene_{i}.mov"
            scene_files.append(scene_file)

            # Adjust motion parameters based on motion level
            if scene.motion_level == "high":
                speed = 200  # Fast movement
                size = scene.object_size
            elif scene.motion_level == "medium":
                speed = 50  # Moderate movement
                size = scene.object_size // 2
            else:  # low
                speed = 10  # Very slow movement
                size = scene.object_size // 4

            # Create scene with appropriate motion
            scene_config = VideoConfig(
                width=config.width,
                height=config.height,
                duration=scene.duration,
                fps=config.fps,
                codec=config.codec,
                preset=config.preset
            )

            motion = MotionRegion(
                x=config.width // 2,
                y=config.height // 2,
                size=size,
                color=scene.object_color,
                speed=speed,
                direction="horizontal"
            )

            create_video_with_motion_in_region(
                scene_file,
                motion,
                scene_config,
                background_color=scene.background_color
            )

            scene_info.append({
                "start": current_time,
                "end": current_time + scene.duration,
                "motion_level": scene.motion_level,
                "duration": scene.duration
            })
            current_time += scene.duration

        # Concatenate scenes
        concat_file = Path(tempfile.gettempdir()) / "concat_list.txt"
        with open(concat_file, "w") as f:
            for scene_file in scene_files:
                f.write(f"file '{scene_file}'\n")

        cmd = [
            "ffmpeg", "-y",
            "-f", "concat",
            "-safe", "0",
            "-i", str(concat_file),
            "-c", "copy",
            str(output_path)
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            raise RuntimeError(f"FFmpeg failed to concatenate scenes: {result.stderr}")

        return {
            "scenes": scene_info,
            "total_duration": current_time
        }

    finally:
        # Clean up temporary scene files
        for scene_file in scene_files:
            if scene_file.exists():
                scene_file.unlink()
        concat_file = Path(tempfile.gettempdir()) / "concat_list.txt"
        if concat_file.exists():
            concat_file.unlink()


def create_test_video_with_audio(
    output_path: Path,
    config: Optional[VideoConfig] = None,
    audio_frequency: int = 440,  # Hz (A4 note)
    audio_duration: Optional[float] = None
) -> None:
    """
    Generate a test video with audio tone.

    Useful for testing audio preservation and tempo matching in acceleration.

    Args:
        output_path: Where to save the generated video
        config: Video configuration
        audio_frequency: Frequency of audio tone in Hz
        audio_duration: Audio duration (defaults to video duration)
    """
    if config is None:
        config = VideoConfig()

    if audio_duration is None:
        audio_duration = config.duration

    # Generate video with audio using sine wave
    cmd = [
        "ffmpeg", "-y",
        "-f", "lavfi",
        "-i", f"color=c=blue:s={config.width}x{config.height}:d={config.duration}:r={config.fps}",
        "-f", "lavfi",
        "-i", f"sine=frequency={audio_frequency}:duration={audio_duration}",
        "-c:v", config.codec,
        "-c:a", "aac",
        "-preset", config.preset,
        "-pix_fmt", config.pix_fmt,
        "-shortest",
        str(output_path)
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"FFmpeg failed to generate video with audio: {result.stderr}")
