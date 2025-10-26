#!/bin/sh
# Intelligently crops video to specified aspect ratio (default square 1:1)
# by analyzing motion and visual activity to find the most interesting region
#
# Usage: smart-crop-video input.mp4 [output.mp4] [aspect_ratio]
#   aspect_ratio format: W:H (e.g., "1:1", "4:5", "9:16")
#   default aspect ratio: 1:1 (square)
#
# Environment variables:
#   PRESET: FFmpeg encoding preset (ultrafast/fast/medium/slow) [default: medium]
#   ANALYSIS_FRAMES: Number of frames to analyze per position [default: 50]
#   CROP_SCALE: Scale factor for crop size (0.5-1.0) [default: 0.75]
#               Lower values = more aggressive, smaller crops, focus on interesting content
#               1.0 = maximize crop size, 0.75 = use 75% of max (more aggressive)
#
# Technique: Uses FFmpeg's cropdetect and entropy analysis to find
# the region with most visual activity/motion
#
# Note: Higher analysis frames = better accuracy and fewer re-runs needed

set -e

INPUT="$1"
ASPECT="${3:-1:1}"

# Performance tuning via environment variables
PRESET="${PRESET:-medium}"
ANALYSIS_FRAMES="${ANALYSIS_FRAMES:-50}"
CROP_SCALE="${CROP_SCALE:-0.75}"

if [ -z "$INPUT" ]; then
    echo "Usage: smart-crop-video input.mp4 [output.mp4] [aspect_ratio]"
    echo "  aspect_ratio format: W:H (e.g., '1:1', '4:5', '9:16')"
    echo "  default: 1:1 (square)"
    echo "  default output: input.smart_cropped.ext"
    exit 1
fi

# Generate default output filename: input.smart_cropped.ext
if [ -z "$2" ]; then
    # Extract basename without extension and extension
    BASE="${INPUT%.*}"
    EXT="${INPUT##*.}"
    OUTPUT="${BASE}.smart_cropped.${EXT}"
else
    OUTPUT="$2"
fi

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file '$INPUT' not found"
    exit 1
fi

echo "Analyzing video: $INPUT"
echo "Target aspect ratio: $ASPECT"

# Parse aspect ratio
ASPECT_W=$(echo "$ASPECT" | cut -d: -f1)
ASPECT_H=$(echo "$ASPECT" | cut -d: -f2)

# Get video dimensions
DIMENSIONS=$(ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0 "$INPUT")
WIDTH=$(echo "$DIMENSIONS" | cut -d, -f1)
HEIGHT=$(echo "$DIMENSIONS" | cut -d, -f2)

echo "Original dimensions: ${WIDTH}x${HEIGHT}"
echo "Crop scale factor: ${CROP_SCALE} (lower = more aggressive cropping)"

# Calculate maximum crop dimensions based on aspect ratio
# We need to determine crop_width and crop_height
# such that crop_width/crop_height = ASPECT_W/ASPECT_H

# Start with the smaller dimension as a constraint
if [ "$WIDTH" -lt "$HEIGHT" ]; then
    # Width is limiting - use full width
    MAX_CROP_W=$WIDTH
    MAX_CROP_H=$((WIDTH * ASPECT_H / ASPECT_W))

    # If calculated height exceeds video height, recalculate
    if [ "$MAX_CROP_H" -gt "$HEIGHT" ]; then
        MAX_CROP_H=$HEIGHT
        MAX_CROP_W=$((HEIGHT * ASPECT_W / ASPECT_H))
    fi
else
    # Height is limiting - use full height
    MAX_CROP_H=$HEIGHT
    MAX_CROP_W=$((HEIGHT * ASPECT_W / ASPECT_H))

    # If calculated width exceeds video width, recalculate
    if [ "$MAX_CROP_W" -gt "$WIDTH" ]; then
        MAX_CROP_W=$WIDTH
        MAX_CROP_H=$((WIDTH * ASPECT_H / ASPECT_W))
    fi
fi

# Apply scale factor to allow more aggressive cropping
# This sacrifices resolution to focus on the most interesting content
CROP_W=$(echo "$MAX_CROP_W $CROP_SCALE" | awk '{printf "%d", $1 * $2}')
CROP_H=$(echo "$MAX_CROP_H $CROP_SCALE" | awk '{printf "%d", $1 * $2}')

# Ensure even dimensions (required for some video codecs)
CROP_W=$((CROP_W - CROP_W % 2))
CROP_H=$((CROP_H - CROP_H % 2))

echo "Max crop dimensions: ${MAX_CROP_W}x${MAX_CROP_H}"
echo "Actual crop dimensions: ${CROP_W}x${CROP_H} (${CROP_SCALE}x scale)"

# Calculate how much horizontal and vertical movement is possible
MAX_X=$((WIDTH - CROP_W))
MAX_Y=$((HEIGHT - CROP_H))

if [ "$MAX_X" -le 0 ] && [ "$MAX_Y" -le 0 ]; then
    # No cropping movement possible
    CROP_X=0
    CROP_Y=0
    echo "Crop dimensions match video, no position analysis needed"
