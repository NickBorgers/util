"""
Parallel video analysis using multiprocessing.

This module provides parallel analysis of multiple crop positions using
Python's multiprocessing module. This can provide 4-8x speedup on typical
systems by analyzing positions concurrently instead of sequentially.

Key benefits:
- Utilizes all CPU cores
- Maintains same results as sequential analysis
- Optional progress callbacks
- Graceful fallback to sequential if needed
"""
from multiprocessing import Pool, cpu_count
from typing import List, Callable, Optional
from smart_crop.core.grid import Position
from smart_crop.core.scoring import PositionMetrics


def _analyze_single_position(args):
    """
    Worker function for multiprocessing.

    This must be a top-level function (not a lambda or nested function)
    so it can be pickled by multiprocessing.

    Args:
        args: Tuple of (video_path, x, y, crop_w, crop_h, sample_frames)

    Returns:
        PositionMetrics for the analyzed position
    """
    video_path, x, y, crop_w, crop_h, sample_frames = args

    # Import here to avoid pickling issues with the class
    # Each worker process will import this independently
    from smart_crop.analysis.ffmpeg import FFmpegAnalyzer

    analyzer = FFmpegAnalyzer(video_path)
    return analyzer.analyze_position(x, y, crop_w, crop_h, sample_frames)


def analyze_positions_parallel(
    video_path: str,
    positions: List[Position],
    crop_w: int,
    crop_h: int,
    sample_frames: int = 50,
    max_workers: Optional[int] = None,
    progress_callback: Optional[Callable[[int, int], None]] = None
) -> List[PositionMetrics]:
    """
    Analyze multiple positions in parallel using multiprocessing.

    This function distributes position analysis across multiple CPU cores
    for significant speedup. On a typical 8-core system, analyzing 25
    positions can be 4-8x faster than sequential analysis.

    Args:
        video_path: Path to video file
        positions: List of Position objects to analyze
        crop_w: Width of crop window
        crop_h: Height of crop window
        sample_frames: Number of frames to sample per position (default: 50)
        max_workers: Maximum number of worker processes. If None, uses
                    min(cpu_count(), len(positions)). Use 1 for sequential.
        progress_callback: Optional function(current, total) called after
                          each position completes. Useful for progress bars.

    Returns:
        List of PositionMetrics in same order as input positions

    Examples:
        >>> from smart_crop.core.grid import generate_analysis_grid
        >>> positions = generate_analysis_grid(400, 300, grid_size=5)
        >>> # Parallel analysis
        >>> results = analyze_positions_parallel(
        ...     "video.mp4", positions, 640, 640,
        ...     max_workers=4,
        ...     progress_callback=lambda c, t: print(f"{c}/{t}")
        ... )
        >>> len(results)
        25

        >>> # Sequential analysis (for debugging)
        >>> results = analyze_positions_parallel(
        ...     "video.mp4", positions, 640, 640,
        ...     max_workers=1
        ... )

    Notes:
        - Returns results in same order as input positions
        - Progress callback is called after each completion (unordered)
        - Uses imap for better progress tracking
        - Falls back to sequential if max_workers=1
    """
    # Handle empty list early
    if not positions:
        return []

    # Determine number of workers
    if max_workers is None:
        # Use all cores, but not more than positions to analyze
        max_workers = min(cpu_count(), len(positions))

    # Validate inputs
    if max_workers < 1:
        raise ValueError(f"max_workers must be at least 1, got {max_workers}")

    # Prepare arguments for each worker
    # Each worker needs: (video_path, x, y, crop_w, crop_h, sample_frames)
    args_list = [
        (video_path, pos.x, pos.y, crop_w, crop_h, sample_frames)
        for pos in positions
    ]

    # Sequential execution (max_workers=1 or single position)
    if max_workers == 1 or len(positions) == 1:
        results = []
        for i, args in enumerate(args_list):
            result = _analyze_single_position(args)
            results.append(result)
            if progress_callback:
                progress_callback(i + 1, len(args_list))
        return results

    # Parallel execution with multiprocessing
    with Pool(processes=max_workers) as pool:
        if progress_callback:
            # Use imap for progress updates
            # imap returns results as they complete, maintaining order
            results = []
            for i, result in enumerate(pool.imap(_analyze_single_position, args_list)):
                results.append(result)
                progress_callback(i + 1, len(args_list))
            return results
        else:
            # Use map for simplicity (blocks until all complete)
            return pool.map(_analyze_single_position, args_list)


