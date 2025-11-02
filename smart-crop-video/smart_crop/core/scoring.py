"""
Scoring strategies for crop position evaluation.

This module contains pure functions for scoring crop positions based on various
visual metrics. All functions are deterministic and have no I/O, making them
highly testable.
"""
from typing import Dict, List
from dataclasses import dataclass
import copy


@dataclass
class PositionMetrics:
    """Metrics for a single crop position"""
    x: int
    y: int
    motion: float         # Frame-to-frame differences
    complexity: float     # Visual complexity (standard deviation)
    edges: float          # Edge content
    saturation: float     # Color variance/saturation


@dataclass
class NormalizationBounds:
    """Min/max values for normalization of metrics"""
    motion_min: float
    motion_max: float
    complexity_min: float
    complexity_max: float
    edges_min: float
    edges_max: float
    saturation_min: float
    saturation_max: float

    @classmethod
    def from_positions(cls, positions: List[PositionMetrics]) -> 'NormalizationBounds':
        """
        Calculate normalization bounds from a list of positions.

        Args:
            positions: List of PositionMetrics to analyze

        Returns:
            NormalizationBounds with min/max for each metric

        Raises:
            ValueError: If positions list is empty

        Examples:
            >>> positions = [
            ...     PositionMetrics(0, 0, 1.0, 2.0, 3.0, 4.0),
            ...     PositionMetrics(100, 100, 5.0, 6.0, 7.0, 8.0),
            ... ]
            >>> bounds = NormalizationBounds.from_positions(positions)
            >>> bounds.motion_min, bounds.motion_max
            (1.0, 5.0)
        """
        if not positions:
            raise ValueError("Cannot calculate bounds from empty positions list")

        return cls(
            motion_min=min(p.motion for p in positions),
            motion_max=max(p.motion for p in positions),
            complexity_min=min(p.complexity for p in positions),
            complexity_max=max(p.complexity for p in positions),
            edges_min=min(p.edges for p in positions),
            edges_max=max(p.edges for p in positions),
            saturation_min=min(p.saturation for p in positions),
            saturation_max=max(p.saturation for p in positions),
        )


def normalize(value: float, min_val: float, max_val: float) -> float:
    """
    Normalize value to 0-100 range.

    This is a PURE function - perfect for unit testing!

    Args:
        value: Value to normalize
        min_val: Minimum value in the range
        max_val: Maximum value in the range

    Returns:
        Normalized value in 0-100 range. Returns 50.0 if min_val == max_val.

    Examples:
        >>> normalize(5, 0, 10)
        50.0
        >>> normalize(0, 0, 10)
        0.0
        >>> normalize(10, 0, 10)
        100.0
        >>> normalize(5, 5, 5)  # No range
        50.0
    """
    if max_val - min_val > 0:
        return ((value - min_val) / (max_val - min_val)) * 100
    return 50.0  # Default to middle if no range


# Strategy definitions - configuration, not code!
# Each strategy defines weights for the four metrics
STRATEGIES = {
    'Subject Detection': {
        'weights': {
            'motion': 0.05,
            'complexity': 0.25,
            'edges': 0.40,
            'saturation': 0.30
        },
        'description': 'Finds people/objects (40% edges, 30% saturation)',
        'use_case': 'Best for videos with people or distinct subjects'
    },
    'Motion Priority': {
        'weights': {
            'motion': 0.50,
            'complexity': 0.15,
            'edges': 0.25,
            'saturation': 0.10
        },
        'description': 'Tracks movement (50% motion, 25% edges)',
        'use_case': 'Best for action videos or when following moving subjects'
    },
    'Visual Detail': {
        'weights': {
            'motion': 0.05,
            'complexity': 0.50,
            'edges': 0.30,
            'saturation': 0.15
        },
        'description': 'Identifies complex areas (50% complexity, 30% edges)',
        'use_case': 'Best for detailed scenes, architecture, or texture-rich content'
    },
    'Balanced': {
        'weights': {
            'motion': 0.25,
            'complexity': 0.25,
            'edges': 0.25,
            'saturation': 0.25
        },
        'description': 'Equal weights (25% each metric)',
        'use_case': 'General purpose, no specific priority'
    },
    'Color Focus': {
        'weights': {
            'motion': 0.05,
            'complexity': 0.20,
            'edges': 0.30,
            'saturation': 0.45
        },
        'description': 'Colorful subjects (45% saturation, 30% edges)',
        'use_case': 'Best for vibrant, colorful content'
    },
}


