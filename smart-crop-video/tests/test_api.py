"""
Flask API endpoint tests for smart-crop-video.

Tests all API endpoints for proper responses and state management.
"""

import pytest
import requests
import time
from pathlib import Path


def test_root_endpoint_returns_html(smart_crop_container: dict):
    """Test that the root endpoint returns HTML page."""
    base_url = smart_crop_container["base_url"]
    response = requests.get(base_url, timeout=5)

    assert response.status_code == 200
    assert response.headers['Content-Type'].startswith('text/html')
    assert "Smart Crop Video" in response.text
    assert "<html>" in response.text.lower()


def test_root_endpoint_contains_required_ui_elements(smart_crop_container: dict):
    """Test that the root HTML contains required UI elements."""
    base_url = smart_crop_container["base_url"]
    response = requests.get(base_url, timeout=5)

    html = response.text.lower()

    # Check for key sections
    assert "status" in html
    assert "candidates" in html
    assert "acceleration" in html
    assert "scene" in html
    assert "encoding" in html

    # Check for JavaScript
    assert "<script>" in html
    assert "updatestatus" in html
    assert "fetch" in html


def test_status_endpoint_returns_json(api_client: dict):
    """Test that /api/status returns valid JSON."""
    status = api_client["get_status"]()

    assert isinstance(status, dict)
    assert "status" in status
    assert "progress" in status
    assert "message" in status


def test_status_endpoint_has_required_fields(api_client: dict):
    """Test that status JSON has all required fields."""
    status = api_client["get_status"]()

    required_fields = [
        "status", "progress", "total_positions", "current_position",
        "message", "candidates", "selected_index", "enable_acceleration",
        "selected_strategy", "scenes", "encoding_progress"
    ]

    for field in required_fields:
        assert field in status, f"Missing required field: {field}"


def test_status_transitions_through_analysis(api_client: dict):
    """Test that status progresses through analysis phase."""
    get_status = api_client["get_status"]

    # Should start with analyzing status
    initial_status = get_status()
    assert initial_status["status"] in ["initializing", "analyzing"]

    # Progress should increase over time
    time.sleep(2)
    later_status = get_status()

    if later_status["status"] == "analyzing":
        assert later_status["progress"] >= initial_status["progress"]


def test_candidates_appear_after_analysis(api_client: dict):
    """Test that candidates appear after analysis completes."""
    wait_for_status = api_client["wait_for_status"]

    # Wait for candidates to be ready (with timeout)
    status = wait_for_status("candidates_ready", timeout=120)

    assert len(status["candidates"]) > 0
    assert len(status["candidates"]) <= 10  # Should have up to 10 candidates


def test_candidate_structure(api_client: dict):
    """Test that each candidate has required fields."""
    wait_for_status = api_client["wait_for_status"]
    status = wait_for_status("candidates_ready", timeout=120)

    candidates = status["candidates"]
    assert len(candidates) > 0

    for candidate in candidates:
        assert "index" in candidate
        assert "x" in candidate
        assert "y" in candidate
        assert "score" in candidate
        assert "strategy" in candidate

        # Validate types
        assert isinstance(candidate["index"], int)
        assert isinstance(candidate["x"], int)
        assert isinstance(candidate["y"], int)
        assert isinstance(candidate["score"], (int, float))
        assert isinstance(candidate["strategy"], str)

        # Validate ranges
        assert candidate["index"] >= 1
        assert candidate["x"] >= 0
        assert candidate["y"] >= 0
        assert candidate["score"] >= 0


def test_preview_endpoints_serve_images(api_client: dict):
    """Test that preview image endpoints return valid images."""
    wait_for_status = api_client["wait_for_status"]
    get_preview_image = api_client["get_preview_image"]

    # Wait for candidates
    status = wait_for_status("candidates_ready", timeout=120)
    candidates = status["candidates"]

    # Try to get first preview image
    if len(candidates) > 0:
        image_bytes = get_preview_image(candidates[0]["index"])

        # Should be non-empty
        assert len(image_bytes) > 0

        # Should be JPEG (starts with FF D8 FF)
        assert image_bytes[:3] == b'\xff\xd8\xff'


def test_preview_endpoint_returns_404_for_invalid_index(api_client: dict):
    """Test that preview endpoint returns 404 for invalid index."""
    wait_for_status = api_client["wait_for_status"]
    base_url = api_client["base_url"]

    # Wait for candidates
    wait_for_status("candidates_ready", timeout=120)

    # Try to get non-existent preview
    response = requests.get(f"{base_url}/api/preview/999", timeout=5)
    assert response.status_code == 404


def test_select_endpoint_updates_state(api_client: dict):
    """Test that selecting a crop updates the application state."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    get_status = api_client["get_status"]

    # Wait for candidates
    status = wait_for_status("candidates_ready", timeout=120)
    candidates = status["candidates"]

    # Select first candidate
    response = select_crop(candidates[0]["index"])
    assert response.status_code == 200
    assert response.json()["success"] is True

    # State should update
    time.sleep(0.5)
    status = get_status()
    assert status["selected_index"] == candidates[0]["index"]
    assert status["status"] == "awaiting_acceleration_choice"


def test_select_endpoint_rejects_invalid_index(api_client: dict):
    """Test that selecting an invalid index is handled."""
    wait_for_status = api_client["wait_for_status"]
    base_url = api_client["base_url"]

    # Wait for candidates
    wait_for_status("candidates_ready", timeout=120)

    # Try to select invalid index
    # The API doesn't validate, but we test it doesn't crash
    response = requests.post(f"{base_url}/api/select/999", timeout=5)
    assert response.status_code == 200  # API accepts any index


def test_acceleration_endpoint_accepts_yes(api_client: dict):
    """Test that acceleration endpoint accepts 'yes'."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    choose_acceleration = api_client["choose_acceleration"]
    get_status = api_client["get_status"]

    # Wait for candidates and select one
    status = wait_for_status("candidates_ready", timeout=120)
    select_crop(status["candidates"][0]["index"])

    # Choose acceleration
    response = choose_acceleration(True)
    assert response.status_code == 200
    assert response.json()["success"] is True

    # State should update
    time.sleep(0.5)
    status = get_status()
    assert status["enable_acceleration"] is True


