# Smart Crop Video

Intelligently crops videos to specified aspect ratios (default square 1:1) for social media posting by analyzing visual activity and motion to find the most interesting region.

## Why?

Social media platforms often require specific aspect ratios (square 1:1 for Instagram, 4:5 for Instagram portrait, 9:16 for Stories/Reels/TikTok). Manually cropping videos requires guessing where the action is, often resulting in cut-off subjects or boring static areas. This utility automatically detects where the most interesting content is and crops accordingly.

## Features

- **Intelligent motion detection**: Analyzes video to find regions with most visual activity
- **Configurable aspect ratios**: Default 1:1 square, but supports any ratio (4:5, 9:16, etc.)
- **High quality output**: Uses FFmpeg with CRF 19 and slow preset for optimal quality
- **Network isolated**: Runs with `--network=none` Docker flag for security
- **Simple one-liner**: Just point it at your video file

## Installation

Add the contents of the `profile` file from the repository root to your shell profile (`.bashrc`, `.zshrc`, etc.):

```bash
cat profile >> ~/.zshrc
source ~/.zshrc
```

The Docker image will be pulled automatically on first use.

## Usage

### Basic usage (square 1:1 crop)
```bash
smart_crop_video input.mp4
# Creates input.smart_cropped.mp4

smart_crop_video my_video.mov
# Creates my_video.smart_cropped.mov
```

### Specify output filename
```bash
smart_crop_video input.mp4 custom_output.mp4
```

### Specify aspect ratio
```bash
# Instagram portrait (4:5)
smart_crop_video input.mp4 output.mp4 4:5

# Stories/Reels/TikTok (9:16) with auto-generated output name
smart_crop_video recording.mp4 "" 9:16
# Creates recording.smart_cropped.mp4

# Twitter/X (16:9)
smart_crop_video input.mp4 output.mp4 16:9
```

**Note**: The default output filename is automatically generated as `input.smart_cropped.ext` to preserve the original filename.

## Aggressive Cropping

**By default, this tool prioritizes interesting content over resolution.** It uses a **0.75 crop scale**, meaning it will use only 75% of the maximum possible crop size to allow more freedom in finding the most visually interesting region.

### Crop Scale Factor

Control how aggressively the tool crops to find interesting content:

```bash
# Default: aggressive cropping (75% of max size)
smart_crop_video input.mp4  # CROP_SCALE=0.75

# Maximum aggression - smallest crop, focuses on most interesting area
CROP_SCALE=0.5 smart_crop_video input.mp4

# Moderate aggression
CROP_SCALE=0.85 smart_crop_video input.mp4

# No aggression - maximize crop size (old behavior)
CROP_SCALE=1.0 smart_crop_video input.mp4
```

**How it works:**
- The tool analyzes a **5×5 grid** (25 positions) across the entire video frame
- Tests positions both horizontally AND vertically for fine-grained positioning
- Measures **actual motion** (temporal frame differences) + **visual complexity** (entropy)
- Combines scores with 60% weight on motion, 40% on entropy
- Selects the position that best centers the action/movement
- Lower `CROP_SCALE` = smaller crop = more scanning area = better at finding and centering action
- Higher resolution is sacrificed for more engaging, better-centered content

### When to adjust CROP_SCALE

- **0.5-0.6**: Screen recordings with small active areas, presentations with limited motion regions
- **0.75** (default): Good balance for most content with some static areas
- **0.85-0.9**: Content where most of the frame is interesting
- **1.0**: You want maximum resolution and the entire frame is important

## Performance Tuning

The utility supports environment variables for performance tuning, especially useful on Mac Silicon:

### Encoding Preset
Control encoding speed vs quality tradeoff:
```bash
# Faster encoding (good for testing/previews)
PRESET=fast smart_crop_video input.mp4

# Maximum speed (great for large files)
PRESET=ultrafast smart_crop_video input.mp4

# High quality (slower, default: medium)
PRESET=slow smart_crop_video input.mp4
```

**Preset options**: `ultrafast`, `superfast`, `veryfast`, `faster`, `fast`, `medium` (default), `slow`, `slower`, `veryslow`

### Analysis Frames
Control how many frames are analyzed per position (default: 50):
```bash
# Faster analysis (fewer frames, less accurate)
ANALYSIS_FRAMES=20 smart_crop_video input.mp4

# Default: thorough analysis (prevents re-runs)
ANALYSIS_FRAMES=50 smart_crop_video input.mp4

# Maximum thoroughness for critical content
ANALYSIS_FRAMES=100 smart_crop_video input.mp4
```

**Philosophy**: More frames upfront = better results = fewer re-runs = saves time overall