else
    echo "Analyzing video for optimal crop position..."
    echo "Preset: $PRESET | Analysis frames per position: $ANALYSIS_FRAMES"
    echo "Scanning a 5x5 grid (25 positions) for regions with most motion/activity..."
    echo "This may take a moment..."

    # Test a 5x5 grid of positions for finer-grained motion detection
    # This allows better centering of action/subjects
    X_POSITIONS="0 $((MAX_X / 4)) $((MAX_X / 2)) $((MAX_X * 3 / 4)) $MAX_X"
    Y_POSITIONS="0 $((MAX_Y / 4)) $((MAX_Y / 2)) $((MAX_Y * 3 / 4)) $MAX_Y"

    BEST_SCORE=0
    BEST_X=$((MAX_X / 2))  # Default to center
    BEST_Y=$((MAX_Y / 2))  # Default to center
    POSITION_NUM=0
    TOTAL_POSITIONS=25

    # First pass: collect all scores to find min/max for normalization
    echo "Pass 1: Collecting scores from all positions..."
    ALL_SCORES=""
    for Y in $Y_POSITIONS; do
        for X in $X_POSITIONS; do
            STATS_OUTPUT=$(ffmpeg -i "$INPUT" -vf "crop=${CROP_W}:${CROP_H}:${X}:${Y},showinfo" -frames:v $ANALYSIS_FRAMES -f null - 2>&1)

            MOTION=$(echo "$STATS_OUTPUT" | grep "Parsed_showinfo" | grep -o "mean:\[[0-9 ]*\]" | \
                    sed 's/mean:\[//' | sed 's/\]//' | awk '{print $1}' | \
                    awk 'NR>1{diff=$1-prev; if(diff<0) diff=-diff; sum+=diff; count++} {prev=$1} END {if(count>0) printf "%.2f", sum/count; else print "0"}')

            COMPLEXITY=$(echo "$STATS_OUTPUT" | grep "Parsed_showinfo" | grep -o "stdev:\[[0-9. ]*\]" | \
                    sed 's/stdev:\[//' | sed 's/\]//' | awk '{print $1}' | \
                    awk '{sum+=$1; count++} END {if(count>0) printf "%.2f", sum/count; else print "0"}')

            MOTION=${MOTION:-0}
            COMPLEXITY=${COMPLEXITY:-0}

            ALL_SCORES="$ALL_SCORES$X,$Y,$MOTION,$COMPLEXITY
"
        done
    done

    # Find min/max for normalization
    MOTION_STATS=$(echo "$ALL_SCORES" | awk -F, '{if(NF>=3) print $3}' | awk 'NR==1{min=$1; max=$1} {if($1<min) min=$1; if($1>max) max=$1} END {print min " " max}')
    MOTION_MIN=$(echo "$MOTION_STATS" | awk '{print $1}')
    MOTION_MAX=$(echo "$MOTION_STATS" | awk '{print $2}')

    COMPLEXITY_STATS=$(echo "$ALL_SCORES" | awk -F, '{if(NF>=4) print $4}' | awk 'NR==1{min=$1; max=$1} {if($1<min) min=$1; if($1>max) max=$1} END {print min " " max}')
    COMPLEXITY_MIN=$(echo "$COMPLEXITY_STATS" | awk '{print $1}')
    COMPLEXITY_MAX=$(echo "$COMPLEXITY_STATS" | awk '{print $2}')

    echo "Motion range: $MOTION_MIN - $MOTION_MAX"
    echo "Complexity range: $COMPLEXITY_MIN - $COMPLEXITY_MAX"
    echo ""
    echo "Pass 2: Scoring with normalized values..."

    # Second pass: normalize and score (avoid subshell to preserve variables)
    POSITION_NUM=0
    while IFS=',' read X Y MOTION_SCORE COMPLEXITY_SCORE; do
        [ -z "$X" ] && continue
        POSITION_NUM=$((POSITION_NUM + 1))

        # Normalize both to 0-100 scale
        MOTION_NORM=$(echo "$MOTION_SCORE $MOTION_MIN $MOTION_MAX" | \
            awk '{range=$3-$2; if(range>0) printf "%.2f", (($1-$2)/range)*100; else print "50"}')

        COMPLEXITY_NORM=$(echo "$COMPLEXITY_SCORE $COMPLEXITY_MIN $COMPLEXITY_MAX" | \
            awk '{range=$3-$2; if(range>0) printf "%.2f", (($1-$2)/range)*100; else print "50"}')

        # Combined score: 80% motion + 20% complexity (motion prioritized for action)
        SCORE=$(echo "$MOTION_NORM $COMPLEXITY_NORM" | awk '{printf "%.2f", ($1 * 0.8) + ($2 * 0.2)}')

        echo "  [$POSITION_NUM/$TOTAL_POSITIONS] Position (x=$X, y=$Y): motion=$MOTION_SCORE→$MOTION_NORM complexity=$COMPLEXITY_SCORE→$COMPLEXITY_NORM combined=$SCORE"

        # Track best score
        IS_BETTER=$(echo "$SCORE $BEST_SCORE" | awk '{if($1 > $2) print "1"; else print "0"}')
        if [ "$IS_BETTER" = "1" ]; then
            BEST_SCORE=$SCORE
            BEST_X=$X
            BEST_Y=$Y
        fi
    done <<EOF
$ALL_SCORES
EOF

    CROP_X=$BEST_X
    CROP_Y=$BEST_Y

    echo "Best crop position: (x=$CROP_X, y=$CROP_Y) with combined score: $BEST_SCORE"
fi

echo "Applying crop: ${CROP_W}x${CROP_H} at position (${CROP_X},${CROP_Y})"
echo "Encoding with preset: $PRESET"

# Apply the crop with configurable encoding preset
ffmpeg -i "$INPUT" \
    -vf "crop=${CROP_W}:${CROP_H}:${CROP_X}:${CROP_Y}" \
    -c:v libx264 -preset "$PRESET" -crf 19 \
    -c:a copy \
    -y "$OUTPUT"

echo "Done! Output saved to: $OUTPUT"
echo "Final dimensions: ${CROP_W}x${CROP_H} (aspect ratio ${ASPECT})"
