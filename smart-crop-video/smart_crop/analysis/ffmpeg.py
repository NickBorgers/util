"""
FFmpeg-based video analyzer implementation.

This module provides a concrete implementation of VideoAnalyzer using FFmpeg
for all video analysis operations.
"""
import subprocess
import re
from typing import List, Tuple
from smart_crop.analysis.analyzer import VideoAnalyzer
from smart_crop.core.scoring import PositionMetrics


class FFmpegAnalyzer(VideoAnalyzer):
    """
    Video analyzer implementation using FFmpeg/FFprobe.

    This class wraps FFmpeg command-line tools to provide video analysis
    functionality. It implements the VideoAnalyzer interface.

    Args:
        video_path: Path to the video file to analyze

    Examples:
        >>> analyzer = FFmpegAnalyzer("video.mp4")
        >>> width, height = analyzer.get_dimensions()
        >>> print(f"Video is {width}x{height}")
        Video is 1920x1080

        >>> metrics = analyzer.analyze_position(100, 100, 640, 640, sample_frames=30)
        >>> print(f"Motion score: {metrics.motion:.2f}")
        Motion score: 5.23
    """

    def __init__(self, video_path: str):
        """
        Initialize FFmpeg analyzer for a video file.

        Args:
            video_path: Path to video file (relative or absolute)
        """
        self.video_path = video_path

    def get_dimensions(self) -> Tuple[int, int]:
        """
        Get video width and height using ffprobe.

        Returns:
            Tuple of (width, height) in pixels
        """
        cmd = [
            'ffprobe', '-v', 'error',
            '-select_streams', 'v:0',
            '-show_entries', 'stream=width,height',
            '-of', 'csv=p=0',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        width, height = map(int, result.stdout.strip().split(','))
        return width, height

    def get_duration(self) -> float:
        """
        Get video duration using ffprobe.

        Returns:
            Duration in seconds
        """
        cmd = [
            'ffprobe', '-v', 'error',
            '-show_entries', 'format=duration',
            '-of', 'default=noprint_wrappers=1:nokey=1',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        return float(result.stdout.strip())

    def get_fps(self) -> float:
        """
        Get video frame rate using ffprobe.

        Returns:
            Frames per second. Returns 24.0 as default if parsing fails.
        """
        cmd = [
            'ffprobe', '-v', 'error',
            '-select_streams', 'v:0',
            '-show_entries', 'stream=r_frame_rate',
            '-of', 'csv=p=0',
            self.video_path
        ]
        result = subprocess.run(cmd, capture_output=True, text=True)
        fps_str = result.stdout.strip()

        try:
            if '/' in fps_str:
                # Handle fractional frame rates like "30000/1001"
                num, den = map(int, fps_str.split('/'))
                return num / den
            else:
                return float(fps_str)
        except (ValueError, ZeroDivisionError):
            return 24.0  # Default fallback

    def get_frame_count(self) -> int:
        """
        Estimate total number of frames.

        Uses duration × fps for fast estimation.

        Returns:
            Estimated frame count (at least 1)
        """
        fps = self.get_fps()
        duration = self.get_duration()
        estimated_frames = int(duration * fps)
        return max(estimated_frames, 1)  # Ensure at least 1

    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """
        Analyze a crop position using FFmpeg filters.

        Uses multiple FFmpeg passes to analyze:
        1. Motion (frame-to-frame mean differences)
        2. Complexity (standard deviation of pixel values)
        3. Edges (edge detection filter)
        4. Saturation (RGB channel variance)

        Args:
            x: X coordinate of crop top-left
            y: Y coordinate of crop top-left
            crop_w: Crop width
            crop_h: Crop height
            sample_frames: Number of frames to sample (default: 50)

        Returns:
            PositionMetrics with calculated scores
        """
        # Pass 1: Get motion and complexity from showinfo
        cmd = [
            'ffmpeg', '-i', self.video_path,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},showinfo',
            '-frames:v', str(sample_frames),
            '-f', 'null', '-'
        ]
        stats_output = self._run_ffmpeg(cmd)

        # Extract motion (frame-to-frame differences in mean brightness)
        means = self._extract_metric_from_showinfo(stats_output, 'mean')
        motion = 0.0
        if len(means) > 1:
            diffs = [abs(means[i] - means[i-1]) for i in range(1, len(means))]
            motion = sum(diffs) / len(diffs) if diffs else 0.0

        # Extract complexity (standard deviation of pixel values)
        stdevs = self._extract_metric_from_showinfo(stats_output, 'stdev')
        complexity = sum(stdevs) / len(stdevs) if stdevs else 0.0

        # Pass 2: Get edge detection score
        cmd = [
            'ffmpeg', '-i', self.video_path,
            '-vf', f'crop={crop_w}:{crop_h}:{x}:{y},edgedetect=low=0.3:high=0.4:mode=colormix,showinfo',
            '-frames:v', str(sample_frames),
            '-f', 'null', '-'
        ]
        edge_output = self._run_ffmpeg(cmd)
        edge_means = self._extract_metric_from_showinfo(edge_output, 'mean')
        edges = sum(edge_means) / len(edge_means) if edge_means else 0.0

        # Extract color variance (sum of RGB channel standard deviations)
        stdev_pattern = r'stdev:\[([0-9. ]+)\]'
        stdev_matches = re.findall(stdev_pattern, stats_output)
        saturation = 0.0
        if stdev_matches:
            rgb_sums = []
            for match in stdev_matches:
                values = list(map(float, match.split()))
                if len(values) >= 3:
                    rgb_sums.append(sum(values[:3]))  # Sum R, G, B channels
            saturation = sum(rgb_sums) / len(rgb_sums) if rgb_sums else 0.0

        return PositionMetrics(
            x=x,
            y=y,
            motion=motion,
            complexity=complexity,
            edges=edges,
            saturation=saturation
        )

    def extract_frame(
        self,
        timestamp: float,
        output_path: str,
        x: int = 0,
        y: int = 0,
        crop_w: int = None,
        crop_h: int = None
    ) -> None:
        """
        Extract a single frame as JPEG.

        Args:
            timestamp: Time in seconds
            output_path: Where to save JPEG
            x: Crop X coordinate (default: 0)
            y: Crop Y coordinate (default: 0)
            crop_w: Crop width (None = full width)
            crop_h: Crop height (None = full height)
        """
        # Build video filter if crop is specified
        vf_parts = []
        if crop_w and crop_h:
            vf_parts.append(f'crop={crop_w}:{crop_h}:{x}:{y}')

        cmd = [
            'ffmpeg',
            '-ss', str(timestamp),  # Seek to timestamp
            '-i', self.video_path,
        ]

        if vf_parts:
            cmd.extend(['-vf', ','.join(vf_parts)])

        cmd.extend([
            '-vframes', '1',  # Extract 1 frame
            '-q:v', '2',      # High quality JPEG
            output_path,
            '-y'              # Overwrite existing
        ])

        subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    def _run_ffmpeg(self, cmd: List[str]) -> str:
        """
        Run FFmpeg command and return stderr output.

        FFmpeg writes progress/info to stderr, not stdout.

        Args:
            cmd: FFmpeg command as list of strings

        Returns:
            stderr output as string
        """
        result = subprocess.run(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        return result.stderr

    def _extract_metric_from_showinfo(self, output: str, metric: str) -> List[float]:
        """
        Extract metric values from FFmpeg showinfo filter output.

        The showinfo filter outputs metrics like:
        mean:[Y U V A] or stdev:[Y U V A]

        This extracts the Y (luma/brightness) channel value.

        Args:
            output: FFmpeg stderr output containing showinfo data
            metric: Metric name to extract (e.g., 'mean', 'stdev')

        Returns:
            List of float values for the metric (one per frame)
        """
        pattern = rf'{metric}:\[([0-9. ]+)\]'
        matches = re.findall(pattern, output)
        values = []
        for match in matches:
            # Take first value from each match (Y channel in YUV)
            parts = match.split()
            if parts:
                values.append(float(parts[0]))
        return values
