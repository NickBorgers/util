"""Docker container management utilities for smart-crop-video tests."""

import pytest
import docker
import time
import requests
import socket
import os
from pathlib import Path
from typing import Generator, Dict, Any


def find_free_port() -> int:
    """Find a free port for the webserver."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(('', 0))
        s.listen(1)
        port = s.getsockname()[1]
    return port


@pytest.fixture(scope="session")
def docker_client() -> docker.DockerClient:
    """Create a Docker client for the test session."""
    return docker.from_env()


@pytest.fixture(scope="session")
def docker_image(docker_client: docker.DockerClient) -> str:
    """Build the Docker image for testing."""
    repo_root = Path(__file__).parent.parent.parent
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
def smart_crop_container(
    docker_client: docker.DockerClient,
    docker_image: str,
    temp_workdir: Path
) -> Generator[Dict[str, Any], None, None]:
    """
    Run smart-crop-video container and yield connection info.

    Yields dict with:
        - container: Docker container object
        - port: Port number (dynamically allocated)
        - base_url: Base URL for API calls
        - workdir: Path to temp working directory
    """
    container = None
    port = None
    try:
        # Find a free port to avoid conflicts between tests
        port = find_free_port()

        # Convert container path to host path for Docker-in-Docker
        # When running tests in Docker, we need to mount the actual host path
        import os
        host_workspace_dir = os.environ.get('HOST_WORKSPACE_DIR')
        if host_workspace_dir:
            # We're running in Docker, convert /workspace path to host path
            work_dir_str = str(temp_workdir)
            if work_dir_str.startswith('/workspace'):
                # Replace /workspace with the actual host path
                host_work_dir = work_dir_str.replace('/workspace', host_workspace_dir, 1)
            else:
                host_work_dir = work_dir_str
        else:
            # Running directly on host
            host_work_dir = str(temp_workdir)

        # Start container with port mapping and volume mount
        container = docker_client.containers.run(
            docker_image,
            command=["smart-crop-video", "example_movie.mov", "output.mov", "1:1"],
            detach=True,
            remove=False,  # We'll remove manually after logs
            stdin_open=True,  # Keep stdin open (enables TTY mode in script)
            tty=True,  # Allocate pseudo-TTY (makes sys.stdin.isatty() return True)
            ports={"8765/tcp": port},  # Map container port 8765 to dynamic host port
            volumes={
                host_work_dir: {"bind": "/content", "mode": "rw"}
            },
            working_dir="/content",
            environment={
                "PRESET": "ultrafast",  # Fast encoding for tests
                "ANALYSIS_FRAMES": "10",  # Fewer frames for faster tests
                "CROP_SCALE": "0.75",
                "AUTO_CONFIRM": "true"  # Non-interactive mode for tests
            }
        )

        # Wait for Flask server to be ready
        base_url = f"http://localhost:{port}"
        max_wait = 240  # Increased to allow for full workflow (analysis + encoding can take 3+ minutes)
        wait_interval = 0.5
        elapsed = 0

        print(f"\nWaiting for Flask server at {base_url}...")

        while elapsed < max_wait:
            try:
                response = requests.get(f"{base_url}/api/status", timeout=1)
                if response.status_code == 200:
                    print(f"Flask server ready after {elapsed:.1f}s on port {port}")
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
            "port": port,
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
            except Exception as e:
                print(f"Warning: Could not fetch container logs: {e}")

            # Stop container with timeout
            try:
                print(f"Stopping container on port {port}...")
                container.stop(timeout=5)
                print(f"Container stopped successfully")
            except Exception as e:
                print(f"Warning: Could not stop container gracefully: {e}")

            # Force remove container
            try:
                container.remove(force=True)
                print(f"Container removed successfully")
            except Exception as e:
                print(f"Warning: Could not remove container: {e}")

            # Small delay to ensure port is released
            time.sleep(0.5)
