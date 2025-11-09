"""
Pytest configuration and fixtures for smart-crop-video tests.

This file imports fixtures from helper modules and provides simple fixtures
for test video files and working directories.
"""

import pytest
import subprocess
from pathlib import Path
from typing import Generator, Dict, Any
import tempfile
import shutil

# Import fixtures from helper modules
from tests.helpers.docker_manager import docker_client, docker_image, smart_crop_container
from tests.helpers.api_helper import api_client


@pytest.fixture(scope="session")
def test_video_path() -> Path:
    """Path to the example test video."""
    repo_root = Path(__file__).parent.parent
    video_path = repo_root / "example_movie.mov"
    if not video_path.exists():
        pytest.skip(f"Test video not found: {video_path}")
    return video_path


@pytest.fixture
def temp_workdir(test_video_path: Path) -> Generator[Path, None, None]:
    """Create a temporary working directory with a copy of the test video."""
    # Create temp dir in /workspace for Docker-in-Docker compatibility
    # If running in test container, this ensures the host can mount the path
    workspace_dir = Path("/workspace")
    if workspace_dir.exists():
        # Running in Docker, use workspace
        base_dir = workspace_dir / "tests" / ".test_output"
        base_dir.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix="smart_crop_test_", dir=base_dir) as tmpdir:
            tmpdir_path = Path(tmpdir)

            # Copy test video to temp directory
            test_video_copy = tmpdir_path / test_video_path.name
            shutil.copy(test_video_path, test_video_copy)

            yield tmpdir_path
    else:
        # Running on host, use system temp
        with tempfile.TemporaryDirectory(prefix="smart_crop_test_") as tmpdir:
            tmpdir_path = Path(tmpdir)

            # Copy test video to temp directory
            test_video_copy = tmpdir_path / test_video_path.name
            shutil.copy(test_video_path, test_video_copy)

            yield tmpdir_path


@pytest.fixture
def ffmpeg_helper():
    """Helper functions for FFmpeg verification."""

    def get_video_info(video_path: Path) -> Dict[str, Any]:
        """Get video metadata using ffprobe."""
        cmd = [
            "ffprobe", "-v", "error",
            "-select_streams", "v:0",
            "-show_entries", "stream=width,height,duration,nb_frames",
            "-show_entries", "format=duration",
            "-of", "json",
            str(video_path)
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            pytest.fail(f"ffprobe failed: {result.stderr}")

        import json
        data = json.loads(result.stdout)

        stream = data['streams'][0] if data['streams'] else {}
        format_info = data.get('format', {})

        return {
            "width": int(stream.get('width', 0)),
            "height": int(stream.get('height', 0)),
            "duration": float(format_info.get('duration', 0)),
            "nb_frames": int(stream.get('nb_frames', 0)) if 'nb_frames' in stream else None
        }

    def verify_aspect_ratio(video_path: Path, expected_ratio: str) -> bool:
        """Verify video has expected aspect ratio."""
        info = get_video_info(video_path)
        width = info['width']
        height = info['height']

        # Parse expected ratio (e.g., "1:1", "9:16")
        aspect_w, aspect_h = map(int, expected_ratio.split(':'))

        # Check if aspect ratio matches (with small tolerance)
        actual_ratio = width / height
        expected_ratio_value = aspect_w / aspect_h
        tolerance = 0.01

        return abs(actual_ratio - expected_ratio_value) < tolerance

    def video_exists_and_valid(video_path: Path) -> bool:
        """Check if video exists and is valid."""
        if not video_path.exists():
            return False

        try:
            info = get_video_info(video_path)
            return info['width'] > 0 and info['height'] > 0
        except:
            return False

    return {
        "get_video_info": get_video_info,
        "verify_aspect_ratio": verify_aspect_ratio,
        "video_exists_and_valid": video_exists_and_valid,
    }
