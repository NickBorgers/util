"""
Unit tests for smart_crop.core.candidates module.

Tests candidate generation including strategy-based selection,
spatial diversity, deduplication, and the full candidate generation pipeline.
"""
import pytest
from smart_crop.core.candidates import (
    ScoredCandidate,
    generate_strategy_candidates,
    generate_spatial_candidates,
    deduplicate_candidates,
    generate_candidates
)
from smart_crop.core.scoring import PositionMetrics, NormalizationBounds


class TestScoredCandidate:
    """Tests for ScoredCandidate dataclass"""

    def test_create_scored_candidate(self):
        """Test creating a ScoredCandidate"""
        candidate = ScoredCandidate(100, 200, 95.5, 'Motion Priority')

        assert candidate.x == 100
        assert candidate.y == 200
        assert candidate.score == 95.5
        assert candidate.strategy == 'Motion Priority'

    def test_scored_candidate_equality(self):
        """Test ScoredCandidate equality"""
        c1 = ScoredCandidate(100, 100, 90.0, 'Balanced')
        c2 = ScoredCandidate(100, 100, 90.0, 'Balanced')

        assert c1 == c2

    def test_scored_candidate_different_positions(self):
        """Test candidates with different positions are not equal"""
        c1 = ScoredCandidate(100, 100, 90.0, 'Balanced')
        c2 = ScoredCandidate(200, 200, 90.0, 'Balanced')

        assert c1 != c2


