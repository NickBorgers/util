#!/usr/bin/env python3
"""
Intelligently crops videos to specified aspect ratios by analyzing visual activity
and motion to find the most interesting region.

Uses multiple scoring strategies to generate candidate crops for user selection.
Provides both web UI and text interface for selection.
"""

import sys
import os
import re
import subprocess
import argparse
import threading
import socket
import time
from pathlib import Path
from typing import List, Tuple, Dict, Optional
from dataclasses import dataclass
from flask import Flask, jsonify, send_file, request
import logging

# Import refactored modules (Phase 6.1: Parallelization)
from smart_crop.core.grid import Position, generate_analysis_grid
from smart_crop.analysis.parallel import analyze_positions_parallel

# Import refactored modules (Phase 6.2: Scene Detection)
from smart_crop.analysis.scenes import (
    Scene,
    parse_scene_timestamps,
    create_scenes_from_timestamps,
    create_time_based_segments as create_time_segments
)

# Import refactored modules (Phase 6.3: Scoring)
from smart_crop.core.scoring import (
    normalize,
    score_position,
    PositionMetrics,
    NormalizationBounds
)

# Import refactored modules (Phase 7A: Remove Duplicates)
from smart_crop.core.candidates import ScoredCandidate

# Disable Flask development server warning
log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

# Phase 7A: Removed duplicate CropPosition class - using PositionMetrics from smart_crop.core.scoring
# Phase 7A: Removed duplicate ScoredCandidate class - using import from smart_crop.core.candidates

class AppState:
    """Shared state between main thread and webserver"""
    def __init__(self):
        self.status = "initializing"
        self.progress = 0
        self.total_positions = 0
        self.current_position = 0
        self.message = "Starting analysis..."
        self.candidates = []
        self.preview_dir = "."
        self.base_name = ""
        self.selected_index = None
        self.enable_acceleration = None
        self.selected_strategy = "Balanced"
        self.scenes = []
        self.scene_selections = None  # Dict mapping scene index to speedup factor
        self.encoding_progress = 0
        self.lock = threading.Lock()

    def update(self, **kwargs):
        """Thread-safe update of state"""
        with self.lock:
            for key, value in kwargs.items():
                setattr(self, key, value)

    def get(self, key):
        """Thread-safe get of state value"""
        with self.lock:
            return getattr(self, key)

    def get_dict(self):
        """Get a thread-safe copy of state as dict"""
        with self.lock:
            return {
                'status': self.status,
                'progress': self.progress,
                'total_positions': self.total_positions,
                'current_position': self.current_position,
                'message': self.message,
                'candidates': [
                    {
                        'index': i + 1,
                        'x': c.x,
                        'y': c.y,
                        'score': c.score,
                        'strategy': c.strategy
                    }
                    for i, c in enumerate(self.candidates)
                ],
                'selected_index': self.selected_index,
                'enable_acceleration': self.enable_acceleration,
                'selected_strategy': self.selected_strategy,
                'scenes': [
                    {
                        'index': i + 1,
                        'start_time': s.start_time,
                        'end_time': s.end_time,
                        'duration': s.duration,
                        'metric_value': s.metric_value,
                        'first_frame': f'scene_{i+1}_first',
                        'last_frame': f'scene_{i+1}_last'
                    }
                    for i, s in enumerate(self.scenes)
                ],
                'encoding_progress': self.encoding_progress
            }


def find_free_port():
    """Find a free port for the webserver"""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(('', 0))
        s.listen(1)
        port = s.getsockname()[1]
    return port


