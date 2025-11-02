"""
Unit tests for smart_crop.core.dimensions module.

Tests all pure functions for crop dimension calculations.
"""
import pytest
from smart_crop.core.dimensions import (
    calculate_crop_dimensions,
    parse_aspect_ratio,
    CropDimensions
)


class TestParseAspectRatio:
    """Tests for parse_aspect_ratio function"""

    def test_parse_valid_square(self):
        """Test parsing 1:1 aspect ratio"""
        assert parse_aspect_ratio("1:1") == (1, 1)

    def test_parse_valid_landscape(self):
        """Test parsing landscape aspect ratios"""
        assert parse_aspect_ratio("16:9") == (16, 9)
        assert parse_aspect_ratio("4:3") == (4, 3)
        assert parse_aspect_ratio("21:9") == (21, 9)

    def test_parse_valid_portrait(self):
        """Test parsing portrait aspect ratios"""
        assert parse_aspect_ratio("9:16") == (9, 16)
        assert parse_aspect_ratio("4:5") == (4, 5)

    def test_parse_invalid_format_no_colon(self):
        """Test that invalid format raises ValueError"""
        with pytest.raises(ValueError, match="Invalid aspect ratio format"):
            parse_aspect_ratio("invalid")

    def test_parse_invalid_format_too_many_parts(self):
        """Test that too many parts raises ValueError"""
        with pytest.raises(ValueError, match="Invalid aspect ratio format"):
            parse_aspect_ratio("1:2:3")

    def test_parse_invalid_non_numeric(self):
        """Test that non-numeric values raise ValueError"""
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("a:b")
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("16:nine")

    def test_parse_invalid_zero_values(self):
        """Test that zero values raise ValueError"""
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("0:9")
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("16:0")

    def test_parse_invalid_negative_values(self):
        """Test that negative values raise ValueError"""
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("-16:9")
        with pytest.raises(ValueError, match="Invalid aspect ratio values"):
            parse_aspect_ratio("16:-9")


