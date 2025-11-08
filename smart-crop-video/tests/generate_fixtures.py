#!/usr/bin/env python3
"""
One-time script to generate pre-generated test video fixtures.

This script uses the video_generator module to create test videos that will
be committed to the repository, eliminating the need for dynamic generation
during test execution.

Run once to generate fixtures:
    python3 tests/generate_fixtures.py
"""

import sys
from pathlib import Path

# Add parent directory to path so we can import helpers
sys.path.insert(0, str(Path(__file__).parent))

from helpers import video_generator as vg

def main():
    """Generate all test video fixtures."""
    fixtures_dir = Path(__file__).parent / "fixtures"
    fixtures_dir.mkdir(exist_ok=True)

    print("Generating test video fixtures...")
    print(f"Output directory: {fixtures_dir}")
    print()

    # 1. Motion top-right video
    print("1/5 Generating motion_top_right.mov...")
    motion_top_right_path = fixtures_dir / "motion_top_right.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)
    motion = vg.MotionRegion(
        x=1400, y=200,
        size=100,
        color="red",
        speed=100,
        direction="horizontal"
    )
    vg.create_video_with_motion_in_region(
        motion_top_right_path,
        motion,
        config,
        background_color="black"
    )
    print(f"   Created: {motion_top_right_path} ({motion_top_right_path.stat().st_size / 1024 / 1024:.2f} MB)")

    # 2. Motion center video
    print("2/5 Generating motion_center.mov...")
    motion_center_path = fixtures_dir / "motion_center.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)
    motion = vg.MotionRegion(
        x=960, y=540,
        size=150,
        color="blue",
        speed=100,
        direction="circular"
    )
    vg.create_video_with_motion_in_region(
        motion_center_path,
        motion,
        config,
        background_color="black"
    )
    print(f"   Created: {motion_center_path} ({motion_center_path.stat().st_size / 1024 / 1024:.2f} MB)")

    # 3. Subject left video
    print("3/5 Generating subject_left.mov...")
    subject_left_path = fixtures_dir / "subject_left.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=5.0, fps=30)
    vg.create_video_with_subject(
        subject_left_path,
        subject_position=(0.25, 0.5),
        subject_size=250,
        config=config,
        subject_shape="circle",
        subject_color="white",
        background_color="black"
    )
    print(f"   Created: {subject_left_path} ({subject_left_path.stat().st_size / 1024 / 1024:.2f} MB)")

    # 4. Multi-scene video
    print("4/5 Generating multi_scene.mov...")
    multi_scene_path = fixtures_dir / "multi_scene.mov"
    scenes = [
        vg.SceneConfig(duration=5.0, motion_level="high", object_color="red"),
        vg.SceneConfig(duration=5.0, motion_level="low", object_color="blue"),
        vg.SceneConfig(duration=5.0, motion_level="high", object_color="green"),
    ]
    scene_info = vg.create_video_with_scenes(
        multi_scene_path,
        scenes,
        config=vg.VideoConfig(width=1920, height=1080, fps=30)
    )
    print(f"   Created: {multi_scene_path} ({multi_scene_path.stat().st_size / 1024 / 1024:.2f} MB)")

    # 5. Audio test video
    print("5/5 Generating audio_test.mov...")
    audio_test_path = fixtures_dir / "audio_test.mov"
    config = vg.VideoConfig(width=1920, height=1080, duration=10.0, fps=30)
    vg.create_test_video_with_audio(audio_test_path, config, audio_frequency=440)
    print(f"   Created: {audio_test_path} ({audio_test_path.stat().st_size / 1024 / 1024:.2f} MB)")

    print()
    print("✅ All fixtures generated successfully!")

    # Calculate total size
    total_size = sum(f.stat().st_size for f in fixtures_dir.glob("*.mov"))
    print(f"Total size: {total_size / 1024 / 1024:.2f} MB")
    print()
    print("Next steps:")
    print("1. Review the generated videos to ensure they look correct")
    print("2. Update test fixtures to use these pre-generated videos")
    print("3. Run tests to verify: ./run-tests.sh comprehensive")
    print("4. Commit the fixtures to the repository")

if __name__ == "__main__":
    main()