def create_app(state: AppState) -> Flask:
    """Create Flask app with routes"""
    app = Flask(__name__)

    @app.route('/')
    def index():
        """Main UI page"""
        html = """
<!DOCTYPE html>
<html>
<head>
    <title>Smart Crop Video</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
            background: #1a1a1a;
            color: #e0e0e0;
        }
        h1 {
            color: #4CAF50;
            border-bottom: 2px solid #4CAF50;
            padding-bottom: 10px;
        }
        .status {
            background: #2a2a2a;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
            border-left: 4px solid #4CAF50;
        }
        .progress-bar {
            width: 100%;
            height: 30px;
            background: #333;
            border-radius: 15px;
            overflow: hidden;
            margin: 10px 0;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #4CAF50, #45a049);
            transition: width 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
        }
        .candidates {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .candidate {
            background: #2a2a2a;
            border: 2px solid #444;
            border-radius: 8px;
            padding: 15px;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        .candidate:hover {
            border-color: #4CAF50;
            transform: translateY(-5px);
            box-shadow: 0 5px 15px rgba(76, 175, 80, 0.3);
        }
        .candidate.selected {
            border-color: #4CAF50;
            background: #1e3a1e;
        }
        .candidate img {
            width: 100%;
            border-radius: 4px;
            margin-bottom: 10px;
        }
        .candidate-info {
            font-size: 14px;
        }
        .strategy {
            color: #4CAF50;
            font-weight: bold;
        }
        .score {
            color: #888;
        }
        button {
            background: #4CAF50;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
            margin-top: 10px;
        }
        button:hover {
            background: #45a049;
        }
        button:disabled {
            background: #666;
            cursor: not-allowed;
        }
        .encoding {
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <h1>🎬 Smart Crop Video</h1>

    <div class="status">
        <h2 id="status-title">Status</h2>
        <div id="status-message">Loading...</div>
        <div class="progress-bar">
            <div class="progress-fill" id="progress-bar" style="width: 0%">0%</div>
        </div>
    </div>

    <div id="candidates-section" style="display: none;">
        <h2>Select Your Preferred Crop</h2>
        <p>Click on an option to select it, then click "Confirm Selection"</p>
        <div class="candidates" id="candidates"></div>
        <button id="confirm-btn" disabled onclick="confirmSelection()">Confirm Selection</button>
    </div>

    <div id="acceleration-section" style="display: none; margin-top: 20px;">
        <div class="status">
            <h2>⚡ Intelligent Acceleration</h2>
            <p>Automatically speed up boring sections of your video (2x-4x) while keeping interesting parts at normal speed.</p>
            <p style="margin-top: 10px;">Analysis uses the <strong id="strategy-name">Balanced</strong> strategy you selected.</p>
            <label style="display: flex; align-items: center; margin-top: 15px; cursor: pointer;">
                <input type="checkbox" id="acceleration-checkbox" style="margin-right: 10px; width: 20px; height: 20px;">
                <span style="font-size: 16px;">Enable intelligent acceleration of boring sections</span>
            </label>
            <button onclick="proceedToSceneAnalysis()" style="margin-top: 15px;">Continue</button>
        </div>
    </div>

    <div id="acceleration-progress-section" class="encoding" style="display: none;">
        <h2>Analyzing Temporal Patterns</h2>
        <div id="acceleration-details" style="margin-bottom: 10px; color: #888;"></div>
        <div class="progress-bar">
            <div class="progress-fill" id="acceleration-bar" style="width: 0%">0%</div>
        </div>
    </div>

    <div id="scene-selection-section" style="display: none; margin-top: 20px;">
        <h2>🎬 Select Scenes to Accelerate</h2>
        <p>Choose which sections of your video to speed up and how fast. Unselected scenes will play at normal speed.</p>
        <div id="scenes-container" class="candidates"></div>
        <div style="margin-top: 20px; display: flex; gap: 10px;">
            <button onclick="selectAllScenes()">Select All</button>
            <button onclick="clearAllScenes()">Clear All</button>
            <button onclick="confirmSceneSelections()" id="confirm-scenes-btn" style="margin-left: auto;">Proceed to Encoding</button>
        </div>
    </div>

    <div id="encoding-section" class="encoding" style="display: none;">
        <h2>Encoding Video</h2>
        <div class="progress-bar">
            <div class="progress-fill" id="encoding-bar" style="width: 0%">0%</div>
        </div>
    </div>

    <script>
        let selectedIndex = null;
        let accelerationChoice = false;
        let sceneSelections = {};  // Map scene index to speedup factor

        function updateStatus() {
            fetch('/api/status')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('status-message').textContent = data.message;
                    document.getElementById('progress-bar').style.width = data.progress + '%';
                    document.getElementById('progress-bar').textContent = data.progress + '%';

                    if (data.status === 'candidates_ready' && data.candidates.length > 0) {
                        showCandidates(data.candidates);
                    }

                    if (data.status === 'awaiting_acceleration_choice') {
                        document.getElementById('acceleration-section').style.display = 'block';
                        document.getElementById('strategy-name').textContent = data.selected_strategy;
                    }

                    if (data.status === 'analyzing_temporal') {
                        document.getElementById('acceleration-progress-section').style.display = 'block';
                        document.getElementById('acceleration-details').textContent = data.message;
                        // Use progress bar for temporal analysis if available
                        if (data.progress > 0) {
                            document.getElementById('acceleration-bar').style.width = data.progress + '%';
                            document.getElementById('acceleration-bar').textContent = data.progress + '%';
                        }
                    }

                    if (data.status === 'awaiting_scene_selection' && data.scenes && data.scenes.length > 0) {
                        showScenes(data.scenes);
                    }

                    if (data.status === 'encoding' || data.selected_index !== null) {
                        document.getElementById('encoding-section').style.display = 'block';
                        document.getElementById('encoding-bar').style.width = data.encoding_progress + '%';
                        document.getElementById('encoding-bar').textContent = data.encoding_progress + '%';
                    }

                    if (data.status !== 'complete') {
                        setTimeout(updateStatus, 500);
                    } else {
                        document.getElementById('status-message').textContent = 'Complete! You can close this window.';
                        document.getElementById('progress-bar').style.width = '100%';
                        document.getElementById('progress-bar').textContent = '100%';
                    }
                });
        }

        function showCandidates(candidates) {
            const section = document.getElementById('candidates-section');
            const container = document.getElementById('candidates');

            if (container.children.length === 0) {
                section.style.display = 'block';

                candidates.forEach(c => {
                    const div = document.createElement('div');
                    div.className = 'candidate';
                    div.onclick = () => selectCandidate(c.index);
                    div.innerHTML = `
                        <img src="/api/preview/${c.index}" alt="Crop option ${c.index}">
                        <div class="candidate-info">
                            <div><strong>Option ${c.index}</strong></div>
                            <div class="strategy">${c.strategy}</div>
                            <div class="score">Score: ${c.score.toFixed(2)}</div>
                            <div style="font-size: 12px; color: #888;">Position: (${c.x}, ${c.y})</div>
                        </div>
                    `;
                    container.appendChild(div);
                });
            }
        }

        function selectCandidate(index) {
            selectedIndex = index;
            document.querySelectorAll('.candidate').forEach((el, i) => {
                el.classList.toggle('selected', i === index - 1);
            });
            document.getElementById('confirm-btn').disabled = false;
        }

        function confirmSelection() {
            if (selectedIndex !== null) {
                fetch(`/api/select/${selectedIndex}`, {method: 'POST'})
                    .then(() => {
                        document.getElementById('candidates-section').style.display = 'none';
                        document.getElementById('status-message').textContent =
                            `Selected option ${selectedIndex}. Choose acceleration settings...`;
                    });
            }
        }

        function proceedToSceneAnalysis() {
            accelerationChoice = document.getElementById('acceleration-checkbox').checked;
            fetch(`/api/acceleration/${accelerationChoice ? 'yes' : 'no'}`, {method: 'POST'})
                .then(() => {
                    document.getElementById('acceleration-section').style.display = 'none';
                    if (accelerationChoice) {
                        document.getElementById('status-message').textContent = 'Analyzing temporal patterns...';
                    } else {
                        document.getElementById('status-message').textContent = 'Encoding video...';
                    }
                });
        }

        function showScenes(scenes) {
            const section = document.getElementById('scene-selection-section');
            const container = document.getElementById('scenes-container');

            if (container.children.length === 0) {
                section.style.display = 'block';
                document.getElementById('acceleration-progress-section').style.display = 'none';

                scenes.forEach(scene => {
                    const div = document.createElement('div');
                    div.className = 'candidate';
                    div.style.position = 'relative';
                    div.innerHTML = `
                        <div style="display: flex; gap: 10px; margin-bottom: 10px;">
                            <img src="/api/scene_thumbnail/${scene.first_frame}/first" style="width: 48%; border-radius: 4px;" alt="Scene ${scene.index} start">
                            <img src="/api/scene_thumbnail/${scene.last_frame}/last" style="width: 48%; border-radius: 4px;" alt="Scene ${scene.index} end">
                        </div>
                        <div style="font-size: 14px;">
                            <div><strong>Scene ${scene.index}</strong></div>
                            <div style="color: #888;">Duration: ${scene.duration.toFixed(1)}s (${scene.start_time.toFixed(1)}s - ${scene.end_time.toFixed(1)}s)</div>
                            <label style="display: flex; align-items: center; margin-top: 8px; cursor: pointer;">
                                <input type="checkbox"
                                       class="scene-checkbox"
                                       data-scene="${scene.index}"
                                       onchange="toggleScene(${scene.index})"
                                       style="margin-right: 8px; width: 18px; height: 18px;">
                                <span>Accelerate this scene</span>
                            </label>
                            <div class="speedup-controls" id="speedup-${scene.index}" style="display: none; margin-top: 8px;">
                                <label style="font-size: 12px; color: #888;">Speed:
                                    <select onchange="updateSceneSpeed(${scene.index}, this.value)"
                                            style="margin-left: 5px; background: #333; color: #e0e0e0; border: 1px solid #666; padding: 2px;">
                                        <option value="2">2x</option>
                                        <option value="3">3x</option>
                                        <option value="4">4x</option>
                                    </select>
                                </label>
                            </div>
                        </div>
                    `;
                    container.appendChild(div);
                });
            }
        }

        function toggleScene(sceneIndex) {
            const checkbox = document.querySelector(`input[data-scene="${sceneIndex}"]`);
            const speedupControls = document.getElementById(`speedup-${sceneIndex}`);

            if (checkbox.checked) {
                speedupControls.style.display = 'block';
                sceneSelections[sceneIndex] = 2.0;  // Default to 2x
            } else {
                speedupControls.style.display = 'none';
                delete sceneSelections[sceneIndex];
            }
        }

        function updateSceneSpeed(sceneIndex, speed) {
            sceneSelections[sceneIndex] = parseFloat(speed);
        }

        function selectAllScenes() {
            document.querySelectorAll('.scene-checkbox').forEach(cb => {
                if (!cb.checked) {
                    cb.checked = true;
                    toggleScene(parseInt(cb.dataset.scene));
                }
            });
        }

        function clearAllScenes() {
            document.querySelectorAll('.scene-checkbox').forEach(cb => {
                cb.checked = false;
                toggleScene(parseInt(cb.dataset.scene));
            });
            sceneSelections = {};
        }

        function confirmSceneSelections() {
            fetch('/api/scene_selections', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({selections: sceneSelections})
            }).then(() => {
                document.getElementById('scene-selection-section').style.display = 'none';
                document.getElementById('status-message').textContent = 'Encoding video with selected scene speeds...';
            });
        }

        updateStatus();
    </script>
</body>
</html>
        """
        return html

    @app.route('/api/status')
    def api_status():
        """Status API endpoint"""
        return jsonify(state.get_dict())

    @app.route('/api/preview/<int:index>')
    def api_preview(index):
        """Serve preview image"""
        preview_file = f"{state.base_name}_crop_option_{index}.jpg"
        preview_path = Path(state.preview_dir) / preview_file
        if preview_path.exists():
            return send_file(preview_path, mimetype='image/jpeg')
        return "Not found", 404

    @app.route('/api/select/<int:index>', methods=['POST'])
    def api_select(index):
        """Handle selection from web UI"""
        state.update(selected_index=index, status='awaiting_acceleration_choice')
        return jsonify({'success': True})

    @app.route('/api/acceleration/<choice>', methods=['POST'])
    def api_acceleration(choice):
        """Handle acceleration choice from web UI"""
        enable = choice.lower() in ('yes', 'true', '1')
        state.update(enable_acceleration=enable)
        return jsonify({'success': True})

    @app.route('/api/scene_thumbnail/<scene_id>/<frame_type>')
    def api_scene_thumbnail(scene_id, frame_type):
        """Serve scene thumbnail (first or last frame)"""
        # scene_id format: "scene_1_first" or "scene_1_last"
        thumbnail_file = f".{state.base_name}_{scene_id}.jpg"
        thumbnail_path = Path(state.preview_dir) / thumbnail_file
        if thumbnail_path.exists():
            return send_file(thumbnail_path, mimetype='image/jpeg')
        return "Not found", 404

    @app.route('/api/scene_selections', methods=['POST'])
    def api_scene_selections():
        """Handle scene selection and speedup choices from web UI"""
        data = request.get_json()
        # data format: {'selections': {scene_index: speedup_factor, ...}}
        selections = data.get('selections', {})
        # Convert string keys to ints
        scene_selections = {int(k): float(v) for k, v in selections.items()}
        state.update(scene_selections=scene_selections)
        return jsonify({'success': True})

    return app


