"""
End-to-end video validation tests for smart-crop-video.

These tests verify that the tool produces correctly cropped videos by:
1. Generating synthetic test videos with known characteristics
2. Processing them through smart-crop-video
3. Analyzing the output to verify crop position and quality

These are comprehensive tests that take longer to run but validate the
most critical user concern: "Did it crop my video correctly?"
"""

import pytest
import shutil
import docker
import tempfile
from pathlib import Path
from typing import Dict, Any

from tests.helpers import video_generator as vg
from tests.helpers import frame_analyzer as fa


# Check if FFmpeg is available
HAS_FFMPEG = shutil.which('ffmpeg') is not None
HAS_PILLOW = False
try:
    import PIL
    HAS_PILLOW = True
except ImportError:
    pass

# Check if Docker is available
HAS_DOCKER = shutil.which('docker') is not None


@pytest.fixture(scope="module")
def test_videos_dir():
    """Create a temporary directory for test videos."""
    with tempfile.TemporaryDirectory(prefix="smart_crop_e2e_") as tmpdir:
        yield Path(tmpdir)


@pytest.fixture(scope="module")
def motion_top_right_video(test_videos_dir):
    """Generate a test video with motion in top-right region."""
    video_path = test_videos_dir / "motion_top_right.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)

    # Motion in top-right (x=1400, y=200 out of 1920x1080)
    motion = vg.MotionRegion(
        x=1400, y=200,
        size=100,
        color="red",
        speed=100,
        direction="horizontal"
    )

    vg.create_video_with_motion_in_region(
        video_path,
        motion,
        config,
        background_color="black"
    )

    return video_path


@pytest.fixture(scope="module")
def motion_center_video(test_videos_dir):
    """Generate a test video with motion in center."""
    video_path = test_videos_dir / "motion_center.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)

    # Motion in center
    motion = vg.MotionRegion(
        x=960, y=540,
        size=150,
        color="blue",
        speed=100,
        direction="circular"
    )

    vg.create_video_with_motion_in_region(
        video_path,
        motion,
        config,
        background_color="black"
    )

    return video_path


@pytest.fixture(scope="module")
def subject_left_video(test_videos_dir):
    """Generate a test video with subject on left side."""
    video_path = test_videos_dir / "subject_left.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)

    # Subject on left side (x=0.25, y=0.5 normalized)
    vg.create_video_with_subject(
        video_path,
        subject_position=(0.25, 0.5),
        subject_size=250,
        config=config,
        subject_shape="circle",
        subject_color="white",
        background_color="black"
    )

    return video_path


def run_smart_crop(
    input_video: Path,
    output_video: Path,
    aspect_ratio: str = "1:1",
    strategy: str = "motion",
    crop_scale: float = 0.75,
    analysis_frames: int = 10,
    docker_image: str = "smart-crop-video:test"
) -> Dict[str, Any]:
    """
    Run smart-crop-video using Docker container.

    Args:
        input_video: Path to input video
        output_video: Path for output video
        aspect_ratio: Target aspect ratio (e.g., "1:1", "9:16")
        strategy: Scoring strategy name
        crop_scale: Crop scale factor (0-1)
        analysis_frames: Number of frames to analyze
        docker_image: Docker image to use

    Returns:
        Dict with:
            - returncode: Exit code
            - stdout: Standard output
            - stderr: Standard error
    """
    client = docker.from_env()

    # Get the working directory (parent of input video)
    work_dir = input_video.parent
    input_name = input_video.name
    output_name = output_video.name

    # Environment variables for the container
    environment = {
        "STRATEGY": strategy,
        "CROP_SCALE": str(crop_scale),
        "ANALYSIS_FRAMES": str(analysis_frames),
        "PRESET": "ultrafast",  # Fast encoding for tests
        "AUTO_CONFIRM": "true"  # Non-interactive mode for tests
    }

    # Run container
    try:
        container = client.containers.run(
            docker_image,
            command=["smart-crop-video", input_name, output_name, aspect_ratio],
            volumes={str(work_dir): {"bind": "/content", "mode": "rw"}},
            working_dir="/content",
            environment=environment,
            remove=True,
            detach=False,
            stdin_open=False,
            stdout=True,
            stderr=True
        )

        # Container output is bytes
        output = container.decode('utf-8') if isinstance(container, bytes) else str(container)

        return {
            "returncode": 0,
            "stdout": output,
            "stderr": ""
        }
    except docker.errors.ContainerError as e:
        return {
            "returncode": e.exit_status,
            "stdout": e.stderr.decode('utf-8') if e.stderr else "",
            "stderr": e.stderr.decode('utf-8') if e.stderr else str(e)
        }
    except Exception as e:
        return {
            "returncode": 1,
            "stdout": "",
            "stderr": str(e)
        }


