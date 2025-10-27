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

# Disable Flask development server warning
log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

@dataclass
class CropPosition:
    """Represents a crop position with its metrics"""
    x: int
    y: int
    motion: float = 0.0
    complexity: float = 0.0
    edges: float = 0.0
    saturation: float = 0.0


@dataclass
class ScoredCandidate:
    """Represents a candidate crop with its score and strategy"""
    x: int
    y: int
    score: float
    strategy: str


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

    <div id="encoding-section" class="encoding" style="display: none;">
        <h2>Encoding Video</h2>
        <div class="progress-bar">
            <div class="progress-fill" id="encoding-bar" style="width: 0%">0%</div>
        </div>
    </div>

    <script>
        let selectedIndex = null;

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
                            `Selected option ${selectedIndex}. Encoding video...`;
                    });
            }
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
        state.update(selected_index=index)
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
                    analysis_frames: int) -> CropPosition:
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

    return CropPosition(x, y, motion, complexity, edges, color_variance)


def normalize(value: float, min_val: float, max_val: float) -> float:
    """Normalize value to 0-100 range"""
    if max_val - min_val > 0:
        return ((value - min_val) / (max_val - min_val)) * 100
    return 50.0


def score_with_strategy(pos: CropPosition, mins: Dict[str, float], maxs: Dict[str, float],
                       strategy: str) -> float:
    """Score a position using a specific strategy"""

    # Normalize metrics
    motion_norm = normalize(pos.motion, mins['motion'], maxs['motion'])
    complexity_norm = normalize(pos.complexity, mins['complexity'], maxs['complexity'])
    edge_norm = normalize(pos.edges, mins['edges'], maxs['edges'])
    sat_norm = normalize(pos.saturation, mins['saturation'], maxs['saturation'])

    # Apply strategy weights
    strategies = {
        'Subject Detection': (0.05, 0.25, 0.40, 0.30),
        'Motion Priority': (0.50, 0.15, 0.25, 0.10),
        'Visual Detail': (0.05, 0.50, 0.30, 0.15),
        'Balanced': (0.25, 0.25, 0.25, 0.25),
        'Color Focus': (0.05, 0.20, 0.30, 0.45),
    }

    w_motion, w_complexity, w_edges, w_sat = strategies[strategy]

    return (motion_norm * w_motion + complexity_norm * w_complexity +
            edge_norm * w_edges + sat_norm * w_sat)


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

    if max_x <= 0 and max_y <= 0:
        print("Crop dimensions match video, no position analysis needed")
        crop_x, crop_y = 0, 0
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

        # Generate 5x5 grid positions (start from 1, not 0)
        x_positions = [max(1, max_x * i // 4) for i in range(5)]
        y_positions = [max(1, max_y * i // 4) for i in range(5)]

        # Analyze all positions
        print("Pass 1: Analyzing all positions...")
        print("Metrics: motion, complexity, strong-edges (high-threshold), color-saturation")
        print(f"This will analyze 25 positions × 3 passes (motion/complexity + strong-edges + saturation)")
        print()

        positions = []
        total = len(x_positions) * len(y_positions)
        current = 0

        for y in y_positions:
            for x in x_positions:
                current += 1
                percent = (current * 100) // total
                progress_msg = f"Analyzing position {current}/{total} (x={x}, y={y})"
                print(f"\r[{percent:3d}%] {progress_msg}...  ", end='', flush=True)

                state.update(
                    current_position=current,
                    progress=percent,
                    message=progress_msg
                )

                pos = analyze_position(input_file, x, y, crop_w, crop_h, analysis_frames)
                positions.append(pos)

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
        # First check if already selected via web UI before we got here
        web_selection = state.get('selected_index')

        if web_selection is not None:
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

        # Cleanup temp files
        if os.path.exists(temp_frame):
            os.remove(temp_frame)

        print()
        print(f"Preview files saved: {base_name}_crop_option_*.jpg")
        print("(You can delete these preview files once you're satisfied with the result)")

    # Apply the crop
    state.update(status="encoding", message="Encoding video...")
    print(f"Applying crop: {crop_w}x{crop_h} at position ({crop_x},{crop_y})")
    print(f"Encoding with preset: {preset}")

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

    print(f"Done! Output saved to: {output_file}")
    print(f"Final dimensions: {crop_w}x{crop_h} (aspect ratio {aspect_ratio})")
    print()
    print("You can now close the web UI.")


if __name__ == '__main__':
    main()