class TestCalculateCropDimensions:
    """Tests for calculate_crop_dimensions function"""

    def test_square_video_square_crop_full_scale(self):
        """Test 1:1 crop on square 1000x1000 video at full scale"""
        dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=1.0)

        assert dims.crop_w == 1000
        assert dims.crop_h == 1000
        assert dims.max_crop_w == 1000
        assert dims.max_crop_h == 1000
        assert dims.max_x == 0  # No movement possible
        assert dims.max_y == 0  # No movement possible

    def test_square_video_square_crop_scaled(self):
        """Test 1:1 crop on square 1000x1000 video at 75% scale"""
        dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=0.75)

        assert dims.crop_w == 750  # 1000 * 0.75
        assert dims.crop_h == 750
        assert dims.max_crop_w == 1000
        assert dims.max_crop_h == 1000
        assert dims.max_x == 250  # 1000 - 750
        assert dims.max_y == 250

    def test_landscape_to_portrait_crop(self):
        """Test 9:16 crop on 1920x1080 landscape video"""
        dims = calculate_crop_dimensions(1920, 1080, 9, 16, crop_scale=1.0)

        # Height is constraint (1080), calculate width
        max_w_unrounded = 1080 * 9 // 16  # 607
        expected_crop_w = max_w_unrounded - (max_w_unrounded % 2)  # 606 (made even)

        assert dims.max_crop_h == 1080
        assert dims.max_crop_w == max_w_unrounded  # 607 before rounding
        assert dims.crop_w == expected_crop_w  # 606 after making even
        assert dims.crop_h == 1080
        assert dims.crop_w % 2 == 0  # Even
        assert dims.crop_h % 2 == 0  # Even

    def test_portrait_to_square_crop(self):
        """Test 1:1 crop on 1080x1920 portrait video"""
        dims = calculate_crop_dimensions(1080, 1920, 1, 1, crop_scale=0.5)

        # Width is constraint (1080)
        assert dims.max_crop_w == 1080
        assert dims.max_crop_h == 1080
        assert dims.crop_w == 540  # 1080 * 0.5
        assert dims.crop_h == 540
        assert dims.max_x == 540  # 1080 - 540
        assert dims.max_y == 1380  # 1920 - 540

    def test_landscape_to_landscape_crop(self):
        """Test 16:9 crop on 1920x1080 landscape video (same aspect)"""
        dims = calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=1.0)

        # Should fit exactly
        assert dims.crop_w == 1920
        assert dims.crop_h == 1080
        assert dims.max_x == 0
        assert dims.max_y == 0

    def test_dimensions_always_even(self):
        """Ensure crop dimensions are always even (H.264 requirement)"""
        # Test with various scales that would produce odd numbers
        for scale in [0.755, 0.333, 0.666, 0.123]:
            dims = calculate_crop_dimensions(1000, 1000, 1, 1, crop_scale=scale)
            assert dims.crop_w % 2 == 0, f"Width not even at scale {scale}: {dims.crop_w}"
            assert dims.crop_h % 2 == 0, f"Height not even at scale {scale}: {dims.crop_h}"

    def test_movement_range_calculation(self):
        """Test that movement range is correctly calculated"""
        dims = calculate_crop_dimensions(1920, 1080, 1, 1, crop_scale=0.5)

        # Crop is 1080 * 0.5 = 540 (square, using height as constraint)
        assert dims.crop_w == 540
        assert dims.crop_h == 540
        assert dims.max_x == 1920 - 540  # 1380
        assert dims.max_y == 1080 - 540  # 540

    def test_4k_video_to_instagram_square(self):
        """Test realistic scenario: 4K video to 1:1 Instagram crop"""
        dims = calculate_crop_dimensions(3840, 2160, 1, 1, crop_scale=0.75)

        # Height is constraint (2160)
        expected_crop = int(2160 * 0.75)
        expected_crop = expected_crop - (expected_crop % 2)

        assert dims.crop_w == expected_crop
        assert dims.crop_h == expected_crop
        assert dims.max_x == 3840 - expected_crop
        assert dims.max_y == 2160 - expected_crop

    def test_4k_video_to_instagram_story(self):
        """Test realistic scenario: 4K video to 9:16 Instagram Story"""
        dims = calculate_crop_dimensions(3840, 2160, 9, 16, crop_scale=1.0)

        # Height is constraint
        expected_w = 2160 * 9 // 16
        expected_w = expected_w - (expected_w % 2)

        assert dims.crop_w == expected_w
        assert dims.crop_h == 2160

    def test_invalid_video_dimensions_zero(self):
        """Test that zero video dimensions raise ValueError"""
        with pytest.raises(ValueError, match="Video dimensions must be positive"):
            calculate_crop_dimensions(0, 1080, 16, 9)

        with pytest.raises(ValueError, match="Video dimensions must be positive"):
            calculate_crop_dimensions(1920, 0, 16, 9)

    def test_invalid_video_dimensions_negative(self):
        """Test that negative video dimensions raise ValueError"""
        with pytest.raises(ValueError, match="Video dimensions must be positive"):
            calculate_crop_dimensions(-1920, 1080, 16, 9)

        with pytest.raises(ValueError, match="Video dimensions must be positive"):
            calculate_crop_dimensions(1920, -1080, 16, 9)

    def test_invalid_aspect_ratio_zero(self):
        """Test that zero aspect ratio components raise ValueError"""
        with pytest.raises(ValueError, match="Aspect ratio components must be positive"):
            calculate_crop_dimensions(1920, 1080, 0, 9)

        with pytest.raises(ValueError, match="Aspect ratio components must be positive"):
            calculate_crop_dimensions(1920, 1080, 16, 0)

    def test_invalid_aspect_ratio_negative(self):
        """Test that negative aspect ratio components raise ValueError"""
        with pytest.raises(ValueError, match="Aspect ratio components must be positive"):
            calculate_crop_dimensions(1920, 1080, -16, 9)

        with pytest.raises(ValueError, match="Aspect ratio components must be positive"):
            calculate_crop_dimensions(1920, 1080, 16, -9)

    def test_invalid_crop_scale_zero(self):
        """Test that zero crop scale raises ValueError"""
        with pytest.raises(ValueError, match="Crop scale must be between 0 and 1"):
            calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=0.0)

    def test_invalid_crop_scale_negative(self):
        """Test that negative crop scale raises ValueError"""
        with pytest.raises(ValueError, match="Crop scale must be between 0 and 1"):
            calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=-0.5)

    def test_invalid_crop_scale_too_large(self):
        """Test that crop scale > 1.0 raises ValueError"""
        with pytest.raises(ValueError, match="Crop scale must be between 0 and 1"):
            calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=1.5)

    def test_very_small_scale(self):
        """Test that very small scale factors work correctly"""
        dims = calculate_crop_dimensions(1920, 1080, 1, 1, crop_scale=0.1)

        expected_size = int(1080 * 0.1)
        expected_size = expected_size - (expected_size % 2)

        assert dims.crop_w == expected_size
        assert dims.crop_h == expected_size
        assert dims.crop_w % 2 == 0
        assert dims.crop_h % 2 == 0

    def test_return_type_is_dataclass(self):
        """Test that return value is CropDimensions dataclass"""
        dims = calculate_crop_dimensions(1920, 1080, 16, 9, crop_scale=0.75)

        assert isinstance(dims, CropDimensions)
        assert hasattr(dims, 'crop_w')
        assert hasattr(dims, 'crop_h')
        assert hasattr(dims, 'max_crop_w')
        assert hasattr(dims, 'max_crop_h')
        assert hasattr(dims, 'max_x')
        assert hasattr(dims, 'max_y')

    def test_non_negative_movement_range(self):
        """Test that movement range is never negative"""
        # Even when crop is larger than video (shouldn't happen, but test defensive code)
        dims = calculate_crop_dimensions(100, 100, 1, 1, crop_scale=1.0)
        assert dims.max_x >= 0
        assert dims.max_y >= 0
