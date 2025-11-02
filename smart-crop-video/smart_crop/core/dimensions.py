"""
Crop dimension calculations - pure functions with no side effects.

This module contains pure mathematical functions for calculating crop dimensions
based on video size, aspect ratio, and scaling factors. All functions are
deterministic and have no I/O side effects, making them highly testable.
"""
from typing import Tuple
from dataclasses import dataclass


@dataclass
class CropDimensions:
    """Result of crop dimension calculation"""
    crop_w: int          # Final crop width (scaled and even)
    crop_h: int          # Final crop height (scaled and even)
    max_crop_w: int      # Maximum possible crop width
    max_crop_h: int      # Maximum possible crop height
    max_x: int           # Maximum X movement range
    max_y: int           # Maximum Y movement range


def parse_aspect_ratio(aspect_str: str) -> Tuple[int, int]:
    """
    Parse aspect ratio string like '1:1' or '16:9'.

    Args:
        aspect_str: Aspect ratio in format 'width:height' (e.g., '1:1', '16:9')

    Returns:
        Tuple of (aspect_width, aspect_height)

    Raises:
        ValueError: If aspect_str is not in valid format

    Examples:
        >>> parse_aspect_ratio("1:1")
        (1, 1)
        >>> parse_aspect_ratio("16:9")
        (16, 9)
    """
    parts = aspect_str.split(':')
    if len(parts) != 2:
        raise ValueError(f"Invalid aspect ratio format: {aspect_str}. Expected 'width:height' (e.g., '1:1', '16:9')")

    try:
        width = int(parts[0])
        height = int(parts[1])
    except ValueError:
        raise ValueError(f"Invalid aspect ratio values: {aspect_str}. Both values must be integers")

    if width <= 0 or height <= 0:
        raise ValueError(f"Invalid aspect ratio values: {aspect_str}. Both values must be positive")

    return width, height


def calculate_crop_dimensions(
    video_width: int,
    video_height: int,
    aspect_w: int,
    aspect_h: int,
    crop_scale: float = 0.75
) -> CropDimensions:
    """
    Calculate crop dimensions for a given video and aspect ratio.

    This is a PURE function - same inputs always produce same outputs.
    No I/O, no side effects - highly testable!

    The function calculates the maximum possible crop size that maintains
    the target aspect ratio, then applies a scale factor, and ensures
    dimensions are even (required for H.264 encoding).

    Args:
        video_width: Original video width in pixels
        video_height: Original video height in pixels
        aspect_w: Target aspect ratio width component
        aspect_h: Target aspect ratio height component
        crop_scale: Scale factor (0.0-1.0) to reduce crop size. Default 0.75
                   means crop will be 75% of maximum possible size

    Returns:
        CropDimensions with calculated values including crop size and movement range

    Raises:
        ValueError: If inputs are invalid (zero/negative dimensions or scale)

    Examples:
        >>> dims = calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=1.0)
        >>> dims.crop_w, dims.crop_h
        (1920, 1080)

        >>> dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=0.75)
        >>> dims.crop_w, dims.crop_h
        (750, 750)
    """
    # Validate inputs
    if video_width <= 0 or video_height <= 0:
        raise ValueError(f"Video dimensions must be positive: {video_width}x{video_height}")
    if aspect_w <= 0 or aspect_h <= 0:
        raise ValueError(f"Aspect ratio components must be positive: {aspect_w}:{aspect_h}")
    if not (0.0 < crop_scale <= 1.0):
        raise ValueError(f"Crop scale must be between 0 and 1: {crop_scale}")

    # Calculate maximum possible crop dimensions that maintain aspect ratio
    # Strategy: Use the constraining dimension (the one that would overflow first)
    if video_width < video_height:
        # Portrait or square - width is constraint
        max_crop_w = video_width
        max_crop_h = video_width * aspect_h // aspect_w
        if max_crop_h > video_height:
            # Height overflowed, use height as constraint instead
            max_crop_h = video_height
            max_crop_w = video_height * aspect_w // aspect_h
    else:
        # Landscape - height is typically constraint
        max_crop_h = video_height
        max_crop_w = video_height * aspect_w // aspect_h
        if max_crop_w > video_width:
            # Width overflowed, use width as constraint instead
            max_crop_w = video_width
            max_crop_h = video_width * aspect_h // aspect_w

    # Apply scale factor to reduce crop size if desired
    crop_w = int(max_crop_w * crop_scale)
    crop_h = int(max_crop_h * crop_scale)

    # Ensure even dimensions (required for H.264/H.265 encoding)
    # Many video codecs require even dimensions for chroma subsampling
    crop_w = crop_w - (crop_w % 2)
    crop_h = crop_h - (crop_h % 2)

    # Calculate movement range - how much we can move the crop window
    max_x = video_width - crop_w
    max_y = video_height - crop_h

    # Ensure non-negative movement range
    max_x = max(0, max_x)
    max_y = max(0, max_y)

    return CropDimensions(
        crop_w=crop_w,
        crop_h=crop_h,
        max_crop_w=max_crop_w,
        max_crop_h=max_crop_h,
        max_x=max_x,
        max_y=max_y
    )