def analyze_positions_parallel_with_analyzer(
    analyzer,
    positions: List[Position],
    crop_w: int,
    crop_h: int,
    sample_frames: int = 50,
    progress_callback: Optional[Callable[[int, int], None]] = None
) -> List[PositionMetrics]:
    """
    Analyze positions using a provided analyzer (supports MockAnalyzer for testing).

    This function is designed for testing and scenarios where you already
    have an analyzer instance. It analyzes sequentially since we can't
    pickle analyzer instances for multiprocessing.

    For production use with FFmpeg, use analyze_positions_parallel() instead,
    which handles multiprocessing automatically.

    Args:
        analyzer: VideoAnalyzer instance (FFmpegAnalyzer or MockAnalyzer)
        positions: List of positions to analyze
        crop_w: Crop width
        crop_h: Crop height
        sample_frames: Frames to sample per position
        progress_callback: Optional progress callback function

    Returns:
        List of PositionMetrics in same order as positions

    Examples:
        >>> from tests.mocks.mock_analyzer import MockAnalyzer
        >>> mock = MockAnalyzer()
        >>> positions = [Position(100, 100), Position(200, 200)]
        >>> results = analyze_positions_parallel_with_analyzer(
        ...     mock, positions, 640, 640
        ... )
        >>> len(results)
        2
    """
    if not positions:
        return []

    results = []
    for i, pos in enumerate(positions):
        result = analyzer.analyze_position(pos.x, pos.y, crop_w, crop_h, sample_frames)
        results.append(result)
        if progress_callback:
            progress_callback(i + 1, len(positions))

    return results


def get_optimal_worker_count(position_count: int, max_workers: Optional[int] = None) -> int:
    """
    Calculate optimal number of workers for a given position count.

    This balances CPU utilization against multiprocessing overhead.
    For small numbers of positions, using all cores may not be beneficial
    due to process startup overhead.

    Args:
        position_count: Number of positions to analyze
        max_workers: Optional maximum worker count

    Returns:
        Recommended number of workers (at least 1)

    Examples:
        >>> get_optimal_worker_count(1)
        1
        >>> get_optimal_worker_count(100)  # On 8-core system
        8
        >>> get_optimal_worker_count(4, max_workers=2)
        2
    """
    if position_count <= 0:
        return 1

    # Get available CPU count
    available_cpus = cpu_count()

    # For very small counts, sequential may be faster due to overhead
    if position_count == 1:
        return 1
    elif position_count <= 3:
        # 2-3 positions: use 2 workers if available
        workers = min(2, available_cpus)
    else:
        # Use all available CPUs, but not more than positions
        workers = min(available_cpus, position_count)

    # Apply max_workers limit if specified
    if max_workers is not None:
        workers = min(workers, max_workers)

    return max(1, workers)


class ProgressTracker:
    """
    Simple progress tracker for parallel analysis.

    Useful for displaying progress in both CLI and web UI.

    Examples:
        >>> tracker = ProgressTracker(total=25)
        >>> def callback(current, total):
        ...     tracker.update(current, total)
        ...     print(f"Progress: {tracker.percent}%")
        >>> analyze_positions_parallel(
        ...     "video.mp4", positions, 640, 640,
        ...     progress_callback=callback
        ... )
    """

    def __init__(self, total: int):
        """
        Initialize progress tracker.

        Args:
            total: Total number of items to track
        """
        self.total = total
        self.current = 0
        self.percent = 0

    def update(self, current: int, total: int = None):
        """
        Update progress.

        Args:
            current: Current progress count
            total: Optional updated total (if it changed)
        """
        if total is not None:
            self.total = total
        self.current = current
        self.percent = int((current / self.total) * 100) if self.total > 0 else 0

    def is_complete(self) -> bool:
        """Check if progress is complete."""
        return self.current >= self.total

    def __str__(self) -> str:
        """String representation of progress."""
        return f"{self.current}/{self.total} ({self.percent}%)"
