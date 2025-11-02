"""
Candidate generation for crop position selection.

This module generates a diverse set of candidate crop positions using multiple
scoring strategies and spatial diversity. The goal is to provide 10 high-quality
candidates that cover different regions and represent different scoring approaches.

Candidate Generation Process:
1. Strategy-based candidates: Top 5 positions from each of 5 scoring strategies (25 candidates)
2. Spatial diversity: Best position from each quadrant + center using Balanced strategy (5 candidates)
3. Deduplication: Remove duplicate positions, keeping highest scored version
4. Selection: Return top 10 unique candidates sorted by score

This approach ensures candidates are both high-scoring and spatially diverse.
"""
from typing import List, Callable, Tuple, Set
from dataclasses import dataclass
from smart_crop.core.scoring import PositionMetrics, NormalizationBounds, score_position


@dataclass
class ScoredCandidate:
    """
    A crop position with its score and the strategy that selected it.

    Attributes:
        x: X coordinate of crop position
        y: Y coordinate of crop position
        score: Normalized score (0-100 range)
        strategy: Name of strategy or spatial region that selected this candidate
    """
    x: int
    y: int
    score: float
    strategy: str


def generate_strategy_candidates(
    positions: List[PositionMetrics],
    bounds: NormalizationBounds,
    strategy: str,
    top_n: int = 5
) -> List[ScoredCandidate]:
    """
    Generate top N candidates using a specific scoring strategy.

    Scores all positions using the given strategy and returns the top N
    highest-scoring positions as ScoredCandidate objects.

    Args:
        positions: List of analyzed positions with metrics
        bounds: Normalization bounds for scoring
        strategy: Strategy name (e.g., 'Motion Priority', 'Balanced')
        top_n: Number of top candidates to return (default: 5)

    Returns:
        List of ScoredCandidate objects, sorted by score descending

    Raises:
        ValueError: If positions list is empty or top_n < 1

    Examples:
        >>> positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0), ...]
        >>> bounds = NormalizationBounds.from_positions(positions)
        >>> candidates = generate_strategy_candidates(
        ...     positions, bounds, 'Motion Priority', top_n=5
        ... )
        >>> len(candidates)
        5
        >>> candidates[0].strategy
        'Motion Priority'
        >>> candidates[0].score > candidates[1].score
        True
    """
    if not positions:
        raise ValueError("Cannot generate candidates from empty positions list")

    if top_n < 1:
        raise ValueError(f"top_n must be at least 1, got {top_n}")

    # Score all positions
    scored = [
        (pos, score_position(pos, bounds, strategy))
        for pos in positions
    ]

    # Sort by score descending
    scored.sort(key=lambda x: x[1], reverse=True)

    # Take top N and convert to ScoredCandidate
    candidates = [
        ScoredCandidate(pos.x, pos.y, score, strategy)
        for pos, score in scored[:top_n]
    ]

    return candidates


