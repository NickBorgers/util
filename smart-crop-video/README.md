# Smart Crop Video

Intelligently crops videos to specified aspect ratios (default square 1:1) for social media posting by analyzing visual activity and motion to find the most interesting region.

## Why?

Social media platforms often require specific aspect ratios (square 1:1 for Instagram, 4:5 for Instagram portrait, 9:16 for Stories/Reels/TikTok). Manually cropping videos requires guessing where the action is, often resulting in cut-off subjects or boring static areas. This utility automatically detects where the most interesting content is and crops accordingly.

## Features

- **Interactive crop selection**: Analyzes top 10 crop candidates and lets you choose the best one
- **Intelligent acceleration**: Optionally speeds up boring sections of video (2x-4x) based on your selected strategy
- **Web UI interface**: Ephemeral webserver provides visual preview and selection interface
- **Text interface fallback**: All decisions can be made via command line without the web UI
- **Preview JPEG files**: Generates preview images for each candidate crop position
- **Intelligent motion detection**: Analyzes video to find regions with most visual activity
- **Configurable aspect ratios**: Default 1:1 square, but supports any ratio (4:5, 9:16, etc.)
- **High quality output**: Uses FFmpeg with CRF 19 and slow preset for optimal quality
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

## Interactive Mode

The tool now provides an **interactive selection experience** to help you choose the perfect crop:

### How it Works

1. **Webserver Launch**: Tool starts an ephemeral webserver and displays the URL
2. **Analysis Phase**: Analyzes a 5×5 grid (25 positions) across the entire video frame
3. **Top Candidates**: Identifies the top 10 crop positions based on multiple scoring strategies
4. **Preview Generation**: Extracts a sample frame from the middle of your video
5. **Visual Feedback**:
   - **Web UI**: View all crop options with live previews at the displayed URL
   - **Text Interface**: Lists all options with preview file paths
6. **User Choice**: Select via web UI or text interface (1-10), or press Enter for automatic selection
7. **Progress Tracking**: Both web UI and text interface show encoding progress

### Example Workflow

```bash
smart_crop_video my_video.mp4

# Output will show:
# - Webserver URL (http://localhost:8765)
# - Analysis progress for 25 positions
# - Top 10 candidates with their scores and strategies
# - Preview file paths: my_video_crop_option_1.jpg, my_video_crop_option_2.jpg, etc.
# - Interactive prompt: "Which crop looks best? [1-10]"

# Option 1: Open http://localhost:8765 in your browser to view all previews visually
# Option 2: View the JPEG files in your image viewer
# Option 3: Choose directly by typing a number

# Choose option 3 by typing: 3
# Or press Enter to use the highest-scoring option automatically
# Preview files remain in your directory for reference
```

### Web UI Interface

The tool starts an ephemeral webserver that provides:
- **Live status updates**: See analysis progress in real-time
- **Visual preview gallery**: View all 10 crop options side-by-side
- **Click to select**: Choose your preferred crop with a single click
- **Progress tracking**: Watch encoding progress after selection
- **Automatic shutdown**: Server stops when the job completes

**The web UI is completely optional** - you can make all decisions via the text interface if you prefer.

### Benefits

- **No more guessing**: See exactly what each crop will look like before committing
- **Better results**: Human judgment can catch cases where the algorithm's top choice isn't ideal
- **Faster iteration**: No need to run the full encoding multiple times to get the crop right
- **Visual feedback**: Preview JPEG files let you evaluate options before the final encode

## Intelligent Acceleration (NEW!)

After selecting your preferred crop, the tool can **automatically accelerate boring sections** of your video to create more engaging content.

### How It Works

1. **Strategy-Based Analysis**: Uses the same metric from your selected crop strategy (motion, edges, complexity)
2. **Scene Detection**: Automatically detects scene changes throughout the video
3. **Intelligent Scoring**: Analyzes each scene using the relevant metric (e.g., motion for "Motion Priority")
4. **Variable Speed**: Boring sections (bottom 30%) are sped up 2x-4x, interesting sections play at normal speed
5. **Smooth Audio**: Adjusts audio tempo to match video speed changes

### Example Use Cases

