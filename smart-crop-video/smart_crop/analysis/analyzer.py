"""
Abstract interface for video analysis.

This module defines the VideoAnalyzer abstract base class, which provides
a contract for all video analysis implementations. This abstraction allows:

1. Testing without FFmpeg (using mocks)
2. Multiple implementations (FFmpeg, GPU, cloud, etc.)
3. Dependency injection for better testability
4. Easy swapping of implementations

Any concrete analyzer (FFmpegAnalyzer, MockAnalyzer, etc.) must implement
all abstract methods defined here.
"""
from abc import ABC, abstractmethod
from typing import Tuple
from smart_crop.core.scoring import PositionMetrics


class VideoAnalyzer(ABC):
    """
    Abstract base class for video analysis.

    Defines the interface that all video analyzers must implement.
    This enables dependency injection and makes business logic testable
    without requiring actual video files or FFmpeg.

    Implementations must be able to:
    - Provide video metadata (dimensions, duration, fps)
    - Analyze crop positions (motion, complexity, edges, saturation)
    - Extract frames for preview generation
    """

    @abstractmethod
    def get_dimensions(self) -> Tuple[int, int]:
        """
        Get video dimensions.

        Returns:
            Tuple of (width, height) in pixels

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> width, height = analyzer.get_dimensions()
            >>> print(f"{width}x{height}")
            1920x1080
        """
        pass

    @abstractmethod
    def get_duration(self) -> float:
        """
        Get video duration.

        Returns:
            Duration in seconds

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> duration = analyzer.get_duration()
            >>> print(f"{duration:.1f}s")
            30.5s
        """
        pass

    @abstractmethod
    def get_fps(self) -> float:
        """
        Get video frame rate.

        Returns:
            Frames per second (fps)

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> fps = analyzer.get_fps()
            >>> print(f"{fps} fps")
            30.0 fps
        """
        pass

    @abstractmethod
    def get_frame_count(self) -> int:
        """
        Get estimated total frame count.

        Returns:
            Estimated number of frames (duration × fps)

        Note:
            This may be an estimate based on duration and fps,
            as exact frame counting can be slow for large videos.

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> frames = analyzer.get_frame_count()
            >>> print(f"{frames} frames")
            915 frames
        """
        pass

    @abstractmethod
    def analyze_position(
        self,
        x: int,
        y: int,
        crop_w: int,
        crop_h: int,
        sample_frames: int = 50
    ) -> PositionMetrics:
        """
        Analyze visual metrics for a specific crop position.

        This is the core analysis method that evaluates how "interesting"
        a particular crop position is by measuring motion, visual complexity,
        edge content, and color saturation.

        Args:
            x: X coordinate of crop top-left corner
            y: Y coordinate of crop top-left corner
            crop_w: Width of crop window
            crop_h: Height of crop window
            sample_frames: Number of frames to sample for analysis (default: 50)
                          Lower values = faster but less accurate
                          Higher values = slower but more accurate

        Returns:
            PositionMetrics with:
                - x, y: Position coordinates (same as input)
                - motion: Frame-to-frame motion score (higher = more movement)
                - complexity: Visual complexity score (higher = more detail)
                - edges: Edge content score (higher = more edges/boundaries)
                - saturation: Color saturation score (higher = more colorful)

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> metrics = analyzer.analyze_position(
            ...     x=100, y=100, crop_w=640, crop_h=640, sample_frames=30
            ... )
            >>> print(f"Motion: {metrics.motion:.2f}")
            Motion: 5.23
            >>> print(f"Complexity: {metrics.complexity:.2f}")
            Complexity: 12.45
        """
        pass

    @abstractmethod
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
        Extract a single frame from the video and save as JPEG.

        Used for generating preview images and scene thumbnails.

        Args:
            timestamp: Time position in seconds to extract frame from
            output_path: Path where JPEG image should be saved
            x: X coordinate for crop (default: 0 = no crop)
            y: Y coordinate for crop (default: 0 = no crop)
            crop_w: Width of crop, or None for full width
            crop_h: Height of crop, or None for full height

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> # Extract full frame at 5 seconds
            >>> analyzer.extract_frame(5.0, "frame_5s.jpg")
            >>> # Extract cropped frame
            >>> analyzer.extract_frame(
            ...     5.0, "frame_5s_cropped.jpg",
            ...     x=100, y=100, crop_w=640, crop_h=640
            ... )
        """
        pass

    def get_video_info(self) -> dict:
        """
        Get all video metadata as a dictionary.

        This is a convenience method that combines all metadata getters.
        Not abstract - provides default implementation using other methods.

        Returns:
            Dictionary with keys: width, height, duration, fps, frame_count

        Examples:
            >>> analyzer = FFmpegAnalyzer("video.mp4")
            >>> info = analyzer.get_video_info()
            >>> print(info)
            {
                'width': 1920,
                'height': 1080,
                'duration': 30.5,
                'fps': 30.0,
                'frame_count': 915
            }
        """
        width, height = self.get_dimensions()
        return {
            'width': width,
            'height': height,
            'duration': self.get_duration(),
            'fps': self.get_fps(),
            'frame_count': self.get_frame_count()
        }
