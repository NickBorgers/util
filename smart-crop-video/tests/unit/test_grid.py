"""
Unit tests for smart_crop.core.grid module.

Tests all pure functions for position grid generation.
"""
import pytest
from smart_crop.core.grid import (
    generate_analysis_grid,
    get_grid_center_position,
    get_grid_corner_positions,
    Position
)


class TestGenerateAnalysisGrid:
    """Tests for generate_analysis_grid function"""

    def test_generate_5x5_grid(self):
        """Test default 5x5 grid generation"""
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=5)

        assert len(positions) == 25  # 5x5 grid
        assert all(isinstance(p, Position) for p in positions)

    def test_grid_contains_corners(self):
        """Test that grid includes corner positions"""
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=5)

        # Check that corners are present
        assert Position(1, 1) in positions  # Top-left (starts from 1, not 0)
        assert Position(400, 300) in positions  # Bottom-right

    def test_grid_row_major_order(self):
        """Test that positions are returned in row-major order"""
        positions = generate_analysis_grid(max_x=400, max_y=200, grid_size=3)

        # Should be 3x3 = 9 positions
        assert len(positions) == 9

        # First row should all have y=1
        assert all(p.y == 1 for p in positions[0:3])
        # Second row should all have y=100 (middle)
        assert all(p.y == 100 for p in positions[3:6])
        # Third row should all have y=200 (bottom)
        assert all(p.y == 200 for p in positions[6:9])

    def test_grid_3x3(self):
        """Test 3x3 grid generation"""
        positions = generate_analysis_grid(max_x=400, max_y=400, grid_size=3)

        assert len(positions) == 9  # 3x3 grid

        # Check some key positions
        x_coords = sorted(set(p.x for p in positions))
        y_coords = sorted(set(p.y for p in positions))

        assert len(x_coords) == 3  # 3 distinct x values
        assert len(y_coords) == 3  # 3 distinct y values

    def test_grid_uniform_spacing(self):
        """Test that grid positions are uniformly spaced"""
        positions = generate_analysis_grid(max_x=400, max_y=400, grid_size=5)

        x_coords = sorted(set(p.x for p in positions))

        # Check spacing between x coordinates
        spacings = [x_coords[i+1] - x_coords[i] for i in range(len(x_coords)-1)]

        # All spacings should be within 1 pixel of each other (rounding)
        assert all(abs(s - spacings[0]) <= 1 for s in spacings)

    def test_no_movement_needed_returns_single_position(self):
        """Test that when crop fits exactly, returns single position at (0, 0)"""
        positions = generate_analysis_grid(max_x=0, max_y=0)

        assert len(positions) == 1
        assert positions[0] == Position(0, 0)

    def test_no_horizontal_movement_only(self):
        """Test grid when only vertical movement is possible"""
        positions = generate_analysis_grid(max_x=0, max_y=300, grid_size=3)

        # Should have 3x3 = 9 positions
        assert len(positions) == 9

        # All x coordinates should be 0
        assert all(p.x == 0 for p in positions)

        # Should have 3 distinct y values
        y_coords = sorted(set(p.y for p in positions))
        assert len(y_coords) == 3

    def test_no_vertical_movement_only(self):
        """Test grid when only horizontal movement is possible"""
        positions = generate_analysis_grid(max_x=400, max_y=0, grid_size=3)

        # Should have 3x3 = 9 positions
        assert len(positions) == 9

        # All y coordinates should be 0
        assert all(p.y == 0 for p in positions)

        # Should have 3 distinct x values
        x_coords = sorted(set(p.x for p in positions))
        assert len(x_coords) == 3

    def test_avoids_zero_position_when_possible(self):
        """Test that grid starts from 1, not 0, to avoid edge artifacts"""
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=5)

        # First position should be (1, 1), not (0, 0)
        # (unless no movement is possible)
        min_x = min(p.x for p in positions)
        min_y = min(p.y for p in positions)

        assert min_x >= 1
        assert min_y >= 1

    def test_large_grid_10x10(self):
        """Test larger 10x10 grid"""
        positions = generate_analysis_grid(max_x=1000, max_y=800, grid_size=10)

        assert len(positions) == 100  # 10x10

    def test_single_position_grid(self):
        """Test 1x1 grid (single position)"""
        positions = generate_analysis_grid(max_x=400, max_y=300, grid_size=1)

        assert len(positions) == 1
        # Single position should be at center
        assert positions[0] == Position(200, 150)  # Center of 400, 300

    def test_invalid_grid_size_zero(self):
        """Test that grid_size of 0 raises ValueError"""
        with pytest.raises(ValueError, match="Grid size must be at least 1"):
            generate_analysis_grid(max_x=400, max_y=300, grid_size=0)

    def test_invalid_grid_size_negative(self):
        """Test that negative grid_size raises ValueError"""
        with pytest.raises(ValueError, match="Grid size must be at least 1"):
            generate_analysis_grid(max_x=400, max_y=300, grid_size=-5)

    def test_negative_max_values_treated_as_zero(self):
        """Test that negative max values are handled gracefully"""
        # Should treat as "no movement possible"
        positions = generate_analysis_grid(max_x=-10, max_y=-10)

        assert len(positions) == 1
        assert positions[0] == Position(0, 0)

    def test_very_small_movement_range(self):
        """Test grid generation with very small movement range"""
        positions = generate_analysis_grid(max_x=10, max_y=10, grid_size=5)

        assert len(positions) == 25
        # Even with small range, should have 5 distinct x and y values
        x_coords = set(p.x for p in positions)
        y_coords = set(p.y for p in positions)

        # May have fewer than 5 due to rounding, but should have at least 2
        assert len(x_coords) >= 2
        assert len(y_coords) >= 2