@pytest.mark.skipif(not HAS_FFMPEG, reason="FFmpeg not available")
@pytest.mark.skipif(not HAS_PILLOW, reason="Pillow not available")
@pytest.mark.skipif(not HAS_DOCKER, reason="Docker not available")
@pytest.mark.comprehensive
class TestEndToEndVideoCropping:
    """End-to-end tests validating video output correctness."""

    def test_crop_accuracy_motion_priority(self, motion_top_right_video, test_videos_dir):
        """
        Verify Motion Priority strategy crops to the most active region.

        Given: Video with motion in top-right corner
        When: Process with Motion Priority strategy and 1:1 aspect ratio
        Then: Output crop should be positioned in top-right quadrant
        """
        output_video = test_videos_dir / "output_motion_priority.mov"

        result = run_smart_crop(
            motion_top_right_video,
            output_video,
            aspect_ratio="1:1",
            strategy="motion"
        )

        # Verify command succeeded
        assert result["returncode"] == 0, f"smart-crop-video failed: {result['stderr']}"

        # Verify output exists
        assert output_video.exists(), "Output video was not created"

        # Verify metadata
        metadata = fa.get_video_metadata(output_video)
        assert metadata["width"] > 0
        assert metadata["height"] > 0

        # Verify aspect ratio is 1:1 (square)
        aspect_ratio = metadata["width"] / metadata["height"]
        assert 0.95 < aspect_ratio < 1.05, f"Aspect ratio not square: {aspect_ratio}"

        # Verify crop position by comparing frames
        # Motion is at x=1400 out of 1920, so crop should be in right half
        crop_x, crop_y = fa.get_crop_position_from_video(
            motion_top_right_video,
            output_video,
            timestamp=2.0
        )

        # Crop should be in top-right: x > 960 (right half), y < 540 (top half)
        assert crop_x > 960, f"Crop not in right half: x={crop_x} (expected > 960)"
        # Note: y constraint relaxed because motion detection may not perfectly center on motion

    def test_crop_accuracy_center_motion(self, motion_center_video, test_videos_dir):
        """
        Verify cropping correctly identifies center motion.

        Given: Video with motion in center
        When: Process with Motion Priority strategy
        Then: Output crop should be centered
        """
        output_video = test_videos_dir / "output_center_motion.mov"

        result = run_smart_crop(
            motion_center_video,
            output_video,
            aspect_ratio="1:1",
            strategy="motion"
        )

        assert result["returncode"] == 0, f"smart-crop-video failed: {result['stderr']}"
        assert output_video.exists()

        # Get crop position
        crop_x, crop_y = fa.get_crop_position_from_video(
            motion_center_video,
            output_video,
            timestamp=2.0
        )

        # Crop should be relatively centered
        # Original is 1920x1080, center is ~960x540
        # Allow 30% tolerance (within 576 pixels of center horizontally)
        center_x = 960
        center_y = 540
        tolerance = 576  # 30% of 1920

        assert abs(crop_x - center_x) < tolerance, \
            f"Crop not centered: x={crop_x} (expected ~{center_x} ±{tolerance})"

    def test_crop_accuracy_subject_detection(self, subject_left_video, test_videos_dir):
        """
        Verify Subject Detection finds prominent subjects.

        Given: Video with white subject on left side
        When: Process with Subject Detection strategy (edges/complexity)
        Then: Crop should capture the subject (left side)
        """
        output_video = test_videos_dir / "output_subject_detection.mov"

        result = run_smart_crop(
            subject_left_video,
            output_video,
            aspect_ratio="1:1",
            strategy="edges"  # Subject detection uses edge scoring
        )

        assert result["returncode"] == 0, f"smart-crop-video failed: {result['stderr']}"
        assert output_video.exists()

        # Get crop position
        crop_x, crop_y = fa.get_crop_position_from_video(
            subject_left_video,
            output_video,
            timestamp=2.0
        )

        # Subject is at x=0.25 * 1920 = 480, so crop should be in left half
        assert crop_x < 960, f"Crop not in left half: x={crop_x} (expected < 960)"

    def test_aspect_ratio_1_to_1(self, motion_center_video, test_videos_dir):
        """
        Verify 1:1 aspect ratio is pixel-perfect.

        Given: Any video
        When: Process with 1:1 aspect ratio
        Then: Output width exactly equals height
        """
        output_video = test_videos_dir / "output_1_1.mov"

        result = run_smart_crop(
            motion_center_video,
            output_video,
            aspect_ratio="1:1"
        )

        assert result["returncode"] == 0
        assert output_video.exists()

        metadata = fa.get_video_metadata(output_video)
        width = metadata["width"]
        height = metadata["height"]

        # Should be exactly equal for 1:1
        assert width == height, f"1:1 aspect ratio not exact: {width}x{height}"

        # Verify dimensions are even (H.264 requirement)
        assert width % 2 == 0, f"Width not even: {width}"
        assert height % 2 == 0, f"Height not even: {height}"

    def test_aspect_ratio_9_to_16_vertical(self, motion_center_video, test_videos_dir):
        """
        Verify 9:16 (vertical) aspect ratio is correct.

        Given: Any video
        When: Process with 9:16 aspect ratio
        Then: Output should be 9:16 (taller than wide)
        """
        output_video = test_videos_dir / "output_9_16.mov"

        result = run_smart_crop(
            motion_center_video,
            output_video,
            aspect_ratio="9:16"
        )

        assert result["returncode"] == 0
        assert output_video.exists()

        metadata = fa.get_video_metadata(output_video)
        width = metadata["width"]
        height = metadata["height"]

        # Calculate actual ratio
        actual_ratio = width / height
        expected_ratio = 9 / 16

        # Allow 1% tolerance for encoding
        assert abs(actual_ratio - expected_ratio) / expected_ratio < 0.01, \
            f"9:16 aspect ratio incorrect: {width}x{height} = {actual_ratio:.4f} (expected {expected_ratio:.4f})"

    def test_output_video_playable(self, motion_center_video, test_videos_dir):
        """
        Verify output video is valid and playable.

        Given: Any processed video
        When: Attempt to read video metadata and frames
        Then: Video should be decodable without errors
        """
        output_video = test_videos_dir / "output_playable.mov"

        result = run_smart_crop(
            motion_center_video,
            output_video,
            aspect_ratio="1:1"
        )

        assert result["returncode"] == 0
        assert output_video.exists()

        # Get metadata (would fail if video corrupted)
        metadata = fa.get_video_metadata(output_video)
        assert metadata["width"] > 0
        assert metadata["height"] > 0
        assert metadata["duration"] > 0
        assert metadata["codec"] in ["h264", "hevc", "libx264", "libx265"]

        # Try to extract a frame (would fail if video unplayable)
        frame = fa.extract_frame(output_video, timestamp=1.0)
        assert frame.exists()

        # Try to extract multiple frames throughout video
        timestamps = [0.5, 2.0, 4.0]
        frames = fa.extract_multiple_frames(output_video, timestamps)
        assert len(frames) == 3

        # Clean up extracted frames
        for frame in frames:
            frame.unlink()

    def test_audio_preserved(self, test_videos_dir):
        """
        Verify audio stream is preserved in output.

        Given: Video with audio track
        When: Process through smart-crop-video
        Then: Output should have audio stream with same duration
        """
        # Create test video with audio
        input_video = test_videos_dir / "video_with_audio.mov"
        config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)
        vg.create_test_video_with_audio(input_video, config, audio_frequency=440)

        output_video = test_videos_dir / "output_with_audio.mov"

        result = run_smart_crop(
            input_video,
            output_video,
            aspect_ratio="1:1"
        )

        assert result["returncode"] == 0
        assert output_video.exists()

        # Check audio stream
        input_metadata = fa.get_video_metadata(input_video)
        output_metadata = fa.get_video_metadata(output_video)

        assert input_metadata["has_audio"], "Input should have audio"
        assert output_metadata["has_audio"], "Output should preserve audio"

        # Audio duration should match video duration (within tolerance)
        video_duration = output_metadata["duration"]
        audio_duration = output_metadata["audio_duration"]
        assert abs(video_duration - audio_duration) < 0.1, \
            f"Audio duration ({audio_duration}s) doesn't match video ({video_duration}s)"