def score_position(
    metrics: PositionMetrics,
    bounds: NormalizationBounds,
    strategy: str = 'Balanced'
) -> float:
    """
    Score a position using a specific strategy.

    This is a PURE function - testable without FFmpeg!

    Args:
        metrics: PositionMetrics for the crop position to score
        bounds: NormalizationBounds for scaling metrics to 0-100 range
        strategy: Name of scoring strategy to use. Must be a key in STRATEGIES.

    Returns:
        Score in 0-100 range (higher is better)

    Raises:
        ValueError: If strategy is not recognized

    Examples:
        >>> metrics = PositionMetrics(100, 100, 5.0, 10.0, 15.0, 20.0)
        >>> bounds = NormalizationBounds(
        ...     motion_min=0, motion_max=10,
        ...     complexity_min=0, complexity_max=20,
        ...     edges_min=0, edges_max=30,
        ...     saturation_min=0, saturation_max=40
        ... )
        >>> score = score_position(metrics, bounds, 'Balanced')
        >>> 0 <= score <= 100
        True
    """
    if strategy not in STRATEGIES:
        valid_strategies = ', '.join(STRATEGIES.keys())
        raise ValueError(
            f"Unknown strategy: '{strategy}'. "
            f"Valid strategies are: {valid_strategies}"
        )

    weights = STRATEGIES[strategy]['weights']

    # Normalize all metrics to 0-100 range
    motion_norm = normalize(metrics.motion, bounds.motion_min, bounds.motion_max)
    complexity_norm = normalize(metrics.complexity, bounds.complexity_min, bounds.complexity_max)
    edges_norm = normalize(metrics.edges, bounds.edges_min, bounds.edges_max)
    saturation_norm = normalize(metrics.saturation, bounds.saturation_min, bounds.saturation_max)

    # Calculate weighted sum
    score = (
        motion_norm * weights['motion'] +
        complexity_norm * weights['complexity'] +
        edges_norm * weights['edges'] +
        saturation_norm * weights['saturation']
    )

    return score


def get_available_strategies() -> List[str]:
    """
    Return list of available strategy names.

    Returns:
        List of strategy names that can be used with score_position()

    Examples:
        >>> strategies = get_available_strategies()
        >>> 'Balanced' in strategies
        True
        >>> 'Motion Priority' in strategies
        True
    """
    return list(STRATEGIES.keys())


def get_strategy_info(strategy: str) -> Dict[str, any]:
    """
    Get information about a scoring strategy.

    Args:
        strategy: Name of the strategy

    Returns:
        Dictionary with 'weights', 'description', and 'use_case' keys.
        Returns a deep copy to prevent modification of original strategies.

    Raises:
        ValueError: If strategy is not recognized

    Examples:
        >>> info = get_strategy_info('Balanced')
        >>> info['weights']['motion']
        0.25
        >>> 'Equal weights' in info['description']
        True
    """
    if strategy not in STRATEGIES:
        valid_strategies = ', '.join(STRATEGIES.keys())
        raise ValueError(
            f"Unknown strategy: '{strategy}'. "
            f"Valid strategies are: {valid_strategies}"
        )

    return copy.deepcopy(STRATEGIES[strategy])


def validate_strategy_weights(weights: Dict[str, float]) -> bool:
    """
    Validate that strategy weights are correct.

    Weights should sum to 1.0 and all be non-negative.

    Args:
        weights: Dictionary of metric weights

    Returns:
        True if weights are valid

    Raises:
        ValueError: If weights are invalid

    Examples:
        >>> validate_strategy_weights({'motion': 0.25, 'complexity': 0.25, 'edges': 0.25, 'saturation': 0.25})
        True
        >>> validate_strategy_weights({'motion': 0.5, 'complexity': 0.5, 'edges': 0.0, 'saturation': 0.0})
        True
    """
    required_keys = {'motion', 'complexity', 'edges', 'saturation'}

    # Check all keys present
    if set(weights.keys()) != required_keys:
        raise ValueError(
            f"Weights must have exactly these keys: {required_keys}. "
            f"Got: {set(weights.keys())}"
        )

    # Check all non-negative
    for key, value in weights.items():
        if value < 0:
            raise ValueError(f"Weight for '{key}' must be non-negative, got {value}")

    # Check sum to 1.0 (with small tolerance for floating point)
    total = sum(weights.values())
    if not (0.99 <= total <= 1.01):
        raise ValueError(
            f"Weights must sum to 1.0 (got {total:.3f}). "
            f"Current weights: {weights}"
        )

    return True


# Validate all built-in strategies on module load
for strategy_name, strategy_config in STRATEGIES.items():
    try:
        validate_strategy_weights(strategy_config['weights'])
    except ValueError as e:
        raise ValueError(f"Built-in strategy '{strategy_name}' has invalid weights: {e}")