- **Screen recordings**: Speed through idle moments, keep action at normal speed
- **Timelapses**: Accelerate static periods while maintaining interesting changes
- **Tutorial videos**: Fast-forward through repetitive setup, normal speed for important steps
- **Long recordings**: Compress boring sections automatically without losing important content

### How to Use

After selecting your crop, you'll be prompted:
```
Would you like to intelligently accelerate boring parts of the video?
This analyzes the video and speeds up sections with low activity
based on your selected strategy ('Motion Priority').

Accelerate boring sections? [y/N]:
```

Simply type `y` and press Enter to enable, or press Enter to skip.

### What Happens

1. **Scene Detection**: Detects all scene changes in your video (or uses time-based segments as fallback)
2. **Automatic Fallback**: If fewer than 3 scenes detected, splits video into 5-second segments
3. **Metric Analysis**: Samples 10 frames per scene/segment to calculate the relevant metric
4. **Boring Section Identification**: Finds scenes below the 30th percentile
5. **Time Savings Report**: Shows which sections will be accelerated and by how much
6. **Segment Encoding**: Encodes each scene with appropriate speed (1x, 2x, 3x, or 4x)
7. **Final Concatenation**: Combines all segments into the final video

### Configuration

Fine-tune the boring section detection:
```bash
# Adjust what qualifies as "boring" (default: 30th percentile)
BORING_THRESHOLD=40.0 smart_crop_video video.mp4

# Lower = more aggressive (more sections sped up)
BORING_THRESHOLD=20.0 smart_crop_video video.mp4

# Higher = more conservative (fewer sections sped up)
BORING_THRESHOLD=40.0 smart_crop_video video.mp4

# Adjust scene detection sensitivity (default: 0.2)
# Lower = more scene changes detected, higher = fewer scenes
SCENE_THRESHOLD=0.15 smart_crop_video video.mp4

# Adjust time-based segment duration when scene detection finds too few scenes (default: 5.0s)
# Shorter = more granular analysis, longer = broader analysis
SEGMENT_DURATION=3.0 smart_crop_video video.mp4

# Combine settings for fine control
BORING_THRESHOLD=25.0 SEGMENT_DURATION=4.0 smart_crop_video video.mp4
```

**Scene Detection vs Time-Based Segmentation:**
- The tool first tries to detect natural scene changes (cuts, fades, etc.)
- If fewer than 3 scenes are detected, it automatically falls back to time-based segments
- This ensures the feature works even on videos without scene changes (e.g., continuous screen recordings)
- You can force smaller/larger segments with `SEGMENT_DURATION`

### Performance Note

Enabling acceleration adds processing time for scene analysis but creates shorter, more engaging final videos. The analysis typically takes 1-3 minutes depending on video length and number of scenes.

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

### Crop Position Analysis

1. **Analyze dimensions**: Determines video size and calculates crop dimensions for target aspect ratio
2. **Apply crop scale**: Reduces crop size by scale factor (default 0.75x) to allow scanning for subjects/content
3. **Sample 5×5 grid**: Tests 25 positions across the entire frame for fine-grained positioning
4. **Measure key metrics**: For each position:
   - **Edge detection**: Uses ffmpeg's edgedetect filter to identify areas with defined subjects, people, and features
   - **Visual complexity**: Measures pixel variance (stdev) - areas with more detail and variation
   - **Temporal motion**: Measures frame-to-frame differences - actual movement between frames
   - **Color saturation**: Identifies colorful, vibrant areas
5. **Generate strategies**: Creates 10 crop candidates using different scoring strategies (Motion Priority, Visual Detail, Subject Detection, etc.)
6. **Interactive selection**: Presents top 10 candidates as preview JPEGs for user review
7. **Optional acceleration**: Analyzes entire video to identify and speed up boring sections based on selected strategy
8. **Apply crop**: Crops video (with or without variable speed) using high-quality encoding settings

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

This utility runs with Docker port mapping (`-p 8765:8765`) to enable the ephemeral webserver for the interactive UI. The webserver:
- Only accessible on localhost at port 8765 (not accessible from other machines)
- Automatically shuts down when the job completes
- Only serves preview images from your current directory
- Runs only during the video processing session
- Uses a fixed port (8765) for predictability and cross-platform compatibility