def generate_spatial_candidates(
    positions: List[PositionMetrics],
    bounds: NormalizationBounds,
    video_width: int,
    video_height: int
) -> List[ScoredCandidate]:
    """
    Generate spatially diverse candidates from different regions of the video.

    Divides the video into 5 spatial regions:
    - Top-Left quadrant
    - Top-Right quadrant
    - Bottom-Left quadrant
    - Bottom-Right quadrant
    - Center region (within 1/4 of width/height from center)

    Finds the best candidate in each region using the Balanced strategy.
    This ensures spatial diversity in the candidate set.

    Args:
        positions: List of analyzed positions with metrics
        bounds: Normalization bounds for scoring
        video_width: Width of video in pixels
        video_height: Height of video in pixels

    Returns:
        List of ScoredCandidate objects (up to 5, one per region)
        Regions with no positions return no candidate

    Raises:
        ValueError: If positions list is empty
        ValueError: If video dimensions are invalid (<= 0)

    Examples:
        >>> positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0), ...]
        >>> bounds = NormalizationBounds.from_positions(positions)
        >>> candidates = generate_spatial_candidates(
        ...     positions, bounds, 1920, 1080
        ... )
        >>> len(candidates) <= 5
        True
        >>> all(c.strategy.startswith('Spatial:') for c in candidates)
        True
    """
    if not positions:
        raise ValueError("Cannot generate spatial candidates from empty positions list")

    if video_width <= 0 or video_height <= 0:
        raise ValueError(f"Invalid video dimensions: {video_width}x{video_height}")

    center_x = video_width // 2
    center_y = video_height // 2

    # Define spatial regions as (name, filter_function) tuples
    regions: List[Tuple[str, Callable[[PositionMetrics], bool]]] = [
        ('Top-Left', lambda p: p.x < center_x and p.y < center_y),
        ('Top-Right', lambda p: p.x >= center_x and p.y < center_y),
        ('Bottom-Left', lambda p: p.x < center_x and p.y >= center_y),
        ('Bottom-Right', lambda p: p.x >= center_x and p.y >= center_y),
        ('Center', lambda p: abs(p.x - center_x) < video_width // 4 and
                             abs(p.y - center_y) < video_height // 4),
    ]

    candidates = []

    for region_name, region_filter in regions:
        # Filter positions in this region
        region_positions = [p for p in positions if region_filter(p)]

        if not region_positions:
            # No positions in this region, skip
            continue

        # Score all positions in region using Balanced strategy
        scored = [
            (pos, score_position(pos, bounds, 'Balanced'))
            for pos in region_positions
        ]

        # Find best position in region
        scored.sort(key=lambda x: x[1], reverse=True)
        pos, score = scored[0]

        # Add as candidate
        candidate = ScoredCandidate(
            pos.x, pos.y, score, f"Spatial:{region_name}"
        )
        candidates.append(candidate)

    return candidates


def deduplicate_candidates(
    candidates: List[ScoredCandidate],
    max_candidates: int = 10
) -> List[ScoredCandidate]:
    """
    Deduplicate candidates and return top N by score.

    When multiple candidates have the same (x, y) position, keeps the one
    with the highest score. Returns up to max_candidates unique positions,
    sorted by score descending.

    Args:
        candidates: List of ScoredCandidate objects (may contain duplicates)
        max_candidates: Maximum number of unique candidates to return (default: 10)

    Returns:
        List of unique ScoredCandidate objects, sorted by score descending
        Length is min(unique_positions, max_candidates)

    Examples:
        >>> candidates = [
        ...     ScoredCandidate(100, 100, 95.0, 'Motion Priority'),
        ...     ScoredCandidate(100, 100, 90.0, 'Balanced'),  # Duplicate position
        ...     ScoredCandidate(200, 200, 85.0, 'Visual Detail'),
        ... ]
        >>> unique = deduplicate_candidates(candidates, max_candidates=10)
        >>> len(unique)
        2
        >>> unique[0].x == 100 and unique[0].score == 95.0
        True

    Notes:
        - Positions with x=0 or y=0 are excluded (invalid crop positions)
        - When duplicates exist, the highest-scoring version is kept
        - Candidates are sorted by score before deduplication, so ties favor
          whichever strategy scored it higher
    """
    if not candidates:
        return []

    if max_candidates < 1:
        raise ValueError(f"max_candidates must be at least 1, got {max_candidates}")

    # Sort all candidates by score descending
    sorted_candidates = sorted(candidates, key=lambda c: c.score, reverse=True)

    # Track seen positions and build unique list
    seen: Set[Tuple[int, int]] = set()
    unique_candidates = []

    for candidate in sorted_candidates:
        # Skip invalid positions (at edges)
        if candidate.x <= 0 or candidate.y <= 0:
            continue

        # Check if we've seen this position
        key = (candidate.x, candidate.y)
        if key in seen:
            continue  # Skip duplicate

        # Add to unique list
        seen.add(key)
        unique_candidates.append(candidate)

        # Stop if we have enough
        if len(unique_candidates) >= max_candidates:
            break

    return unique_candidates


def generate_candidates(
    positions: List[PositionMetrics],
    bounds: NormalizationBounds,
    video_width: int,
    video_height: int,
    max_candidates: int = 10,
    top_per_strategy: int = 5
) -> List[ScoredCandidate]:
    """
    Generate a diverse set of crop position candidates.

    This is the main entry point for candidate generation. It orchestrates
    the full process:

    1. Generates top 5 candidates from each of 5 scoring strategies (25 candidates)
    2. Generates best candidate from each spatial region (up to 5 candidates)
    3. Deduplicates all candidates by position
    4. Returns top 10 unique candidates sorted by score

    The result is a set of high-quality, spatially diverse candidates that
    represent different scoring priorities.

    Args:
        positions: List of analyzed positions with metrics
        bounds: Normalization bounds for scoring
        video_width: Width of video in pixels
        video_height: Height of video in pixels
        max_candidates: Maximum number of candidates to return (default: 10)
        top_per_strategy: Number of candidates per strategy (default: 5)

    Returns:
        List of unique ScoredCandidate objects (up to max_candidates)
        Sorted by score descending

    Raises:
        ValueError: If positions list is empty
        ValueError: If video dimensions are invalid
        ValueError: If max_candidates or top_per_strategy < 1

    Examples:
        >>> positions = [PositionMetrics(...), ...]  # 25 analyzed positions
        >>> bounds = NormalizationBounds.from_positions(positions)
        >>> candidates = generate_candidates(
        ...     positions, bounds, 1920, 1080,
        ...     max_candidates=10, top_per_strategy=5
        ... )
        >>> len(candidates) <= 10
        True
        >>> candidates[0].score >= candidates[-1].score
        True
        >>> len(set((c.x, c.y) for c in candidates)) == len(candidates)
        True

    Notes:
        - Total candidates before deduplication: ~30 (25 from strategies + 5 from spatial)
        - Deduplication typically reduces to 10-20 unique positions
        - Final selection takes top max_candidates (default 10)
        - Strategies used: Subject Detection, Motion Priority, Visual Detail,
          Balanced, Color Focus
    """
    if not positions:
        raise ValueError("Cannot generate candidates from empty positions list")

    if video_width <= 0 or video_height <= 0:
        raise ValueError(f"Invalid video dimensions: {video_width}x{video_height}")

    if max_candidates < 1:
        raise ValueError(f"max_candidates must be at least 1, got {max_candidates}")

    if top_per_strategy < 1:
        raise ValueError(f"top_per_strategy must be at least 1, got {top_per_strategy}")

    all_candidates = []

    # Define the 5 scoring strategies to use
    strategies = [
        'Subject Detection',
        'Motion Priority',
        'Visual Detail',
        'Balanced',
        'Color Focus'
    ]

    # Generate top candidates from each strategy
    for strategy in strategies:
        strategy_candidates = generate_strategy_candidates(
            positions, bounds, strategy, top_n=top_per_strategy
        )
        all_candidates.extend(strategy_candidates)

    # Generate spatially diverse candidates
    spatial_candidates = generate_spatial_candidates(
        positions, bounds, video_width, video_height
    )
    all_candidates.extend(spatial_candidates)

    # Deduplicate and select top N
    unique_candidates = deduplicate_candidates(all_candidates, max_candidates)

    return unique_candidates