class TestGetGridCenterPosition:
    """Tests for get_grid_center_position function"""

    def test_center_of_square_area(self):
        """Test center of square movement area"""
        center = get_grid_center_position(400, 400)

        assert center == Position(200, 200)

    def test_center_of_rectangular_area(self):
        """Test center of rectangular movement area"""
        center = get_grid_center_position(400, 300)

        assert center == Position(200, 150)

    def test_center_of_no_movement_area(self):
        """Test center when no movement is possible"""
        center = get_grid_center_position(0, 0)

        assert center == Position(0, 0)

    def test_center_with_odd_dimensions(self):
        """Test center calculation with odd dimensions"""
        center = get_grid_center_position(401, 301)

        # Integer division should round down
        assert center == Position(200, 150)

    def test_center_return_type(self):
        """Test that return value is Position dataclass"""
        center = get_grid_center_position(400, 300)

        assert isinstance(center, Position)


class TestGetGridCornerPositions:
    """Tests for get_grid_corner_positions function"""

    def test_corner_positions_count(self):
        """Test that exactly 4 corners are returned"""
        corners = get_grid_corner_positions(400, 300)

        assert len(corners) == 4

    def test_corner_positions_values(self):
        """Test that corners are at correct positions"""
        corners = get_grid_corner_positions(400, 300)

        assert Position(0, 0) in corners      # Top-left
        assert Position(400, 0) in corners    # Top-right
        assert Position(0, 300) in corners    # Bottom-left
        assert Position(400, 300) in corners  # Bottom-right

    def test_corner_positions_order(self):
        """Test that corners are in expected order"""
        corners = get_grid_corner_positions(400, 300)

        assert corners[0] == Position(0, 0)      # Top-left
        assert corners[1] == Position(400, 0)    # Top-right
        assert corners[2] == Position(0, 300)    # Bottom-left
        assert corners[3] == Position(400, 300)  # Bottom-right

    def test_corner_positions_no_movement(self):
        """Test corners when no movement is possible"""
        corners = get_grid_corner_positions(0, 0)

        # All corners should be at (0, 0)
        assert all(c == Position(0, 0) for c in corners)

    def test_corner_positions_return_type(self):
        """Test that all returned values are Position dataclasses"""
        corners = get_grid_corner_positions(400, 300)

        assert all(isinstance(c, Position) for c in corners)


class TestPosition:
    """Tests for Position dataclass"""

    def test_position_creation(self):
        """Test Position creation"""
        pos = Position(100, 200)

        assert pos.x == 100
        assert pos.y == 200

    def test_position_equality(self):
        """Test Position equality comparison"""
        pos1 = Position(100, 200)
        pos2 = Position(100, 200)
        pos3 = Position(200, 100)

        assert pos1 == pos2
        assert pos1 != pos3

    def test_position_in_list(self):
        """Test Position membership in list"""
        positions = [Position(100, 100), Position(200, 200)]

        assert Position(100, 100) in positions
        assert Position(300, 300) not in positions