def run_flask_server(app: Flask, port: int):
    """Run Flask server in thread"""
    app.run(host='0.0.0.0', port=port, debug=False, use_reloader=False)


def run_ffmpeg(cmd: List[str]) -> str:
    """Run ffmpeg command and return stderr output"""
    result = subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    return result.stderr


def get_video_dimensions(input_file: str) -> Tuple[int, int]:
    """Get video width and height"""
    cmd = [
        'ffprobe', '-v', 'error',
        '-select_streams', 'v:0',
        '-show_entries', 'stream=width,height',
        '-of', 'csv=p=0',
        input_file
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    width, height = map(int, result.stdout.strip().split(','))
    return width, height


def get_video_duration(input_file: str) -> float:
    """Get video duration in seconds"""
    cmd = [
        'ffprobe', '-v', 'error',
        '-show_entries', 'format=duration',
        '-of', 'default=noprint_wrappers=1:nokey=1',
        input_file
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return float(result.stdout.strip())


def get_video_frame_count(input_file: str) -> int:
    """Estimate total number of frames in video (fast method using duration and fps)"""
    # Get FPS
    cmd_fps = [
        'ffprobe', '-v', 'error',
        '-select_streams', 'v:0',
        '-show_entries', 'stream=r_frame_rate',
        '-of', 'csv=p=0',
        input_file
    ]
    result_fps = subprocess.run(cmd_fps, capture_output=True, text=True)
    fps_str = result_fps.stdout.strip()

    try:
        if '/' in fps_str:
            num, den = map(int, fps_str.split('/'))
            fps = num / den
        else:
            fps = float(fps_str)
    except (ValueError, ZeroDivisionError):
        fps = 24.0  # Default fallback

    # Get duration
    duration = get_video_duration(input_file)

    # Estimate frame count
    estimated_frames = int(duration * fps)
    return max(estimated_frames, 1)  # Ensure at least 1


def get_video_fps(input_file: str) -> float:
    """Get video frame rate"""
    cmd = [
        'ffprobe', '-v', 'error',
        '-select_streams', 'v:0',
        '-show_entries', 'stream=r_frame_rate',
        '-of', 'csv=p=0',
        input_file
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    fps_str = result.stdout.strip()

    try:
        if '/' in fps_str:
            num, den = map(int, fps_str.split('/'))
            return num / den
        else:
            return float(fps_str)
    except (ValueError, ZeroDivisionError):
        return 24.0  # Default fallback


# Phase 7A: Removed duplicate Scene class - using import from smart_crop.analysis.scenes

def detect_scenes(input_file: str, threshold: float = 0.3) -> List[Scene]:
    """Detect scene changes in video using FFmpeg's scene detection"""
    # Use FFmpeg to detect scenes
    cmd = [
        'ffmpeg', '-i', input_file,
        '-vf', f'select=gt(scene\\,{threshold}),showinfo',
        '-f', 'null', '-'
    ]

    result = subprocess.run(cmd, capture_output=True, text=True, check=False)

    # Parse scene changes from showinfo output (Phase 6.2: Use refactored module)
    scene_changes = parse_scene_timestamps(result.stderr)

    # Create Scene objects (Phase 6.2: Use refactored module)
    duration = get_video_duration(input_file)
    total_frames = int(get_video_frame_count(input_file))
    scenes = create_scenes_from_timestamps(scene_changes, duration, total_frames)

    return scenes


def create_time_based_segments(input_file: str, segment_duration: float = 5.0) -> List[Scene]:
    """Create fixed-duration segments when scene detection doesn't find enough scenes

    Args:
        input_file: Path to video file
        segment_duration: Duration of each segment in seconds (default: 5.0)

    Returns:
        List of Scene objects representing time-based segments
    """
    # Phase 6.2: Use refactored module function
    duration = get_video_duration(input_file)
    fps = get_video_fps(input_file)

    return create_time_segments(duration, fps, segment_duration)


def extract_scene_thumbnails(input_file: str, scenes: List[Scene], x: int, y: int,
                            crop_w: int, crop_h: int, base_name: str, state: AppState,
                            progress_offset: int = 0) -> None:
    """Extract first and last frame thumbnails for each scene

    Args:
        progress_offset: Starting progress percentage (e.g., 40 means progress goes from 40-100%)
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
        # Scale progress from offset to 100
        extraction_progress = (current / total_extractions) * progress_range
        progress_pct = int(progress_offset + extraction_progress)
        progress_msg = f"Extracting thumbnails for scene {i+1}/{len(scenes)} (first frame)"
        print(f"\r[{current:3d}/{total_extractions}] {progress_msg}...  ", end='', flush=True)
        state.update(progress=progress_pct, message=progress_msg)

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
        state.update(progress=progress_pct, message=progress_msg)

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


def analyze_scene_metrics(input_file: str, scene: Scene, x: int, y: int,
                          crop_w: int, crop_h: int, metric_type: str,
                          sample_frames: int = 10) -> float:
    """Analyze a specific metric for a scene"""
    # Sample frames from the scene
    duration = scene.duration
    if duration < 0.1:  # Scene too short
        return 0.0

    # Calculate frame sampling interval
    total_scene_frames = int((scene.end_frame - scene.start_frame))
    if total_scene_frames < 1:
        return 0.0

    # Sample up to N frames evenly distributed across the scene
    sample_count = min(sample_frames, max(1, total_scene_frames))

    # Get metrics based on type
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
        # Analyze visual complexity
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


def determine_primary_metric(strategy: str) -> str:
    """Determine which metric is most important for a strategy"""
    strategy_metrics = {
        'Subject Detection': 'edges',
        'Motion Priority': 'motion',
        'Visual Detail': 'complexity',
        'Balanced': 'motion',  # Default to motion
        'Color Focus': 'edges',  # Use edges as proxy
    }

    # Handle spatial strategies
    if 'Spatial:' in strategy:
        return 'motion'  # Default for spatial

    return strategy_metrics.get(strategy, 'motion')


def identify_boring_sections(scenes: List[Scene], percentile_threshold: float = 30.0) -> List[tuple]:
    """Identify boring sections based on metric values

    Returns list of (scene_index, speedup_factor) tuples
    """
    if not scenes:
        return []

    # Calculate threshold (30th percentile by default)
    metric_values = [s.metric_value for s in scenes]
    metric_values.sort()
    threshold_idx = int(len(metric_values) * (percentile_threshold / 100.0))
    threshold = metric_values[threshold_idx] if threshold_idx < len(metric_values) else metric_values[0]

    # Identify boring sections
    boring_sections = []
    for i, scene in enumerate(scenes):
        if scene.metric_value < threshold:
            # Boring section - calculate speedup factor based on how boring
            # Very boring (near 0) = 4x speed
            # Slightly boring (near threshold) = 2x speed
            if threshold > 0:
                ratio = scene.metric_value / threshold
                speedup = 2.0 + (2.0 * (1.0 - ratio))  # Range: 2x to 4x
            else:
                speedup = 3.0
            boring_sections.append((i, min(speedup, 4.0)))  # Cap at 4x

    return boring_sections


def encode_with_variable_speed(input_file: str, output_file: str, crop_w: int, crop_h: int,
                                crop_x: int, crop_y: int, scenes: List[Scene],
                                boring_sections: List[tuple], preset: str,
                                state: AppState) -> None:
    """Encode video with variable speed for boring sections

    Uses segment-based approach: split, speed up boring parts, concat
    """
    import tempfile

    # Create temporary directory for segments
    temp_dir = tempfile.mkdtemp(prefix='smart_crop_')

    try:
        # Create speedup map
        speedup_map = {idx: factor for idx, factor in boring_sections}

        # Generate segments
        segment_files = []
        print("Encoding segments with variable speed...")
        print()

        for i, scene in enumerate(scenes):
            speedup = speedup_map.get(i, 1.0)
            segment_file = os.path.join(temp_dir, f'segment_{i:04d}.mp4')
            segment_files.append(segment_file)

            status = f"normal" if speedup == 1.0 else f"{speedup:.1f}x"
            print(f"\r[{i+1}/{len(scenes)}] Scene {i+1}: {scene.start_time:.1f}s-{scene.end_time:.1f}s ({status})  ",
                  end='', flush=True)

            # Build filter for this segment
            if speedup == 1.0:
                # Normal speed - just crop
                vf = f'crop={crop_w}:{crop_h}:{crop_x}:{crop_y}'
            else:
                # Speed up - crop and adjust speed
                vf = f'crop={crop_w}:{crop_h}:{crop_x}:{crop_y},setpts=PTS/{speedup}'

            # Extract and encode this segment
            cmd = [
                'ffmpeg', '-ss', str(scene.start_time), '-t', str(scene.duration),
                '-i', input_file,
                '-vf', vf,
                '-af', f'atempo={min(speedup, 2.0)}' if speedup <= 2.0 else f'atempo=2.0,atempo={speedup/2.0}',
                '-c:v', 'libx264', '-preset', preset, '-crf', '19',
                '-c:a', 'aac',  # Re-encode audio due to atempo
                '-y', segment_file
            ]

            subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

        print(f"\r{' '*80}\r✓ Encoded all {len(scenes)} segments")
        print()

        # Create concat file
        concat_file = os.path.join(temp_dir, 'concat.txt')
        with open(concat_file, 'w') as f:
            for seg_file in segment_files:
                f.write(f"file '{seg_file}'\n")

        # Concatenate segments
        print("Concatenating segments into final video...")
        state.update(status="encoding", message="Concatenating segments...")

        cmd = [
            'ffmpeg', '-f', 'concat', '-safe', '0', '-i', concat_file,
            '-c', 'copy',
            '-y', output_file
        ]

        subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

        print("✓ Final video created")

    finally:
        # Cleanup temp directory
        import shutil
        try:
            shutil.rmtree(temp_dir)
        except:
            pass


def analyze_temporal_patterns(input_file: str, x: int, y: int, crop_w: int, crop_h: int,
                               strategy: str, base_name: str, state: AppState) -> Optional[List[Scene]]:
    """Analyze video temporal patterns and extract scene thumbnails for user selection

    Returns list of scenes with thumbnails, or None if not enough scenes
    """
    state.update(status="analyzing_temporal", progress=0, message="Starting temporal analysis...")
    print("\n" + "="*70)
    print("Temporal Analysis: Detecting scenes and extracting thumbnails")
    print("="*70)
    print()

    # Detect scenes (takes 0-30% of progress)
    print("Step 1: Detecting scene changes...")
    state.update(progress=5, message="Detecting scene changes...")

    scene_threshold = float(os.getenv('SCENE_THRESHOLD', '0.2'))  # Lowered from 0.3
    scenes = detect_scenes(input_file, scene_threshold)

    state.update(progress=30, message=f"Detected {len(scenes)} scenes")
    print(f"✓ Detected {len(scenes)} scenes")
    print()

    # If not enough scenes detected, fall back to time-based segmentation
    min_scenes = 3
    if len(scenes) < min_scenes:
        print(f"Too few scenes detected ({len(scenes)}).")
        print("Falling back to time-based segmentation...")
        state.update(progress=35, message="Creating time-based segments...")

        # Use configurable segment duration (default 5 seconds)
        segment_duration = float(os.getenv('SEGMENT_DURATION', '5.0'))
        scenes = create_time_based_segments(input_file, segment_duration)

        state.update(progress=40, message=f"Created {len(scenes)} time-based segments")
        print(f"✓ Created {len(scenes)} time-based segments ({segment_duration}s each)")
        print()

        if len(scenes) < 2:
            print("Video too short for temporal analysis, skipping speedup")
            return None

    # Extract thumbnails for each scene (takes 40-100% of progress)
    print("Step 2: Extracting scene thumbnails...")
    state.update(progress=40, message="Starting thumbnail extraction...")
    extract_scene_thumbnails(input_file, scenes, x, y, crop_w, crop_h, base_name, state, progress_offset=40)

    # Update state with scenes for web UI
    state.update(scenes=scenes, status='awaiting_scene_selection', progress=100,
                 message=f"Select which of {len(scenes)} scenes to accelerate")

    print()
    print(f"✓ Scenes ready for selection")
    print(f"  Review {len(scenes)} scenes and choose which to accelerate")
    print("="*70)
    print()

    return scenes


def extract_metric_from_showinfo(output: str, metric: str) -> List[float]:
    """Extract metric values from ffmpeg showinfo filter output"""
    pattern = rf'{metric}:\[([0-9. ]+)\]'
    matches = re.findall(pattern, output)
    values = []
    for match in matches:
        # Take first value from each match (for Y channel in YUV)
        parts = match.split()
        if parts:
            values.append(float(parts[0]))
    return values


def analyze_position(input_file: str, x: int, y: int, crop_w: int, crop_h: int,
                    analysis_frames: int) -> PositionMetrics:
    """Analyze a crop position and return its metrics"""

    # Get motion and complexity from showinfo
    cmd = [
        'ffmpeg', '-i', input_file,
        '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},showinfo',
        '-frames:v', str(analysis_frames),
        '-f', 'null', '-'
    ]
    stats_output = run_ffmpeg(cmd)

    # Extract motion (frame-to-frame differences in mean)
    means = extract_metric_from_showinfo(stats_output, 'mean')
    motion = 0.0
    if len(means) > 1:
        diffs = [abs(means[i] - means[i-1]) for i in range(1, len(means))]
        motion = sum(diffs) / len(diffs) if diffs else 0.0

    # Extract complexity (standard deviation)
    stdevs = extract_metric_from_showinfo(stats_output, 'stdev')
    complexity = sum(stdevs) / len(stdevs) if stdevs else 0.0

    # Get edge detection score
    cmd = [
        'ffmpeg', '-i', input_file,
        '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},edgedetect=low=0.3:high=0.4:mode=colormix,showinfo',
        '-frames:v', str(analysis_frames),
        '-f', 'null', '-'
    ]
    edge_output = run_ffmpeg(cmd)
    edge_means = extract_metric_from_showinfo(edge_output, 'mean')
    edges = sum(edge_means) / len(edge_means) if edge_means else 0.0

    # Color variance (sum of RGB channel standard deviations)
    # Re-parse stats_output for stdev of all channels
    stdev_pattern = r'stdev:\[([0-9. ]+)\]'
    stdev_matches = re.findall(stdev_pattern, stats_output)
    color_variance = 0.0
    if stdev_matches:
        rgb_sums = []
        for match in stdev_matches:
            values = list(map(float, match.split()))
            if len(values) >= 3:
                rgb_sums.append(sum(values[:3]))  # R, G, B
        color_variance = sum(rgb_sums) / len(rgb_sums) if rgb_sums else 0.0

    return PositionMetrics(x, y, motion, complexity, edges, color_variance)


# Phase 6.3: normalize() imported from smart_crop.core.scoring


def score_with_strategy(pos: PositionMetrics, mins: Dict[str, float], maxs: Dict[str, float],
                       strategy: str) -> float:
    """Score a position using a specific strategy (Phase 6.3: Use refactored module)"""

    # Phase 7A: No longer need to convert - pos is already PositionMetrics
    metrics = pos

    # Convert mins/maxs dicts to NormalizationBounds
    bounds = NormalizationBounds(
        motion_min=mins['motion'],
        motion_max=maxs['motion'],
        complexity_min=mins['complexity'],
        complexity_max=maxs['complexity'],
        edges_min=mins['edges'],
        edges_max=maxs['edges'],
        saturation_min=mins['saturation'],
        saturation_max=maxs['saturation']
    )

    # Use refactored scoring function
    return score_position(metrics, bounds, strategy)


def main():
    parser = argparse.ArgumentParser(description='Smart crop video to specified aspect ratio')
    parser.add_argument('input', help='Input video file')
    parser.add_argument('output', nargs='?', help='Output video file')
    parser.add_argument('aspect', nargs='?', default='1:1', help='Aspect ratio (e.g., 1:1, 4:5, 9:16)')
    args = parser.parse_args()

    input_file = args.input
    # Handle empty string aspect ratio (happens when shell function passes "$3" but user didn't provide it)
    aspect_ratio = args.aspect if args.aspect else '1:1'

    # Generate default output filename if not provided
    if not args.output:
        input_path = Path(input_file)
        output_file = f"{input_path.stem}.smart_cropped{input_path.suffix}"
    else:
        output_file = args.output

    # Get environment variables
    preset = os.getenv('PRESET', 'medium')
    analysis_frames = int(os.getenv('ANALYSIS_FRAMES', '50'))
    crop_scale = float(os.getenv('CROP_SCALE', '0.75'))

    # Initialize state and Flask app
    state = AppState()
    state.base_name = Path(input_file).stem
    app = create_app(state)

    # Use fixed port for consistency with Docker port mapping
    port = 8765
    server_thread = threading.Thread(target=run_flask_server, args=(app, port), daemon=True)
    server_thread.start()

    # Wait a moment for server to start
    time.sleep(1)

    print("\n" + "="*70)
    print("🌐 Web UI available at: http://localhost:{}".format(port))
    print("="*70)
    print()

    print(f"Analyzing video: {input_file}")
    print(f"Target aspect ratio: {aspect_ratio}")

    # Get video dimensions
    width, height = get_video_dimensions(input_file)
    print(f"Original dimensions: {width}x{height}")
    print(f"Crop scale factor: {crop_scale} (lower = more aggressive cropping)")

    # Parse aspect ratio
    aspect_w, aspect_h = map(int, aspect_ratio.split(':'))

    # Calculate crop dimensions
    if width < height:
        max_crop_w = width
        max_crop_h = width * aspect_h // aspect_w
        if max_crop_h > height:
            max_crop_h = height
            max_crop_w = height * aspect_w // aspect_h
    else:
        max_crop_h = height
        max_crop_w = height * aspect_w // aspect_h
        if max_crop_w > width:
            max_crop_w = width
            max_crop_h = width * aspect_h // aspect_w

    # Apply scale factor
    crop_w = int(max_crop_w * crop_scale)
    crop_h = int(max_crop_h * crop_scale)

    # Ensure even dimensions
    crop_w = crop_w - (crop_w % 2)
    crop_h = crop_h - (crop_h % 2)

    print(f"Max crop dimensions: {max_crop_w}x{max_crop_h}")
    print(f"Actual crop dimensions: {crop_w}x{crop_h} ({crop_scale}x scale)")

    # Calculate movement range
    max_x = width - crop_w
    max_y = height - crop_h

    # Initialize selected_strategy (will be overridden if position analysis is performed)
    selected_strategy = 'Balanced'

    if max_x <= 0 and max_y <= 0:
        print("Crop dimensions match video, no position analysis needed")
        crop_x, crop_y = 0, 0
        # Update state for web UI
        state.update(selected_strategy=selected_strategy, status='awaiting_acceleration_choice')
    else:
        state.update(status="analyzing", total_positions=25, message="Analyzing positions...")

        print("\n" + "="*70)
        print("Analysis Plan:")
        print("="*70)
        print(f"• Grid: 5×5 (25 positions)")
        print(f"• Frames per position: {analysis_frames}")
        print(f"• Total frame analyses: ~{analysis_frames * 25 * 3} (motion/complexity + strong-edges + saturation)")
        print(f"• Encoding preset: {preset}")
        print("\nThis will take a few moments. Progress will be shown below...")
        print("="*70)
        print()

        # Generate 5x5 grid positions (Phase 6.4: Use refactored module)
        grid_positions = generate_analysis_grid(max_x, max_y, grid_size=5)

        # Analyze all positions (Phase 6.1: Using parallel analysis)
        print("Pass 1: Analyzing all positions...")
        print("Metrics: motion, complexity, strong-edges (high-threshold), color-saturation")
        print(f"This will analyze {len(grid_positions)} positions × 3 passes (motion/complexity + strong-edges + saturation)")
        print()

        total = len(grid_positions)

        # Progress callback for parallel analysis
        def progress_callback(current, total_positions):
            percent = (current * 100) // total_positions
            if current <= len(grid_positions):
                pos = grid_positions[current - 1]
                progress_msg = f"Analyzing position {current}/{total_positions} (x={pos.x}, y={pos.y})"
            else:
                progress_msg = f"Analyzing position {current}/{total_positions}"

            print(f"\r[{percent:3d}%] {progress_msg}...  ", end='', flush=True)

            state.update(
                current_position=current,
                progress=percent,
                message=progress_msg
            )

        # Parallel analysis (4-8x faster than sequential)
        position_metrics = analyze_positions_parallel(
            input_file,
            grid_positions,
            crop_w=crop_w,
            crop_h=crop_h,
            sample_frames=analysis_frames,
            max_workers=None,  # Auto-detect CPU cores
            progress_callback=progress_callback
        )

        # Phase 7A: No longer need conversion - use PositionMetrics directly
        positions = position_metrics

        print(f"\r{' '*80}\r✓ Completed analyzing all {total} positions")
        print()

        # Find min/max for normalization
        mins = {
            'motion': min(p.motion for p in positions),
            'complexity': min(p.complexity for p in positions),
            'edges': min(p.edges for p in positions),
            'saturation': min(p.saturation for p in positions),
        }
        maxs = {
            'motion': max(p.motion for p in positions),
            'complexity': max(p.complexity for p in positions),
            'edges': max(p.edges for p in positions),
            'saturation': max(p.saturation for p in positions),
        }

        print("Pass 2: Generating candidates using 5 different scoring strategies...")
        print()
        print("Strategies:")
        print("  1. Subject Detection - Finds people/objects (40% edges, 30% saturation)")
        print("  2. Motion Priority - Tracks movement (50% motion, 25% edges)")
        print("  3. Visual Detail - Identifies complex areas (50% complexity, 30% edges)")
        print("  4. Balanced - Equal weights (25% each metric)")
        print("  5. Color Focus - Colorful subjects (45% saturation, 30% edges)")
        print()

        # Generate candidates from each strategy
        all_candidates = []
        strategies = ['Subject Detection', 'Motion Priority', 'Visual Detail', 'Balanced', 'Color Focus']

        for strategy in strategies:
            scored = [(p, score_with_strategy(p, mins, maxs, strategy)) for p in positions]
            scored.sort(key=lambda x: x[1], reverse=True)

            for pos, score in scored[:5]:
                candidate = ScoredCandidate(pos.x, pos.y, score, strategy)
                all_candidates.append(candidate)

        # Add spatial diversity (quadrants + center)
        center_x = width // 2
        center_y = height // 2

        # Balanced strategy for spatial
        quadrants = {
            'Top-Left': lambda p: p.x < center_x and p.y < center_y,
            'Top-Right': lambda p: p.x >= center_x and p.y < center_y,
            'Bottom-Left': lambda p: p.x < center_x and p.y >= center_y,
            'Bottom-Right': lambda p: p.x >= center_x and p.y >= center_y,
            'Center': lambda p: abs(p.x - center_x) < width//4 and abs(p.y - center_y) < height//4,
        }

        for quad_name, condition in quadrants.items():
            scored = [(p, score_with_strategy(p, mins, maxs, 'Balanced')) for p in positions if condition(p)]
            if scored:
                scored.sort(key=lambda x: x[1], reverse=True)
                pos, score = scored[0]
                candidate = ScoredCandidate(pos.x, pos.y, score, f"Spatial:{quad_name}")
                all_candidates.append(candidate)

        # Deduplicate and take top 10
        seen = set()
        unique_candidates = []
        for cand in sorted(all_candidates, key=lambda c: c.score, reverse=True):
            key = (cand.x, cand.y)
            if key not in seen and cand.x > 0 and cand.y > 0:
                seen.add(key)
                unique_candidates.append(cand)
                if len(unique_candidates) >= 10:
                    break

        # Extract sample frame for previews
        duration = get_video_duration(input_file)
        sample_time = duration / 2

        state.update(status="generating_previews", message="Extracting sample frame...")
        print(f"Extracting sample frame at {sample_time:.1f}s for preview generation...")

        preview_dir = os.getcwd()  # Use absolute path for Flask to find images
        base_name = Path(input_file).stem
        temp_frame = f".{base_name}_temp_frame.jpg"

        # Update state with absolute preview directory
        state.preview_dir = preview_dir

        cmd = [
            'ffmpeg', '-ss', str(sample_time), '-i', input_file,
            '-vframes', '1', '-q:v', '2',
            temp_frame, '-y'
        ]
        subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

        # Clean up old preview files from previous runs
        for old_preview in Path('.').glob(f"{base_name}_crop_option_*.jpg"):
            old_preview.unlink()

        # Generate preview crops
        print()
        print("Generating preview crops...")

        for i, candidate in enumerate(unique_candidates, 1):
            preview_file = f"{base_name}_crop_option_{i}.jpg"
            print(f"  [{i}/{len(unique_candidates)}] {candidate.strategy} (x={candidate.x}, y={candidate.y}, score={candidate.score:.2f})")
            print(f"        Generating preview... ", end='', flush=True)

            cmd = [
                'ffmpeg', '-i', temp_frame,
                '-vf', f'crop={crop_w}:{crop_h}:{candidate.x}:{candidate.y}',
                preview_file, '-y'
            ]
            subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

            print(f"✓ {preview_file}")

        # Update state with candidates
        state.update(status="candidates_ready", candidates=unique_candidates, message="Preview options ready")

        print()
        print("="*50)
        print("Review options at: http://localhost:{}".format(port))
        print("Or view preview images: {}_crop_option_*.jpg".format(base_name))
        print("="*50)
        print()

        # Interactive selection with web UI awareness
        # First check for non-interactive mode (used by tests and automated workflows)
        if os.getenv('AUTO_CONFIRM'):
            print("AUTO_CONFIRM enabled, using automatic selection")
            selected = unique_candidates[0]
            print(f"Using automatic selection: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
            print(f"Strategy: {selected.strategy}")
        # Then check if already selected via web UI before we got here
        elif (web_selection := state.get('selected_index')) is not None:
            selected = unique_candidates[web_selection - 1]
            print(f"Using web UI selection #{web_selection}: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
            print(f"Strategy: {selected.strategy}")
        else:
            # Poll for web selection while waiting for text input
            print(f"Waiting for selection via web UI or text input...")
            print(f"  - Web UI: Open http://localhost:{port} and click an option")
            print(f"  - Text: Enter a number [1-{len(unique_candidates)}] or press Enter for automatic selection")
            print()

            import select
            import sys

            # Wait up to 300 seconds (5 minutes) for selection
            max_wait = 300
            poll_interval = 0.5
            elapsed = 0
            choice = None

            # Check if stdin is a terminal
            is_terminal = sys.stdin.isatty()

            if is_terminal:
                print(f"Which crop looks best? [1-{len(unique_candidates)}] ", end='', flush=True)

                while elapsed < max_wait:
                    # Check for web UI selection
                    web_selection = state.get('selected_index')
                    if web_selection is not None:
                        print(f"\n\n✓ Selection received from web UI: #{web_selection}")
                        selected = unique_candidates[web_selection - 1]
                        break

                    # Check for text input (non-blocking on Unix-like systems)
                    if hasattr(select, 'select'):
                        ready, _, _ = select.select([sys.stdin], [], [], poll_interval)
                        if ready:
                            choice = sys.stdin.readline().strip()
                            break
                    else:
                        # Fallback for systems without select (Windows)
                        time.sleep(poll_interval)

                    elapsed += poll_interval

                # If we timed out or got input via stdin
                if web_selection is None:
                    if choice is None and elapsed >= max_wait:
                        # Timeout - use automatic selection
                        print("\n\nNo selection made, using automatic selection")
                        choice = ""
                    elif choice is None:
                        # This shouldn't happen, but just in case
                        choice = input()
            else:
                # Not a terminal (piped input or non-interactive), read directly
                print(f"Which crop looks best? [1-{len(unique_candidates)}] (or press Enter for automatic selection): ", end='', flush=True)
                choice = input().strip()

            # Process text input choice if no web selection was made
            if web_selection is None:
                if not choice:
                    selected = unique_candidates[0]
                    print(f"Using automatic selection: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
                    print(f"Strategy: {selected.strategy}")
                else:
                    try:
                        idx = int(choice) - 1
                        if 0 <= idx < len(unique_candidates):
                            selected = unique_candidates[idx]
                            state.update(selected_index=int(choice))
                            print(f"Using your selection #{choice}: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
                            print(f"Strategy: {selected.strategy}")
                        else:
                            print("Invalid selection, using automatic selection")
                            selected = unique_candidates[0]
                    except ValueError:
                        print("Invalid input, using automatic selection")
                        selected = unique_candidates[0]
            else:
                print(f"Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
                print(f"Strategy: {selected.strategy}")

        crop_x, crop_y = selected.x, selected.y
        selected_strategy = selected.strategy

        # Update state with selected strategy
        state.update(selected_strategy=selected_strategy, status='awaiting_acceleration_choice')

        # Cleanup temp files
        if os.path.exists(temp_frame):
            os.remove(temp_frame)

        print()
        print(f"Preview files saved: {base_name}_crop_option_*.jpg")
        print("(You can delete these preview files once you're satisfied with the result)")

    # Ask user if they want to accelerate boring sections (wait for both web UI and text input)
    print()
    print("="*70)
    print("Would you like to intelligently accelerate boring parts of the video?")
    print("This analyzes the video and speeds up sections with low activity")
    print(f"based on your selected strategy ('{selected_strategy}').")
    print("="*70)
    print()

    # Check for explicit ENABLE_ACCELERATION setting first (for tests)
    enable_accel_env = os.getenv('ENABLE_ACCELERATION')
    if enable_accel_env is not None:
        enable_speedup = enable_accel_env.lower() in ('true', '1', 'yes')
        print(f"ENABLE_ACCELERATION set to: {enable_speedup}")
    # Check for non-interactive mode
    elif os.getenv('AUTO_CONFIRM'):
        print("AUTO_CONFIRM enabled, defaulting to no acceleration")
        enable_speedup = False
    # Check if web UI already made the choice
    elif (web_acceleration_choice := state.get('enable_acceleration')) is not None:
        # Web UI made the choice
        print(f"Using web UI choice: {'Yes' if web_acceleration_choice else 'No'}")
        enable_speedup = web_acceleration_choice
    else:
        # Wait for either web UI or text input
        print(f"Waiting for acceleration choice via web UI or text input...")
        print(f"  - Web UI: Open http://localhost:{port}, select crop, and choose acceleration")
        print(f"  - Text: Enter below")
        print()

        import select
        import sys

        # Wait for selection
        max_wait = 300
        poll_interval = 0.5
        elapsed = 0
        choice = None

        # Check if stdin is a terminal
        is_terminal = sys.stdin.isatty()

        if is_terminal:
            print(f"Accelerate boring sections? [y/N]: ", end='', flush=True)

            while elapsed < max_wait:
                # Check for web UI selection
                web_choice = state.get('enable_acceleration')
                if web_choice is not None:
                    print(f"\n\n✓ Choice received from web UI: {'Yes' if web_choice else 'No'}")
                    enable_speedup = web_choice
                    break

                # Check for text input (non-blocking on Unix-like systems)
                if hasattr(select, 'select'):
                    ready, _, _ = select.select([sys.stdin], [], [], poll_interval)
                    if ready:
                        choice = sys.stdin.readline().strip()
                        break
                else:
                    # Fallback for systems without select (Windows)
                    time.sleep(poll_interval)

                elapsed += poll_interval

            # If we timed out or got input via stdin
            if web_choice is None:
                if choice is None and elapsed >= max_wait:
                    # Timeout - default to no acceleration
                    print("\n\nNo selection made, defaulting to no acceleration")
                    enable_speedup = False
                elif choice is None:
                    # This shouldn't happen, but just in case
                    choice = input().strip()
                    enable_speedup = choice.lower() in ('y', 'yes')
                else:
                    enable_speedup = choice.lower() in ('y', 'yes')
        else:
            # Not a terminal (piped input or non-interactive), read directly
            print(f"Accelerate boring sections? [y/N]: ", end='', flush=True)
            choice = input().strip()
            enable_speedup = choice.lower() in ('y', 'yes')

    scenes = None
    scene_selections = None

    if enable_speedup:
        scenes = analyze_temporal_patterns(input_file, crop_x, crop_y, crop_w, crop_h,
                                          selected_strategy, base_name, state)

        if scenes:
            # Wait for user to select scenes (from web UI or text interface)
            print()
            print(f"Waiting for scene selection via web UI or text input...")
            print(f"  - Web UI: Open http://localhost:{port} to select scenes visually")
            print(f"  - Text: Scene thumbnails saved as {base_name}_scene_*_first.jpg and *_last.jpg")
            print()

            # Check if web UI already made selections
            web_scene_selections = state.get('scene_selections')

            if web_scene_selections is not None:
                print(f"Using web UI selections: {len(web_scene_selections)} scenes to accelerate")
                scene_selections = web_scene_selections
            else:
                # Wait for web UI selection
                max_wait = 600  # 10 minutes
                poll_interval = 0.5
                elapsed = 0

                print("Waiting for scene selections from web UI...")
                print(f"(Timeout in {max_wait}s)")

                while elapsed < max_wait:
                    web_selections = state.get('scene_selections')
                    if web_selections is not None:
                        print(f"\n✓ Selections received from web UI: {len(web_selections)} scenes to accelerate")
                        scene_selections = web_selections
                        break

                    time.sleep(poll_interval)
                    elapsed += poll_interval

                if scene_selections is None:
                    print("\n\nNo scene selections made, skipping acceleration")

    # Convert scene selections to boring_sections format for encoding
    boring_sections = []
    if scene_selections and scenes:
        for scene_idx, speedup_factor in scene_selections.items():
            # scene_idx is 1-based from UI, convert to 0-based
            boring_sections.append((scene_idx - 1, speedup_factor))
        print(f"\nAccelerating {len(boring_sections)} scenes with custom speeds")

    # Apply the crop (with or without variable speed)
    state.update(status="encoding", message="Encoding video...")
    print()
    print(f"Applying crop: {crop_w}x{crop_h} at position ({crop_x},{crop_y})")
    print(f"Encoding with preset: {preset}")

    # Remove existing output file if it exists to ensure clean overwrite
    if os.path.exists(output_file):
        try:
            os.remove(output_file)
            print(f"Removed existing output file: {output_file}")
        except Exception as e:
            print(f"Warning: Could not remove existing output file: {e}")

    if boring_sections and scenes:
        # Sort boring_sections by scene index
        boring_sections.sort(key=lambda x: x[0])
        # Use variable-speed encoding
        print("Encoding with variable speed for boring sections...")
        print()
        encode_with_variable_speed(input_file, output_file, crop_w, crop_h,
                                   crop_x, crop_y, scenes, boring_sections,
                                   preset, state)
        state.update(encoding_progress=100)

    else:
        # Normal encoding without variable speed
        # Get total frame count for progress tracking
        print("Estimating total frames for progress tracking...")
        total_frames = get_video_frame_count(input_file)
        print(f"Estimated frames: {total_frames} (duration × fps)")

        if total_frames <= 0:
            print("Warning: Could not determine frame count, progress tracking will be disabled")
            total_frames = 1  # Avoid division by zero

        # Create a temporary file for FFmpeg progress output
        import tempfile
        progress_fd, progress_file = tempfile.mkstemp(suffix='.txt', text=True)
        os.close(progress_fd)  # Close the file descriptor, we'll read it separately

        cmd = [
            'ffmpeg', '-i', input_file,
            '-vf', f'crop={crop_w}:{crop_h}:{crop_x}:{crop_y}',
            '-c:v', 'libx264', '-preset', preset, '-crf', '19',
            '-c:a', 'copy',
            '-progress', progress_file,  # Write progress to temp file
            '-y', output_file
        ]

        # Run encoding in background
        process = subprocess.Popen(
            cmd,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )

        # Monitor progress file in real-time
        last_frame = 0
        import time as time_module

        print(f"Monitoring progress file: {progress_file}")
        state.update(encoding_progress=0)  # Initialize to 0

        while process.poll() is None:  # While process is running
            try:
                with open(progress_file, 'r') as f:
                    lines = f.readlines()

                # Parse the progress file
                current_frame = 0
                for line in lines:
                    line = line.strip()
                    if line.startswith('frame='):
                        try:
                            current_frame = int(line.split('=')[1])
                        except (ValueError, IndexError):
                            pass

                # Update if we have a new frame count
                if current_frame > last_frame and current_frame > 0:
                    last_frame = current_frame
                    progress = min(int((current_frame / total_frames) * 100), 99)
                    state.update(encoding_progress=progress)
                    print(f"\rEncoding: {current_frame}/{total_frames} frames ({progress}%)  ", end='', flush=True)

            except (IOError, FileNotFoundError) as e:
                # File might not exist yet - this is normal at the start
                pass

            time_module.sleep(0.5)  # Check every 500ms

        # Wait for process to fully complete
        process.wait()

        # Clean up progress file
        try:
            os.unlink(progress_file)
        except:
            pass

        print()  # New line after progress

    state.update(status="complete", encoding_progress=100, message="Encoding complete!")

    # Clean up all temporary files
    print("Cleaning up temporary files...")

    # Remove crop option preview images
    for preview in Path('.').glob(f"{base_name}_crop_option_*.jpg"):
        try:
            preview.unlink()
        except:
            pass

    # Remove scene thumbnails
    for thumbnail in Path('.').glob(f".{base_name}_scene_*_first.jpg"):
        try:
            thumbnail.unlink()
        except:
            pass
    for thumbnail in Path('.').glob(f".{base_name}_scene_*_last.jpg"):
        try:
            thumbnail.unlink()
        except:
            pass

    # Remove temporary frame if it exists
    temp_frame = f".{base_name}_temp_frame.jpg"
    if os.path.exists(temp_frame):
        try:
            os.remove(temp_frame)
        except:
            pass

    print(f"Done! Output saved to: {output_file}")
    print(f"Final dimensions: {crop_w}x{crop_h} (aspect ratio {aspect_ratio})")
    print()
    print("You can now close the web UI.")


if __name__ == '__main__':
    main()
