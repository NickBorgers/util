"""
Pytest configuration and fixtures for smart-crop-video integration tests.

Provides Docker container management, test video files, and helper utilities.
"""

import pytest
import docker
import time
import subprocess
import requests
from pathlib import Path
from typing import Generator, Dict, Any
import tempfile
import shutil
import os


@pytest.fixture(scope="session")
def docker_client() -> docker.DockerClient:
    """Create a Docker client for the test session."""
    return docker.from_env()


@pytest.fixture(scope="session")
def test_video_path() -> Path:
    """Path to the example test video."""
    repo_root = Path(__file__).parent.parent
    video_path = repo_root / "example_movie.mov"
    if not video_path.exists():
        pytest.skip(f"Test video not found: {video_path}")
    return video_path


@pytest.fixture(scope="session")
def docker_image(docker_client: docker.DockerClient) -> str:
    """Build the Docker image for testing."""
    repo_root = Path(__file__).parent.parent
    print(f"\nBuilding Docker image from {repo_root}...")

    image, build_logs = docker_client.images.build(
        path=str(repo_root),
        tag="smart-crop-video:test",
        rm=True,
        forcerm=True
    )

    # Print build output
    for log in build_logs:
        if 'stream' in log:
            print(log['stream'].strip())

    return image.tags[0]


@pytest.fixture
def temp_workdir(test_video_path: Path) -> Generator[Path, None, None]:
    """Create a temporary working directory with a copy of the test video."""
    with tempfile.TemporaryDirectory(prefix="smart_crop_test_") as tmpdir:
        tmpdir_path = Path(tmpdir)

        # Copy test video to temp directory
        test_video_copy = tmpdir_path / test_video_path.name
        shutil.copy(test_video_path, test_video_copy)

        yield tmpdir_path


@pytest.fixture
def smart_crop_container(
    docker_client: docker.DockerClient,
    docker_image: str,
    temp_workdir: Path
) -> Generator[Dict[str, Any], None, None]:
    """
    Run smart-crop-video container and yield connection info.

    Yields dict with:
        - container: Docker container object
        - port: Port number (8765)
        - base_url: Base URL for API calls
        - workdir: Path to temp working directory
    """
    container = None
    try:
        # Start container with port mapping and volume mount
        container = docker_client.containers.run(
            docker_image,
            command=["example_movie.mov", "output.mov", "1:1"],
            detach=True,
            remove=False,  # We'll remove manually after logs
            stdin_open=True,  # Keep stdin open (enables TTY mode in script)
            tty=True,  # Allocate pseudo-TTY (makes sys.stdin.isatty() return True)
            ports={"8765/tcp": 8765},
            volumes={
                str(temp_workdir): {"bind": "/content", "mode": "rw"}
            },
            working_dir="/content",
            environment={
                "PRESET": "ultrafast",  # Fast encoding for tests
                "ANALYSIS_FRAMES": "10",  # Fewer frames for faster tests
                "CROP_SCALE": "0.75"
            }
        )

        # Wait for Flask server to be ready
        base_url = "http://localhost:8765"
        max_wait = 30
        wait_interval = 0.5
        elapsed = 0

        print(f"\nWaiting for Flask server at {base_url}...")

        while elapsed < max_wait:
            try:
                response = requests.get(f"{base_url}/api/status", timeout=1)
                if response.status_code == 200:
                    print(f"Flask server ready after {elapsed:.1f}s")
                    break
            except requests.exceptions.RequestException:
                pass

            time.sleep(wait_interval)
            elapsed += wait_interval
        else:
            # Timeout - print logs for debugging
            logs = container.logs().decode('utf-8')
            pytest.fail(f"Flask server didn't start within {max_wait}s. Logs:\n{logs}")

        yield {
            "container": container,
            "port": 8765,
            "base_url": base_url,
            "workdir": temp_workdir
        }

    finally:
        if container:
            # Get logs before removing (useful for debugging)
            try:
                logs = container.logs().decode('utf-8')
                if os.getenv('PYTEST_VERBOSE'):
                    print(f"\n=== Container logs ===\n{logs}\n===================")
            except:
                pass

            try:
                container.stop(timeout=5)
            except:
                pass

            try:
                container.remove(force=True)
            except:
                pass


@pytest.fixture
def api_client(smart_crop_container: Dict[str, Any]) -> Dict[str, Any]:
    """
    Provide an API client with helper methods for testing.

    Returns dict with:
        - base_url: Base URL for API calls
        - get_status: Function to get current status
        - select_crop: Function to select a crop option
        - choose_acceleration: Function to choose acceleration
        - select_scenes: Function to submit scene selections
    """
    base_url = smart_crop_container["base_url"]

    def get_status() -> Dict[str, Any]:
        """Get current application status."""
        # Increased timeout because FFmpeg analysis can block the Flask thread
        response = requests.get(f"{base_url}/api/status", timeout=30)
        response.raise_for_status()
        return response.json()

    def wait_for_status(
        target_status: str,
        timeout: float = 60,
        poll_interval: float = 0.5
    ) -> Dict[str, Any]:
        """Wait for a specific status and return the data."""
        elapsed = 0
        while elapsed < timeout:
            status_data = get_status()
            if status_data['status'] == target_status:
                return status_data
            time.sleep(poll_interval)
            elapsed += poll_interval

        pytest.fail(f"Timeout waiting for status '{target_status}' after {timeout}s")

    def select_crop(index: int) -> requests.Response:
        """Select a crop option by index (1-based)."""
        response = requests.post(f"{base_url}/api/select/{index}", timeout=10)
        response.raise_for_status()
        return response

    def choose_acceleration(enable: bool) -> requests.Response:
        """Choose whether to enable acceleration."""
        choice = "yes" if enable else "no"
        response = requests.post(f"{base_url}/api/acceleration/{choice}", timeout=10)
        response.raise_for_status()
        return response

    def select_scenes(selections: Dict[int, float]) -> requests.Response:
        """Submit scene selections (scene_index -> speedup_factor)."""
        response = requests.post(
            f"{base_url}/api/scene_selections",
            json={"selections": selections},
            timeout=10
        )
        response.raise_for_status()
        return response

    def get_preview_image(index: int) -> bytes:
        """Get preview image bytes."""
        response = requests.get(f"{base_url}/api/preview/{index}", timeout=5)
        response.raise_for_status()
        return response.content

    def get_scene_thumbnail(scene_id: str, frame_type: str) -> bytes:
        """Get scene thumbnail bytes."""
        response = requests.get(
            f"{base_url}/api/scene_thumbnail/{scene_id}/{frame_type}",
            timeout=5
        )
        response.raise_for_status()
        return response.content

    return {
        "base_url": base_url,
        "get_status": get_status,
        "wait_for_status": wait_for_status,
        "select_crop": select_crop,
        "choose_acceleration": choose_acceleration,
        "select_scenes": select_scenes,
        "get_preview_image": get_preview_image,
        "get_scene_thumbnail": get_scene_thumbnail,
    }


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
