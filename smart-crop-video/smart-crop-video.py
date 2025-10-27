#!/usr/bin/env python3
"""
Intelligently crops videos to specified aspect ratios by analyzing visual activity
and motion to find the most interesting region.

Uses multiple scoring strategies to generate candidate crops for user selection.
"""

import sys
import os
import re
import subprocess
import argparse
from pathlib import Path
from typing import List, Tuple, Dict
from dataclasses import dataclass


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
                print(f"\r[{percent:3d}%] Analyzing position {current:2d}/{total} (x={x}, y={y})...  ", end='', flush=True)

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

        print(f"Extracting sample frame at {sample_time:.1f}s for preview generation...")

        preview_dir = "."
        base_name = Path(input_file).stem
        temp_frame = f".{base_name}_temp_frame.jpg"

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

        print()
        print("="*50)
        print("Please review the preview images above.")
        print("Each option uses a different scoring strategy.")
        print("Open them in your image viewer if needed.")
        print("="*50)
        print()

        # Interactive selection
        choice = input(f"Which crop looks best? [1-{len(unique_candidates)}] (or press Enter for automatic selection): ").strip()

        if not choice:
            selected = unique_candidates[0]
            print(f"Using automatic selection: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
            print(f"Strategy: {selected.strategy}")
        else:
            try:
                idx = int(choice) - 1
                if 0 <= idx < len(unique_candidates):
                    selected = unique_candidates[idx]
                    print(f"Using your selection #{choice}: Position (x={selected.x}, y={selected.y}) with score: {selected.score:.2f}")
                    print(f"Strategy: {selected.strategy}")
                else:
                    print("Invalid selection, using automatic selection")
                    selected = unique_candidates[0]
            except ValueError:
                print("Invalid input, using automatic selection")
                selected = unique_candidates[0]

        crop_x, crop_y = selected.x, selected.y

        # Cleanup temp files
        if os.path.exists(temp_frame):
            os.remove(temp_frame)

        print()
        print(f"Preview files saved: {base_name}_crop_option_*.jpg")
        print("(You can delete these preview files once you're satisfied with the result)")

    # Apply the crop
    print(f"Applying crop: {crop_w}x{crop_h} at position ({crop_x},{crop_y})")
    print(f"Encoding with preset: {preset}")

    cmd = [
        'ffmpeg', '-i', input_file,
        '-vf', f'crop={crop_w}:{crop_h}:{crop_x}:{crop_y}',
        '-c:v', 'libx264', '-preset', preset, '-crf', '19',
        '-c:a', 'copy',
        '-y', output_file
    ]

    subprocess.run(cmd)

    print(f"Done! Output saved to: {output_file}")
    print(f"Final dimensions: {crop_w}x{crop_h} (aspect ratio {aspect_ratio})")


if __name__ == '__main__':
    main()
