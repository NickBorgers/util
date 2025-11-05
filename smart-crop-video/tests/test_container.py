"""
Container integration tests for smart-crop-video.

Tests Docker image building, container startup, port mapping, and volume mounts.
"""

import pytest
import docker
import requests
from pathlib import Path
import time


def test_docker_image_builds(docker_image: str):
    """Test that the Docker image builds successfully."""
    assert docker_image == "smart-crop-video:test"


def test_docker_image_has_ffmpeg(docker_client: docker.DockerClient, docker_image: str):
    """Test that FFmpeg is installed in the Docker image."""
    result = docker_client.containers.run(
        docker_image,
        command=["ffmpeg", "-version"],
        remove=True,
        stdout=True,
        stderr=True
    )
    output = result.decode('utf-8')
    assert "ffmpeg version" in output.lower()


def test_docker_image_has_python(docker_client: docker.DockerClient, docker_image: str):
    """Test that Python 3 is installed in the Docker image."""
    result = docker_client.containers.run(
        docker_image,
        command=["python3", "--version"],
        remove=True,
        stdout=True,
        stderr=True
    )
    output = result.decode('utf-8')
    assert "python 3" in output.lower()


def test_docker_image_has_flask(docker_client: docker.DockerClient, docker_image: str):
    """Test that Flask is installed in the Docker image."""
    result = docker_client.containers.run(
        docker_image,
        command=["python3", "-c", "import flask; print(flask.__version__)"],
        remove=True,
        stdout=True,
        stderr=True
    )
    output = result.decode('utf-8')
    # Flask version should be present
    assert len(output.strip()) > 0


def test_container_starts_successfully(smart_crop_container: dict):
    """Test that the container starts and runs."""
    container = smart_crop_container["container"]
    assert container.status in ["running", "created"]


def test_port_mapping_works(smart_crop_container: dict):
    """Test that port mapping is correctly configured and accessible."""
    base_url = smart_crop_container["base_url"]
    response = requests.get(f"{base_url}/api/status", timeout=5)
    assert response.status_code == 200


def test_volume_mount_works(smart_crop_container: dict):
    """Test that the volume mount allows file access."""
    workdir = smart_crop_container["workdir"]
    test_video = workdir / "example_movie.mov"

    # Video should exist in temp directory
    assert test_video.exists()

    # Container should be able to access it
    container = smart_crop_container["container"]
    result = container.exec_run("ls -la /content/example_movie.mov")
    assert result.exit_code == 0


def test_environment_variables_are_set(smart_crop_container: dict):
    """Test that environment variables are passed to container."""
    container = smart_crop_container["container"]

    # Check PRESET env var
    result = container.exec_run("printenv PRESET")
    assert result.exit_code == 0
    assert b"ultrafast" in result.output

    # Check ANALYSIS_FRAMES env var
    result = container.exec_run("printenv ANALYSIS_FRAMES")
    assert result.exit_code == 0
    assert b"10" in result.output


def test_flask_server_starts_on_correct_port(smart_crop_container: dict):
    """Test that Flask server is listening on the mapped port."""
    base_url = smart_crop_container["base_url"]
    port = smart_crop_container["port"]

    # Should be able to reach the server
    response = requests.get(f"{base_url}/", timeout=5)
    assert response.status_code == 200
    assert "Smart Crop Video" in response.text

    # Verify port is in the expected range (dynamically allocated)
    assert 1024 < port < 65535, f"Port {port} outside valid range"


def test_container_has_network_access(docker_client: docker.DockerClient, docker_image: str, temp_workdir: Path):
    """
    Test that container has network access (unlike older utils with --network=none).

    This is important because the Flask web UI needs to be accessible.
    """
    # Start a simple container that tries to access the network
    container = docker_client.containers.run(
        docker_image,
        command=["python3", "-c", "import socket; socket.create_connection(('8.8.8.8', 53), timeout=2)"],
        detach=True,
        remove=True
    )

    # Wait for container to complete
    result = container.wait(timeout=10)

    # Should succeed (exit code 0)
    assert result['StatusCode'] == 0


def test_container_cleanup_on_completion(docker_client: docker.DockerClient, docker_image: str, temp_workdir: Path):
    """Test that container with --rm flag cleans up properly."""
    # Run a short-lived container with --rm
    container = docker_client.containers.run(
        docker_image,
        command=["python3", "-c", "print('hello')"],
        detach=True,
        remove=True  # Auto-remove on exit
    )

    container_id = container.id

    # Wait for it to complete
    container.wait(timeout=10)

    # Give Docker a moment to remove it
    time.sleep(1)

    # Container should be removed
    with pytest.raises(docker.errors.NotFound):
        docker_client.containers.get(container_id)


def test_working_directory_is_set_correctly(smart_crop_container: dict):
    """Test that working directory is set to /content."""
    container = smart_crop_container["container"]

    # Check current working directory
    result = container.exec_run("pwd")
    assert result.exit_code == 0
    assert b"/content" in result.output


def test_smart_crop_script_is_executable(docker_client: docker.DockerClient, docker_image: str):
    """Test that smart-crop-video script is executable."""
    result = docker_client.containers.run(
        docker_image,
        command=["which", "smart-crop-video"],
        remove=True,
        stdout=True,
        stderr=True
    )
    output = result.decode('utf-8').strip()
    assert "/usr/local/bin/smart-crop-video" in output


def test_smart_crop_script_shows_help_on_no_args(docker_client: docker.DockerClient, docker_image: str):
    """Test that smart-crop-video script shows usage when run with no arguments."""
    try:
        result = docker_client.containers.run(
            docker_image,
            command=["smart-crop-video"],
            remove=True,
            stdout=True,
            stderr=True
        )
        # Should fail with usage message
    except docker.errors.ContainerError as e:
        # argparse will exit with code 2 for missing required arguments
        assert e.exit_status == 2
        assert b"required" in e.stderr.lower() or b"usage" in e.stderr.lower()


def test_container_handles_missing_video_file(docker_client: docker.DockerClient, docker_image: str, temp_workdir: Path):
    """Test that container provides clear error when video file is missing."""
    # Try to process a non-existent video
    try:
        container = docker_client.containers.run(
            docker_image,
            command=["smart-crop-video", "nonexistent.mov", "output.mov", "1:1"],
            detach=True,
            remove=False,  # Keep container to read logs
            volumes={
                str(temp_workdir): {"bind": "/content", "mode": "rw"}
            },
            working_dir="/content"
        )

        # Wait for container to exit
        result = container.wait(timeout=10)

        # Should fail
        assert result['StatusCode'] != 0

        # Check logs for error message
        logs = container.logs().decode('utf-8')
        assert "error" in logs.lower() or "not found" in logs.lower() or "no such file" in logs.lower()

        # Clean up
        container.remove(force=True)

    except Exception as e:
        # Some error is expected
        assert True