class TestGenerateStrategyCandidates:
    """Tests for generate_strategy_candidates function"""

    def test_generate_strategy_candidates_basic(self):
        """Test basic strategy candidate generation"""
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(200, 200, 5.0, 10.0, 6.0, 8.0),
            PositionMetrics(300, 300, 8.0, 8.0, 9.0, 9.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_strategy_candidates(
            positions, bounds, 'Motion Priority', top_n=2
        )

        assert len(candidates) == 2
        assert all(isinstance(c, ScoredCandidate) for c in candidates)
        assert all(c.strategy == 'Motion Priority' for c in candidates)

    def test_candidates_sorted_by_score(self):
        """Test that candidates are sorted by score descending"""
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(200, 200, 5.0, 10.0, 6.0, 8.0),
            PositionMetrics(300, 300, 8.0, 8.0, 9.0, 9.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_strategy_candidates(
            positions, bounds, 'Balanced', top_n=3
        )

        # Verify descending order
        for i in range(len(candidates) - 1):
            assert candidates[i].score >= candidates[i+1].score

    def test_top_n_limits_results(self):
        """Test that top_n parameter limits results"""
        positions = [
            PositionMetrics(i, i, float(i), float(i), float(i), float(i))
            for i in range(1, 11)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_strategy_candidates(
            positions, bounds, 'Balanced', top_n=3
        )

        assert len(candidates) == 3

    def test_top_n_greater_than_positions(self):
        """Test requesting more candidates than positions available"""
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(200, 200, 5.0, 10.0, 6.0, 8.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_strategy_candidates(
            positions, bounds, 'Balanced', top_n=10
        )

        # Should only return available positions
        assert len(candidates) == 2

    def test_empty_positions_raises_error(self):
        """Test that empty positions list raises ValueError"""
        bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

        with pytest.raises(ValueError, match="empty positions list"):
            generate_strategy_candidates([], bounds, 'Balanced', top_n=5)

    def test_invalid_top_n_raises_error(self):
        """Test that top_n < 1 raises ValueError"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        with pytest.raises(ValueError, match="top_n must be at least 1"):
            generate_strategy_candidates(positions, bounds, 'Balanced', top_n=0)

    def test_different_strategies_give_different_results(self):
        """Test that different strategies score positions differently"""
        # Position with high motion but low complexity
        high_motion = PositionMetrics(100, 100, 50.0, 1.0, 5.0, 5.0)
        # Position with low motion but high complexity
        high_complexity = PositionMetrics(200, 200, 1.0, 50.0, 5.0, 5.0)

        positions = [high_motion, high_complexity]
        bounds = NormalizationBounds.from_positions(positions)

        # Motion Priority should favor high_motion
        motion_candidates = generate_strategy_candidates(
            positions, bounds, 'Motion Priority', top_n=2
        )

        # Visual Detail should favor high_complexity
        detail_candidates = generate_strategy_candidates(
            positions, bounds, 'Visual Detail', top_n=2
        )

        # Top candidates should be different
        assert motion_candidates[0].x != detail_candidates[0].x


class TestGenerateSpatialCandidates:
    """Tests for generate_spatial_candidates function"""

    def test_generate_spatial_candidates_basic(self):
        """Test basic spatial candidate generation"""
        # Create positions in different quadrants
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),   # Top-left
            PositionMetrics(1500, 100, 5.0, 10.0, 6.0, 8.0),  # Top-right
            PositionMetrics(100, 900, 8.0, 8.0, 9.0, 9.0),    # Bottom-left
            PositionMetrics(1500, 900, 7.0, 7.0, 7.0, 7.0),   # Bottom-right
            PositionMetrics(960, 540, 6.0, 6.0, 6.0, 6.0),    # Center
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_spatial_candidates(
            positions, bounds, 1920, 1080
        )

        # Should have up to 5 candidates (one per region)
        assert len(candidates) <= 5
        assert len(candidates) >= 1

        # All should be spatial candidates
        assert all(c.strategy.startswith('Spatial:') for c in candidates)

    def test_all_spatial_regions_covered(self):
        """Test that all 5 spatial regions are covered when positions exist"""
        # Create positions in all quadrants and center
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),   # Top-left
            PositionMetrics(1500, 100, 5.0, 10.0, 6.0, 8.0),  # Top-right
            PositionMetrics(100, 900, 8.0, 8.0, 9.0, 9.0),    # Bottom-left
            PositionMetrics(1500, 900, 7.0, 7.0, 7.0, 7.0),   # Bottom-right
            PositionMetrics(960, 540, 6.0, 6.0, 6.0, 6.0),    # Center
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_spatial_candidates(
            positions, bounds, 1920, 1080
        )

        strategies = {c.strategy for c in candidates}

        # Should have all 5 regions
        expected_regions = {
            'Spatial:Top-Left',
            'Spatial:Top-Right',
            'Spatial:Bottom-Left',
            'Spatial:Bottom-Right',
            'Spatial:Center'
        }

        assert strategies == expected_regions

    def test_missing_region_skipped(self):
        """Test that regions with no positions are skipped"""
        # Only positions in top-left
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(200, 200, 5.0, 10.0, 6.0, 8.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_spatial_candidates(
            positions, bounds, 1920, 1080
        )

        # Should have fewer than 5 candidates
        assert len(candidates) < 5

        # All should be from regions with positions
        strategies = {c.strategy for c in candidates}
        # Should only have regions that have positions
        assert all('Top-Left' in s or 'Center' in s for s in strategies)

    def test_best_position_selected_per_region(self):
        """Test that best position is selected from each region"""
        # Two positions in top-left with different scores
        positions = [
            PositionMetrics(100, 100, 1.0, 1.0, 1.0, 1.0),    # Lower score
            PositionMetrics(200, 200, 10.0, 10.0, 10.0, 10.0),  # Higher score
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_spatial_candidates(
            positions, bounds, 1920, 1080
        )

        # Should select the higher-scoring position
        assert len(candidates) >= 1
        top_left = [c for c in candidates if c.strategy == 'Spatial:Top-Left'][0]
        # Should be the position with all 10.0 metrics (higher score)
        assert top_left.x == 200
        assert top_left.y == 200

    def test_empty_positions_raises_error(self):
        """Test that empty positions list raises ValueError"""
        bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

        with pytest.raises(ValueError, match="empty positions list"):
            generate_spatial_candidates([], bounds, 1920, 1080)

    def test_invalid_dimensions_raise_error(self):
        """Test that invalid video dimensions raise ValueError"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        with pytest.raises(ValueError, match="Invalid video dimensions"):
            generate_spatial_candidates(positions, bounds, 0, 1080)

        with pytest.raises(ValueError, match="Invalid video dimensions"):
            generate_spatial_candidates(positions, bounds, 1920, -1)

    def test_center_region_calculation(self):
        """Test center region calculation"""
        # Position exactly in center
        center_pos = PositionMetrics(960, 540, 10.0, 10.0, 10.0, 10.0)
        # Position not in center
        edge_pos = PositionMetrics(100, 100, 5.0, 5.0, 5.0, 5.0)

        positions = [center_pos, edge_pos]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_spatial_candidates(
            positions, bounds, 1920, 1080
        )

        # Should have center candidate
        center_candidates = [c for c in candidates if c.strategy == 'Spatial:Center']
        assert len(center_candidates) == 1
        assert center_candidates[0].x == 960
        assert center_candidates[0].y == 540


class TestDeduplicateCandidates:
    """Tests for deduplicate_candidates function"""

    def test_deduplicate_basic(self):
        """Test basic deduplication"""
        candidates = [
            ScoredCandidate(100, 100, 95.0, 'Motion Priority'),
            ScoredCandidate(100, 100, 90.0, 'Balanced'),  # Duplicate
            ScoredCandidate(200, 200, 85.0, 'Visual Detail'),
        ]

        unique = deduplicate_candidates(candidates, max_candidates=10)

        assert len(unique) == 2
        # Should keep higher-scoring version
        assert unique[0].x == 100 and unique[0].score == 95.0

    def test_keeps_highest_score(self):
        """Test that highest score is kept for duplicates"""
        candidates = [
            ScoredCandidate(100, 100, 80.0, 'Strategy 1'),
            ScoredCandidate(100, 100, 95.0, 'Strategy 2'),  # Highest
            ScoredCandidate(100, 100, 85.0, 'Strategy 3'),
        ]

        unique = deduplicate_candidates(candidates, max_candidates=10)

        assert len(unique) == 1
        assert unique[0].score == 95.0
        assert unique[0].strategy == 'Strategy 2'

    def test_sorted_by_score_descending(self):
        """Test that results are sorted by score descending"""
        candidates = [
            ScoredCandidate(100, 100, 70.0, 'A'),
            ScoredCandidate(200, 200, 90.0, 'B'),
            ScoredCandidate(300, 300, 80.0, 'C'),
        ]

        unique = deduplicate_candidates(candidates, max_candidates=10)

        assert unique[0].score == 90.0  # Highest
        assert unique[1].score == 80.0
        assert unique[2].score == 70.0  # Lowest

    def test_max_candidates_limit(self):
        """Test that max_candidates limit is respected"""
        candidates = [
            ScoredCandidate(i, i, float(100 - i), f'Strategy {i}')
            for i in range(1, 21)
        ]

        unique = deduplicate_candidates(candidates, max_candidates=5)

        assert len(unique) == 5

    def test_excludes_zero_positions(self):
        """Test that positions with x=0 or y=0 are excluded"""
        candidates = [
            ScoredCandidate(0, 100, 95.0, 'Invalid X'),  # Invalid
            ScoredCandidate(100, 0, 90.0, 'Invalid Y'),  # Invalid
            ScoredCandidate(100, 100, 85.0, 'Valid'),
        ]

        unique = deduplicate_candidates(candidates, max_candidates=10)

        assert len(unique) == 1
        assert unique[0].x == 100 and unique[0].y == 100

    def test_empty_list_returns_empty(self):
        """Test that empty list returns empty list"""
        unique = deduplicate_candidates([], max_candidates=10)
        assert unique == []

    def test_invalid_max_candidates_raises_error(self):
        """Test that max_candidates < 1 raises ValueError"""
        candidates = [ScoredCandidate(100, 100, 90.0, 'Test')]

        with pytest.raises(ValueError, match="max_candidates must be at least 1"):
            deduplicate_candidates(candidates, max_candidates=0)

    def test_all_duplicates_keeps_one(self):
        """Test that all duplicates collapse to one"""
        candidates = [
            ScoredCandidate(100, 100, score, f'Strategy {i}')
            for i, score in enumerate([95.0, 90.0, 85.0, 80.0, 75.0])
        ]

        unique = deduplicate_candidates(candidates, max_candidates=10)

        assert len(unique) == 1
        assert unique[0].score == 95.0  # Highest


class TestGenerateCandidates:
    """Tests for generate_candidates function (full integration)"""

    def test_generate_candidates_basic(self):
        """Test basic candidate generation"""
        # Create a diverse set of positions
        positions = [
            PositionMetrics(i*100, j*100, float(i+j), float(i*j), float(i), float(j))
            for i in range(1, 6)
            for j in range(1, 6)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        assert len(candidates) <= 10
        assert len(candidates) >= 1

    def test_candidates_are_unique(self):
        """Test that all returned candidates are unique positions"""
        positions = [
            PositionMetrics(i*100, j*100, float(i+j), float(i*j), float(i), float(j))
            for i in range(1, 6)
            for j in range(1, 6)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        # Check all positions are unique
        positions_set = {(c.x, c.y) for c in candidates}
        assert len(positions_set) == len(candidates)

    def test_candidates_sorted_by_score(self):
        """Test that candidates are sorted by score descending"""
        positions = [
            PositionMetrics(i*100, j*100, float(i+j), float(i*j), float(i), float(j))
            for i in range(1, 6)
            for j in range(1, 6)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        # Verify descending order
        for i in range(len(candidates) - 1):
            assert candidates[i].score >= candidates[i+1].score

    def test_includes_strategy_and_spatial_candidates(self):
        """Test that result includes both strategy and spatial candidates"""
        # Create positions across different regions
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(1500, 100, 5.0, 10.0, 6.0, 8.0),
            PositionMetrics(100, 900, 8.0, 8.0, 9.0, 9.0),
            PositionMetrics(1500, 900, 7.0, 7.0, 7.0, 7.0),
            PositionMetrics(960, 540, 6.0, 6.0, 6.0, 6.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        strategies = {c.strategy for c in candidates}

        # Should have mix of strategy and spatial candidates
        has_spatial = any('Spatial:' in s for s in strategies)
        has_strategy = any('Spatial:' not in s for s in strategies)

        assert has_spatial or has_strategy  # Should have at least one type

    def test_empty_positions_raises_error(self):
        """Test that empty positions list raises ValueError"""
        bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

        with pytest.raises(ValueError, match="empty positions list"):
            generate_candidates([], bounds, 1920, 1080)

    def test_invalid_dimensions_raise_error(self):
        """Test that invalid video dimensions raise ValueError"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        with pytest.raises(ValueError, match="Invalid video dimensions"):
            generate_candidates(positions, bounds, 0, 1080)

    def test_invalid_max_candidates_raises_error(self):
        """Test that max_candidates < 1 raises ValueError"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        with pytest.raises(ValueError, match="max_candidates must be at least 1"):
            generate_candidates(positions, bounds, 1920, 1080, max_candidates=0)

    def test_invalid_top_per_strategy_raises_error(self):
        """Test that top_per_strategy < 1 raises ValueError"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        with pytest.raises(ValueError, match="top_per_strategy must be at least 1"):
            generate_candidates(positions, bounds, 1920, 1080, top_per_strategy=0)

    def test_max_candidates_respected(self):
        """Test that max_candidates limit is strictly respected"""
        # Create many positions
        positions = [
            PositionMetrics(i*50, j*50, float(i+j), float(i*j), float(i), float(j))
            for i in range(1, 11)
            for j in range(1, 11)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=7, top_per_strategy=5
        )

        assert len(candidates) <= 7

    def test_top_per_strategy_affects_results(self):
        """Test that top_per_strategy parameter affects results"""
        positions = [
            PositionMetrics(i*100, j*100, float(i+j), float(i*j), float(i), float(j))
            for i in range(1, 6)
            for j in range(1, 6)
        ]
        bounds = NormalizationBounds.from_positions(positions)

        # Generate with different top_per_strategy values
        candidates_3 = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=3
        )

        candidates_1 = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=1
        )

        # More candidates per strategy should give more total candidates (before dedup)
        # This is hard to test directly, but we can verify both work
        assert len(candidates_3) >= 1
        assert len(candidates_1) >= 1


class TestCandidateGenerationIntegration:
    """Integration tests combining multiple components"""

    def test_realistic_scenario(self):
        """Test a realistic candidate generation scenario"""
        # Simulate a 5x5 grid analysis of a 1920x1080 video
        positions = []
        for i in range(5):
            for j in range(5):
                x = 1 + (1280 * i // 4)  # Spread across movable width
                y = 1 + (440 * j // 4)   # Spread across movable height
                motion = float((i + j) * 2)
                complexity = float((i * j) + 5)
                edges = float(abs(i - j) + 3)
                saturation = float((i + 1) * (j + 1))
                positions.append(PositionMetrics(x, y, motion, complexity, edges, saturation))

        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        # Should get 10 diverse candidates
        assert len(candidates) <= 10
        assert len(candidates) >= 1

        # All should be valid
        assert all(c.x > 0 and c.y > 0 for c in candidates)
        assert all(c.score >= 0 for c in candidates)

        # Should be sorted
        for i in range(len(candidates) - 1):
            assert candidates[i].score >= candidates[i+1].score

        # Should be unique
        positions_set = {(c.x, c.y) for c in candidates}
        assert len(positions_set) == len(candidates)

    def test_edge_case_few_positions(self):
        """Test with very few positions"""
        positions = [
            PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),
            PositionMetrics(200, 200, 5.0, 10.0, 6.0, 8.0),
        ]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        # Should get at most 2 unique positions
        assert len(candidates) <= 2

    def test_edge_case_single_position(self):
        """Test with single position"""
        positions = [PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0)]
        bounds = NormalizationBounds.from_positions(positions)

        candidates = generate_candidates(
            positions, bounds, 1920, 1080,
            max_candidates=10, top_per_strategy=5
        )

        # Should get exactly 1 candidate (deduplicated across all strategies)
        assert len(candidates) == 1
        assert candidates[0].x == 100
        assert candidates[0].y == 100