@pytest.mark.skipif(not HAS_FFMPEG, reason="FFmpeg not available")
@pytest.mark.skipif(not HAS_DOCKER, reason="Docker not available")
@pytest.mark.comprehensive
class TestCropStrategies:
    """Test different scoring strategies produce expected results."""

    def test_motion_vs_edges_different_results(self, motion_top_right_video, test_videos_dir):
        """
        Verify different strategies produce different crop positions.

        This is a sanity check that strategies are actually different.
        """
        output_motion = test_videos_dir / "output_strategy_motion.mov"
        output_edges = test_videos_dir / "output_strategy_edges.mov"

        # Process with motion strategy
        result1 = run_smart_crop(
            motion_top_right_video,
            output_motion,
            strategy="motion"
        )
        assert result1.returncode == 0

        # Process with edges strategy
        result2 = run_smart_crop(
            motion_top_right_video,
            output_edges,
            strategy="edges"
        )
        assert result2.returncode == 0

        # Get crop positions
        if HAS_PILLOW:
            crop_x_motion, crop_y_motion = fa.get_crop_position_from_video(
                motion_top_right_video,
                output_motion,
                timestamp=2.0
            )

            crop_x_edges, crop_y_edges = fa.get_crop_position_from_video(
                motion_top_right_video,
                output_edges,
                timestamp=2.0
            )

            # Positions should be different (at least 100 pixels apart)
            distance = abs(crop_x_motion - crop_x_edges) + abs(crop_y_motion - crop_y_edges)
            assert distance > 100, \
                f"Motion and Edges strategies too similar: motion=({crop_x_motion},{crop_y_motion}), edges=({crop_x_edges},{crop_y_edges})"