### Combined Tuning Examples
```bash
# Quick preview (fast but less accurate)
PRESET=fast ANALYSIS_FRAMES=20 CROP_SCALE=0.75 smart_crop_video input.mp4

# Default: balanced settings (recommended)
PRESET=medium ANALYSIS_FRAMES=50 CROP_SCALE=0.75 smart_crop_video input.mp4

# High accuracy + very aggressive for screen recordings
PRESET=medium ANALYSIS_FRAMES=75 CROP_SCALE=0.6 smart_crop_video input.mp4

# Maximum quality for final exports
PRESET=slow ANALYSIS_FRAMES=100 CROP_SCALE=0.75 smart_crop_video input.mp4

# Maximum resolution preservation
PRESET=slow ANALYSIS_FRAMES=50 CROP_SCALE=1.0 smart_crop_video input.mp4
```

### Performance Notes for Mac Silicon

The Docker image is built natively for ARM64 (Apple Silicon), providing optimal performance. Additional tips:

- **Default settings** (medium preset, 50 frames, 0.75 crop scale) prioritize accuracy over speed
- **For quick previews**: Use `PRESET=fast ANALYSIS_FRAMES=20` for faster processing
- **The tool analyzes a 5×5 grid** (25 positions) for fine-grained motion detection
- **Default analysis**: 1,250 frames total (50 frames × 25 positions) for reliable, well-centered results
- **Quick preview**: `ANALYSIS_FRAMES=20` = 500 frames total (20 × 25 positions)
- **Maximum thoroughness**: `ANALYSIS_FRAMES=100` = 2,500 frames total (100 × 25 positions)
- **Each position is tested twice**: once for motion detection, once for entropy (visual complexity)

**Rationale**: Thorough motion analysis centers action better and prevents disappointing re-runs, saving overall time

## How It Works

1. **Analyze dimensions**: Determines video size and calculates crop dimensions for target aspect ratio
2. **Apply crop scale**: Reduces crop size by scale factor (default 0.75x) to allow scanning for action/motion
3. **Sample 5×5 grid**: Tests 25 positions across the entire frame for fine-grained positioning
4. **Measure motion and complexity**: For each position:
   - **Motion score**: Measures temporal frame differences (YDIF) - actual movement between frames
   - **Entropy score**: Measures visual complexity - variation in pixel values
5. **Combine scores**: Weighted combination: 60% motion + 40% entropy (motion prioritized for centering action)
6. **Select best crop**: Chooses the position that best captures and centers the action/movement
7. **Apply crop**: Crops video at the winning position using high-quality encoding settings

**Why this motion-based approach works better:**
- **YDIF (temporal differences)** detects actual movement between frames, not just visual complexity
- Motion is weighted at 60% because it better identifies where subjects/action are
- Entropy (40%) still helps distinguish interesting vs. boring areas
- 5×5 grid (vs 3×3) provides finer positioning to better center the action
- Static backgrounds, empty spaces, and unchanging areas score low on both metrics
- Sacrificing some resolution (via crop scale) allows focusing on where the action actually is

## Technical Details

- **Base image**: Alpine Linux 3.16.2
- **Tools**: FFmpeg with full filter support
- **Encoding**: libx264, CRF 19, slow preset
- **Audio**: Copy (no re-encoding)
- **Analysis**: Samples up to 50 frames per position for entropy calculation

## Common Social Media Aspect Ratios

- **1:1** (Square) - Instagram feed, LinkedIn, Facebook
- **4:5** (Portrait) - Instagram portrait, optimal for mobile feed
- **9:16** (Vertical) - Instagram Stories/Reels, TikTok, YouTube Shorts
- **16:9** (Landscape) - YouTube, Twitter/X, Facebook video
- **2:3** (Tall portrait) - Pinterest

## Examples

Crop a screen recording to square format for Instagram:
```bash
smart_crop_video screen_recording.mp4
# Creates: screen_recording.smart_cropped.mp4
```

Crop a landscape video to vertical format for TikTok:
```bash
smart_crop_video landscape_video.mp4 "" 9:16
# Creates: landscape_video.smart_cropped.mp4
```

Crop a video to Instagram portrait format with custom output:
```bash
smart_crop_video original.mp4 instagram_portrait.mp4 4:5
```

Multiple videos with aggressive cropping:
```bash
CROP_SCALE=0.6 smart_crop_video video1.mp4
CROP_SCALE=0.6 smart_crop_video video2.mp4
CROP_SCALE=0.6 smart_crop_video video3.mp4
# Creates: video1.smart_cropped.mp4, video2.smart_cropped.mp4, video3.smart_cropped.mp4
```

## Requirements

- Docker installed and running
- Input video file in current directory (or use absolute paths)

## Security

This utility runs with Docker's `--network=none` flag, ensuring the container has no network access and cannot exfiltrate data.
