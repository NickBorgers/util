"""API client helper utilities for smart-crop-video tests."""

import pytest
import requests
import time
from typing import Dict, Any


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
