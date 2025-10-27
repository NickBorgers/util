# Smart Crop Video

Intelligently crops videos to specified aspect ratios (default square 1:1) for social media posting by analyzing visual activity and motion to find the most interesting region.

## Why?

Social media platforms often require specific aspect ratios (square 1:1 for Instagram, 4:5 for Instagram portrait, 9:16 for Stories/Reels/TikTok). Manually cropping videos requires guessing where the action is, often resulting in cut-off subjects or boring static areas. This utility automatically detects where the most interesting content is and crops accordingly.

## Features

- **Interactive crop selection**: Analyzes top 5 crop candidates and lets you choose the best one
- **Preview JPEG files**: Generates preview images for each candidate crop position in your current directory
- **Intelligent motion detection**: Analyzes video to find regions with most visual activity
- **Configurable aspect ratios**: Default 1:1 square, but supports any ratio (4:5, 9:16, etc.)
- **High quality output**: Uses FFmpeg with CRF 19 and slow preset for optimal quality
- **Network isolated**: Runs with `--network=none` Docker flag for security
- **Simple one-liner**: Just point it at your video file
- **ASCII art previews** (optional): Shows full frame with crop region highlighted in color (requires jp2a to be added to Docker image)

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

## Interactive Mode

The tool now provides an **interactive selection experience** to help you choose the perfect crop:

### How it Works

1. **Analysis Phase**: The tool analyzes a 5×5 grid (25 positions) across the entire video frame
2. **Top Candidates**: Identifies the top 5 crop positions based on motion and visual complexity
3. **Preview Generation**: Extracts a sample frame from the middle of your video
4. **Visual Feedback**: Shows you:
   - **ASCII art visualization** showing the full frame with crop region highlighted in red
   - **Preview JPEG files** saved to your current directory as `{video_name}_crop_option_*.jpg`
5. **User Choice**: You select which crop looks best (1-5), or press Enter for automatic selection

### Example Workflow

```bash
smart_crop_video my_video.mp4

# Output will show:
# - Analysis progress for 25 positions
# - Top 5 candidates with their scores
# - ASCII art previews (if jp2a is available in the container)
# - Preview file paths: my_video_crop_option_1.jpg, my_video_crop_option_2.jpg, etc.
# - Interactive prompt: "Which crop looks best? [1-5]"

# Choose option 3 by typing: 3
# Or press Enter to use the highest-scoring option automatically
# Preview files remain in your directory for reference
```

### ASCII Art Visualization (Optional Feature)

The script supports ASCII art visualization if `jp2a` is available in the Docker container. When enabled, you'll see beautiful ASCII art previews showing:
- The **full video frame** rendered as ASCII art
- The **crop region highlighted in red** so you can see exactly what will be kept
- Each option's score for reference

**Note**: ASCII art is currently not included by default due to build complexity. The tool works great without it - you'll see the preview JPEG file paths and can open them in any image viewer. If you want ASCII art, you can modify the Dockerfile to add jp2a.

### Benefits

- **No more guessing**: See exactly what each crop will look like before committing
- **Better results**: Human judgment can catch cases where the algorithm's top choice isn't ideal
- **Faster iteration**: No need to run the full encoding multiple times to get the crop right
- **Visual feedback**: Preview JPEG files let you evaluate options before the final encode

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
- Measures **three key metrics**:
  - **Edge detection (40%)**: Identifies areas with defined subjects/features (people, objects)
  - **Visual complexity (50%)**: Measures pixel variance and detail
  - **Temporal motion (10%)**: Detects frame-to-frame changes
- This weighting prioritizes areas with actual subjects over gradual lighting changes (perfect for timelapses!)
- Selects positions that best capture subjects and visual interest
- Lower `CROP_SCALE` = smaller crop = more scanning area = better at finding and centering subjects
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
- **The tool analyzes a 5×5 grid** (25 positions) for fine-grained subject detection
- **Default analysis**: 50 frames × 25 positions × 2 passes = 2,500 frame analyses (motion/complexity + edge detection)
- **Quick preview**: `ANALYSIS_FRAMES=20` = 20 × 25 × 2 = 1,000 frame analyses
- **Maximum thoroughness**: `ANALYSIS_FRAMES=100` = 100 × 25 × 2 = 5,000 frame analyses
- **Each position is tested with two ffmpeg passes**:
  1. Motion + complexity (showinfo)
  2. Edge detection (edgedetect + showinfo)

**Rationale**: Thorough motion analysis centers action better and prevents disappointing re-runs, saving overall time

## How It Works

1. **Analyze dimensions**: Determines video size and calculates crop dimensions for target aspect ratio
2. **Apply crop scale**: Reduces crop size by scale factor (default 0.75x) to allow scanning for subjects/content
3. **Sample 5×5 grid**: Tests 25 positions across the entire frame for fine-grained positioning
4. **Measure three key metrics**: For each position:
   - **Edge detection (40%)**: Uses ffmpeg's edgedetect filter to identify areas with defined subjects, people, and features
   - **Visual complexity (50%)**: Measures pixel variance (stdev) - areas with more detail and variation
   - **Temporal motion (10%)**: Measures frame-to-frame differences - actual movement between frames
5. **Combine scores**: Weighted combination: 10% motion + 50% complexity + 40% edges
6. **Interactive selection**: Presents top 5 candidates as preview JPEGs for user review
7. **Apply crop**: Crops video at the selected position using high-quality encoding settings

**Why this approach works for diverse content:**
- **Edge detection (40%)** identifies people, objects, and defined subjects - not just smooth backgrounds
- **Visual complexity (50%)** distinguishes detailed areas from empty sky/grass
- **Temporal motion (10%)** is de-emphasized to avoid false positives from lighting changes (perfect for timelapses!)
- Works well for:
  - Timelapses where lighting changes but subjects are stationary
  - Screen recordings with specific areas of interest
  - Videos with both static and moving elements
- 5×5 grid provides finer positioning to better center subjects
- Sacrificing some resolution (via crop scale) allows focusing on where the subjects actually are

## Technical Details

- **Base image**: Alpine Linux 3.19
- **Tools**: FFmpeg with full filter support (including edgedetect)
- **Analysis**: Three metrics per position:
  - Motion: Temporal frame differences using showinfo filter
  - Complexity: Pixel variance (stdev) using showinfo filter
  - Edges: Edge detection using edgedetect + showinfo filters
- **Samples**: Up to 50 frames per position (configurable via `ANALYSIS_FRAMES`)
- **Encoding**: libx264, CRF 19, configurable preset (default: medium)
- **Audio**: Copy (no re-encoding)

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