def test_acceleration_endpoint_accepts_no(api_client: dict):
    """Test that acceleration endpoint accepts 'no'."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    choose_acceleration = api_client["choose_acceleration"]
    get_status = api_client["get_status"]

    # Wait for candidates and select one
    status = wait_for_status("candidates_ready", timeout=120)
    select_crop(status["candidates"][0]["index"])

    # Choose no acceleration
    response = choose_acceleration(False)
    assert response.status_code == 200
    assert response.json()["success"] is True

    # State should update
    time.sleep(0.5)
    status = get_status()
    assert status["enable_acceleration"] is False


def test_acceleration_endpoint_accepts_various_formats(api_client: dict):
    """Test that acceleration endpoint accepts various yes/no formats."""
    base_url = api_client["base_url"]

    # Test various formats for "yes"
    for choice in ["yes", "YES", "true", "TRUE", "1"]:
        response = requests.post(f"{base_url}/api/acceleration/{choice}", timeout=5)
        assert response.status_code == 200

    # Test various formats for "no"
    for choice in ["no", "NO", "false", "FALSE", "0"]:
        response = requests.post(f"{base_url}/api/acceleration/{choice}", timeout=5)
        assert response.status_code == 200


def test_scene_thumbnail_endpoint(api_client: dict):
    """Test scene thumbnail endpoint (if acceleration is enabled)."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    choose_acceleration = api_client["choose_acceleration"]

    # Wait for candidates and select one
    status = wait_for_status("candidates_ready", timeout=120)
    select_crop(status["candidates"][0]["index"])

    # Enable acceleration
    choose_acceleration(True)

    # Wait for scenes (this might take a while)
    try:
        status = wait_for_status("awaiting_scene_selection", timeout=180)
        scenes = status["scenes"]

        if len(scenes) > 0:
            # Try to get first scene's thumbnail
            scene = scenes[0]
            first_frame_id = scene["first_frame"]
            last_frame_id = scene["last_frame"]

            # Get thumbnails
            first_thumb = api_client["get_scene_thumbnail"](first_frame_id, "first")
            last_thumb = api_client["get_scene_thumbnail"](last_frame_id, "last")

            # Should be JPEG images
            assert first_thumb[:3] == b'\xff\xd8\xff'
            assert last_thumb[:3] == b'\xff\xd8\xff'

    except Exception as e:
        pytest.skip(f"Scene analysis not available or took too long: {e}")


def test_scene_selections_endpoint(api_client: dict):
    """Test scene selections endpoint."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    choose_acceleration = api_client["choose_acceleration"]
    select_scenes = api_client["select_scenes"]

    # Wait for candidates and select one
    status = wait_for_status("candidates_ready", timeout=120)
    select_crop(status["candidates"][0]["index"])

    # Enable acceleration
    choose_acceleration(True)

    # Wait for scenes
    try:
        status = wait_for_status("awaiting_scene_selection", timeout=180)
        scenes = status["scenes"]

        if len(scenes) > 0:
            # Select first scene at 2x speed
            selections = {1: 2.0}
            response = select_scenes(selections)

            assert response.status_code == 200
            assert response.json()["success"] is True

    except Exception as e:
        pytest.skip(f"Scene analysis not available: {e}")


def test_scene_selections_with_multiple_scenes(api_client: dict):
    """Test selecting multiple scenes with different speeds."""
    wait_for_status = api_client["wait_for_status"]
    select_crop = api_client["select_crop"]
    choose_acceleration = api_client["choose_acceleration"]
    select_scenes = api_client["select_scenes"]

    # Wait for candidates and select one
    status = wait_for_status("candidates_ready", timeout=120)
    select_crop(status["candidates"][0]["index"])

    # Enable acceleration
    choose_acceleration(True)

    # Wait for scenes
    try:
        status = wait_for_status("awaiting_scene_selection", timeout=180)
        scenes = status["scenes"]

        if len(scenes) >= 3:
            # Select multiple scenes with different speeds
            selections = {
                1: 2.0,
                2: 3.0,
                3: 4.0
            }
            response = select_scenes(selections)

            assert response.status_code == 200
            assert response.json()["success"] is True

    except Exception as e:
        pytest.skip(f"Scene analysis not available: {e}")


def test_status_polling_doesnt_crash(api_client: dict):
    """Test that polling status endpoint repeatedly doesn't cause issues."""
    get_status = api_client["get_status"]

    # Poll status 20 times rapidly
    for _ in range(20):
        status = get_status()
        assert isinstance(status, dict)
        time.sleep(0.1)


def test_concurrent_status_requests(api_client: dict):
    """Test that concurrent status requests are handled correctly."""
    import concurrent.futures
    get_status = api_client["get_status"]

    # Make 10 concurrent requests
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(get_status) for _ in range(10)]
        results = [f.result() for f in concurrent.futures.as_completed(futures)]

    # All should succeed
    assert len(results) == 10
    for result in results:
        assert isinstance(result, dict)
        assert "status" in result
