"""
Position grid generation for crop analysis.

This module generates analysis grids - sets of positions where the crop window
should be analyzed to find the optimal crop location. All functions are pure
and deterministic.
"""
from typing import List
from dataclasses import dataclass


@dataclass
class Position:
    """
    A single position in the analysis grid.

    Represents an (x, y) coordinate where the top-left corner of the crop
    window will be placed for analysis.
    """
    x: int
    y: int


def generate_analysis_grid(
    max_x: int,
    max_y: int,
    grid_size: int = 5
) -> List[Position]:
    """
    Generate a uniform grid of positions to analyze.

    Creates a grid of evenly-spaced positions within the movement range
    of the crop window. This is a PURE function - same inputs always
    produce the same output.

    The grid uses 1-based positions (starts from 1, not 0) to avoid
    analyzing positions right at the edge, which may have artifacts
    or black bars.

    Args:
        max_x: Maximum x coordinate (video_width - crop_width).
               The crop can be positioned from 0 to max_x horizontally.
        max_y: Maximum y coordinate (video_height - crop_height).
               The crop can be positioned from 0 to max_y vertically.
        grid_size: Number of positions per dimension. Default is 5, which
                  creates a 5x5 grid (25 total positions).

    Returns:
        List of Position objects in row-major order (left-to-right, top-to-bottom).
        If there's no movement possible (max_x=0, max_y=0), returns a single
        position at (0, 0).

    Examples:
        >>> positions = generate_analysis_grid(400, 300, grid_size=3)
        >>> len(positions)
        9
        >>> positions[0]  # Top-left
        Position(x=1, y=1)
        >>> positions[-1]  # Bottom-right
        Position(x=400, y=300)

        >>> # No movement needed
        >>> positions = generate_analysis_grid(0, 0)
        >>> len(positions)
        1
        >>> positions[0]
        Position(x=0, y=0)
    """
    # Special case: crop fits exactly, no movement possible
    if max_x <= 0 and max_y <= 0:
        return [Position(0, 0)]

    # Validate grid size
    if grid_size < 1:
        raise ValueError(f"Grid size must be at least 1, got {grid_size}")

    # Generate evenly-spaced positions along each axis
    # Start from 1 (not 0) to avoid edge artifacts
    # Use integer division for precise positioning
    if grid_size == 1:
        # Single position - use center or max values
        x_positions = [max_x // 2 if max_x > 0 else 0]
        y_positions = [max_y // 2 if max_y > 0 else 0]
    else:
        if max_x > 0:
            x_positions = [max(1, max_x * i // (grid_size - 1)) for i in range(grid_size)]
        else:
            # No horizontal movement possible
            x_positions = [0] * grid_size

        if max_y > 0:
            y_positions = [max(1, max_y * i // (grid_size - 1)) for i in range(grid_size)]
        else:
            # No vertical movement possible
            y_positions = [0] * grid_size

    # Create all combinations in row-major order
    positions = []
    for y in y_positions:
        for x in x_positions:
            positions.append(Position(x, y))

    return positions


def get_grid_center_position(
    max_x: int,
    max_y: int
) -> Position:
    """
    Get the center position of the crop area.

    Useful for quick centering or as a fallback position.

    Args:
        max_x: Maximum x coordinate
        max_y: Maximum y coordinate

    Returns:
        Position at the center of the movement range

    Examples:
        >>> center = get_grid_center_position(400, 300)
        >>> center
        Position(x=200, y=150)
    """
    return Position(
        x=max_x // 2,
        y=max_y // 2
    )


def get_grid_corner_positions(
    max_x: int,
    max_y: int
) -> List[Position]:
    """
    Get the four corner positions of the crop area.

    Useful for quick analysis of corner positions.

    Args:
        max_x: Maximum x coordinate
        max_y: Maximum y coordinate

    Returns:
        List of 4 Position objects representing corners
        [top-left, top-right, bottom-left, bottom-right]

    Examples:
        >>> corners = get_grid_corner_positions(400, 300)
        >>> len(corners)
        4
        >>> corners[0]  # Top-left
        Position(x=0, y=0)
        >>> corners[-1]  # Bottom-right
        Position(x=400, y=300)
    """
    return [
        Position(0, 0),           # Top-left
        Position(max_x, 0),       # Top-right
        Position(0, max_y),       # Bottom-left
        Position(max_x, max_y),   # Bottom-right
    ]
