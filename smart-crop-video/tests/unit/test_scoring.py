"""
Unit tests for smart_crop.core.scoring module.

Tests all pure functions for scoring strategies.
"""
import pytest
from smart_crop.core.scoring import (
    normalize,
    score_position,
    get_available_strategies,
    get_strategy_info,
    validate_strategy_weights,
    PositionMetrics,
    NormalizationBounds,
    STRATEGIES
)


class TestNormalize:
    """Tests for normalize function"""

    def test_normalize_middle_value(self):
        """Test normalizing middle value"""
        assert normalize(5, 0, 10) == 50.0

    def test_normalize_min_value(self):
        """Test normalizing minimum value"""
        assert normalize(0, 0, 10) == 0.0

    def test_normalize_max_value(self):
        """Test normalizing maximum value"""
        assert normalize(10, 0, 10) == 100.0

    def test_normalize_quarter_value(self):
        """Test normalizing quarter value"""
        assert normalize(2.5, 0, 10) == 25.0

    def test_normalize_three_quarter_value(self):
        """Test normalizing three-quarter value"""
        assert normalize(7.5, 0, 10) == 75.0

    def test_normalize_same_min_max(self):
        """Test normalizing when min equals max (no range)"""
        # Should return middle value (50.0)
        assert normalize(5, 5, 5) == 50.0
        assert normalize(0, 0, 0) == 50.0
        assert normalize(100, 100, 100) == 50.0

    def test_normalize_negative_range(self):
        """Test normalizing with negative values"""
        assert normalize(-5, -10, 0) == 50.0
        assert normalize(-10, -10, 0) == 0.0
        assert normalize(0, -10, 0) == 100.0

    def test_normalize_large_values(self):
        """Test normalizing large values"""
        assert normalize(500, 0, 1000) == 50.0
        assert normalize(1000, 0, 1000) == 100.0

    def test_normalize_fractional_values(self):
        """Test normalizing fractional values"""
        result = normalize(0.5, 0.0, 1.0)
        assert abs(result - 50.0) < 0.01


class TestPositionMetrics:
    """Tests for PositionMetrics dataclass"""

    def test_create_position_metrics(self):
        """Test creating PositionMetrics"""
        metrics = PositionMetrics(
            x=100,
            y=200,
            motion=1.5,
            complexity=2.5,
            edges=3.5,
            saturation=4.5
        )

        assert metrics.x == 100
        assert metrics.y == 200
        assert metrics.motion == 1.5
        assert metrics.complexity == 2.5
        assert metrics.edges == 3.5
        assert metrics.saturation == 4.5

    def test_position_metrics_equality(self):
        """Test PositionMetrics equality"""
        m1 = PositionMetrics(100, 200, 1.0, 2.0, 3.0, 4.0)
        m2 = PositionMetrics(100, 200, 1.0, 2.0, 3.0, 4.0)
        m3 = PositionMetrics(100, 200, 1.0, 2.0, 3.0, 5.0)

        assert m1 == m2
        assert m1 != m3


class TestNormalizationBounds:
    """Tests for NormalizationBounds dataclass and from_positions method"""

    def test_create_normalization_bounds(self):
        """Test creating NormalizationBounds"""
        bounds = NormalizationBounds(
            motion_min=0.0, motion_max=10.0,
            complexity_min=0.0, complexity_max=20.0,
            edges_min=0.0, edges_max=30.0,
            saturation_min=0.0, saturation_max=40.0
        )

        assert bounds.motion_min == 0.0
        assert bounds.motion_max == 10.0

    def test_from_positions_simple(self):
        """Test calculating bounds from simple position list"""
        positions = [
            PositionMetrics(0, 0, 1.0, 2.0, 3.0, 4.0),
            PositionMetrics(100, 100, 5.0, 6.0, 7.0, 8.0),
            PositionMetrics(200, 200, 3.0, 4.0, 5.0, 6.0),
        ]

        bounds = NormalizationBounds.from_positions(positions)

        assert bounds.motion_min == 1.0
        assert bounds.motion_max == 5.0
        assert bounds.complexity_min == 2.0
        assert bounds.complexity_max == 6.0
        assert bounds.edges_min == 3.0
        assert bounds.edges_max == 7.0
        assert bounds.saturation_min == 4.0
        assert bounds.saturation_max == 8.0

    def test_from_positions_single_position(self):
        """Test calculating bounds from single position"""
        positions = [PositionMetrics(0, 0, 5.0, 10.0, 15.0, 20.0)]

        bounds = NormalizationBounds.from_positions(positions)

        # Min and max should be the same
        assert bounds.motion_min == 5.0
        assert bounds.motion_max == 5.0
        assert bounds.complexity_min == 10.0
        assert bounds.complexity_max == 10.0

    def test_from_positions_empty_raises(self):
        """Test that empty positions list raises ValueError"""
        with pytest.raises(ValueError, match="Cannot calculate bounds from empty"):
            NormalizationBounds.from_positions([])

    def test_from_positions_with_zeros(self):
        """Test calculating bounds when some metrics are zero"""
        positions = [
            PositionMetrics(0, 0, 0.0, 0.0, 0.0, 0.0),
            PositionMetrics(100, 100, 10.0, 20.0, 30.0, 40.0),
        ]

        bounds = NormalizationBounds.from_positions(positions)

        assert bounds.motion_min == 0.0
        assert bounds.motion_max == 10.0


class TestScorePosition:
    """Tests for score_position function"""

    def test_score_balanced_all_middle_values(self):
        """Test balanced scoring with all metrics at middle"""
        metrics = PositionMetrics(100, 100, 5.0, 10.0, 15.0, 20.0)
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=20,
            edges_min=0, edges_max=30,
            saturation_min=0, saturation_max=40
        )

        score = score_position(metrics, bounds, 'Balanced')

        # All metrics are at 50% of their range
        # Balanced strategy: 0.25 * 50 + 0.25 * 50 + 0.25 * 50 + 0.25 * 50 = 50
        assert score == 50.0

    def test_score_balanced_all_max_values(self):
        """Test balanced scoring with all metrics at maximum"""
        metrics = PositionMetrics(100, 100, 10.0, 20.0, 30.0, 40.0)
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=20,
            edges_min=0, edges_max=30,
            saturation_min=0, saturation_max=40
        )

        score = score_position(metrics, bounds, 'Balanced')

        # All metrics at 100%, balanced: 0.25 * 100 + 0.25 * 100 + 0.25 * 100 + 0.25 * 100 = 100
        assert score == 100.0

    def test_score_balanced_all_min_values(self):
        """Test balanced scoring with all metrics at minimum"""
        metrics = PositionMetrics(100, 100, 0.0, 0.0, 0.0, 0.0)
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=20,
            edges_min=0, edges_max=30,
            saturation_min=0, saturation_max=40
        )

        score = score_position(metrics, bounds, 'Balanced')

        # All metrics at 0%
        assert score == 0.0

    def test_score_motion_priority_high_motion(self):
        """Test Motion Priority strategy with high motion"""
        metrics = PositionMetrics(
            x=100, y=100,
            motion=10.0,     # Max motion (100% normalized)
            complexity=0.0,  # Min complexity (0% normalized)
            edges=0.0,       # Min edges (0% normalized)
            saturation=0.0   # Min saturation (0% normalized)
        )
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=10,
            edges_min=0, edges_max=10,
            saturation_min=0, saturation_max=10
        )

        score = score_position(metrics, bounds, 'Motion Priority')

        # Motion Priority: 50% motion weight
        # Score = 100 * 0.5 + 0 * 0.15 + 0 * 0.25 + 0 * 0.10 = 50
        assert score == 50.0

    def test_score_visual_detail_high_complexity(self):
        """Test Visual Detail strategy with high complexity"""
        metrics = PositionMetrics(
            x=100, y=100,
            motion=0.0,       # Min
            complexity=20.0,  # Max (100% normalized)
            edges=0.0,        # Min
            saturation=0.0    # Min
        )
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=20,
            edges_min=0, edges_max=10,
            saturation_min=0, saturation_max=10
        )

        score = score_position(metrics, bounds, 'Visual Detail')

        # Visual Detail: 50% complexity weight
        # Score = 0 * 0.05 + 100 * 0.50 + 0 * 0.30 + 0 * 0.15 = 50
        assert score == 50.0

    def test_score_invalid_strategy_raises(self):
        """Test that invalid strategy raises ValueError"""
        metrics = PositionMetrics(100, 100, 1.0, 2.0, 3.0, 4.0)
        bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

        with pytest.raises(ValueError, match="Unknown strategy"):
            score_position(metrics, bounds, 'NonexistentStrategy')

    def test_score_returns_value_in_range(self):
        """Test that score is always in 0-100 range"""
        metrics = PositionMetrics(100, 100, 5.0, 7.0, 9.0, 11.0)
        bounds = NormalizationBounds(0, 10, 0, 10, 0, 10, 0, 10)

        for strategy in get_available_strategies():
            score = score_position(metrics, bounds, strategy)
            assert 0 <= score <= 100, f"Score out of range for {strategy}: {score}"

    def test_score_all_strategies(self):
        """Test scoring with all available strategies"""
        metrics = PositionMetrics(100, 100, 5.0, 10.0, 15.0, 20.0)
        bounds = NormalizationBounds(
            motion_min=0, motion_max=10,
            complexity_min=0, complexity_max=20,
            edges_min=0, edges_max=30,
            saturation_min=0, saturation_max=40
        )

        scores = {}
        for strategy in get_available_strategies():
            scores[strategy] = score_position(metrics, bounds, strategy)

        # All should be valid scores
        assert all(0 <= score <= 100 for score in scores.values())

        # Different strategies should potentially give different scores
        # (though not guaranteed with this specific data)
        assert len(scores) == len(get_available_strategies())


class TestGetAvailableStrategies:
    """Tests for get_available_strategies function"""

    def test_returns_list(self):
        """Test that function returns a list"""
        strategies = get_available_strategies()
        assert isinstance(strategies, list)

    def test_returns_expected_strategies(self):
        """Test that all expected strategies are present"""
        strategies = get_available_strategies()

        expected = ['Subject Detection', 'Motion Priority', 'Visual Detail', 'Balanced', 'Color Focus']
        for strategy in expected:
            assert strategy in strategies

    def test_returns_at_least_5_strategies(self):
        """Test that at least 5 strategies are available"""
        strategies = get_available_strategies()
        assert len(strategies) >= 5

    def test_no_empty_strategy_names(self):
        """Test that no strategy names are empty"""
        strategies = get_available_strategies()
        assert all(len(s) > 0 for s in strategies)


class TestGetStrategyInfo:
    """Tests for get_strategy_info function"""

    def test_get_balanced_strategy_info(self):
        """Test getting info for Balanced strategy"""
        info = get_strategy_info('Balanced')

        assert 'weights' in info
        assert 'description' in info
        assert 'use_case' in info

        # Balanced should have equal weights
        weights = info['weights']
        assert weights['motion'] == 0.25
        assert weights['complexity'] == 0.25
        assert weights['edges'] == 0.25
        assert weights['saturation'] == 0.25

    def test_get_motion_priority_info(self):
        """Test getting info for Motion Priority strategy"""
        info = get_strategy_info('Motion Priority')

        weights = info['weights']
        # Motion should have highest weight
        assert weights['motion'] == 0.50

    def test_get_info_invalid_strategy_raises(self):
        """Test that invalid strategy raises ValueError"""
        with pytest.raises(ValueError, match="Unknown strategy"):
            get_strategy_info('NonexistentStrategy')

    def test_get_info_all_strategies(self):
        """Test getting info for all strategies"""
        for strategy in get_available_strategies():
            info = get_strategy_info(strategy)

            assert 'weights' in info
            assert 'description' in info
            assert 'use_case' in info

            # Weights should sum to 1.0
            weights = info['weights']
            total = sum(weights.values())
            assert abs(total - 1.0) < 0.01

    def test_get_info_returns_copy(self):
        """Test that get_strategy_info returns a copy, not reference"""
        info1 = get_strategy_info('Balanced')
        info2 = get_strategy_info('Balanced')

        # Modifying one shouldn't affect the other
        info1['weights']['motion'] = 0.99

        assert info2['weights']['motion'] == 0.25  # Should still be original value


class TestValidateStrategyWeights:
    """Tests for validate_strategy_weights function"""

    def test_validate_balanced_weights(self):
        """Test validating balanced weights"""
        weights = {
            'motion': 0.25,
            'complexity': 0.25,
            'edges': 0.25,
            'saturation': 0.25
        }

        assert validate_strategy_weights(weights) is True

    def test_validate_unbalanced_weights(self):
        """Test validating unbalanced weights"""
        weights = {
            'motion': 0.5,
            'complexity': 0.3,
            'edges': 0.15,
            'saturation': 0.05
        }

        assert validate_strategy_weights(weights) is True

    def test_validate_missing_key_raises(self):
        """Test that missing key raises ValueError"""
        weights = {
            'motion': 0.5,
            'complexity': 0.5,
            # Missing 'edges' and 'saturation'
        }

        with pytest.raises(ValueError, match="must have exactly these keys"):
            validate_strategy_weights(weights)

    def test_validate_extra_key_raises(self):
        """Test that extra key raises ValueError"""
        weights = {
            'motion': 0.25,
            'complexity': 0.25,
            'edges': 0.25,
            'saturation': 0.25,
            'extra': 0.0  # Extra key
        }

        with pytest.raises(ValueError, match="must have exactly these keys"):
            validate_strategy_weights(weights)

    def test_validate_negative_weight_raises(self):
        """Test that negative weight raises ValueError"""
        weights = {
            'motion': -0.25,
            'complexity': 0.5,
            'edges': 0.5,
            'saturation': 0.25
        }

        with pytest.raises(ValueError, match="must be non-negative"):
            validate_strategy_weights(weights)

    def test_validate_weights_not_summing_to_one_raises(self):
        """Test that weights not summing to 1.0 raises ValueError"""
        weights = {
            'motion': 0.25,
            'complexity': 0.25,
            'edges': 0.25,
            'saturation': 0.10  # Sum = 0.85, not 1.0
        }

        with pytest.raises(ValueError, match="must sum to 1.0"):
            validate_strategy_weights(weights)

    def test_validate_weights_summing_slightly_off_accepted(self):
        """Test that weights slightly off 1.0 (due to floating point) are accepted"""
        # This might happen due to floating point precision
        weights = {
            'motion': 0.333333,
            'complexity': 0.333333,
            'edges': 0.333333,
            'saturation': 0.000001  # Sum ~= 1.0
        }

        assert validate_strategy_weights(weights) is True

    def test_validate_all_built_in_strategies(self):
        """Test that all built-in strategies have valid weights"""
        for strategy_name, strategy_config in STRATEGIES.items():
            # Should not raise
            assert validate_strategy_weights(strategy_config['weights']) is True


class TestStrategyConsistency:
    """Tests for consistency of strategy definitions"""

    def test_all_strategies_have_required_fields(self):
        """Test that all strategies have required fields"""
        for strategy_name, strategy_config in STRATEGIES.items():
            assert 'weights' in strategy_config, f"{strategy_name} missing 'weights'"
            assert 'description' in strategy_config, f"{strategy_name} missing 'description'"
            assert 'use_case' in strategy_config, f"{strategy_name} missing 'use_case'"

    def test_all_strategies_have_four_weights(self):
        """Test that all strategies have exactly four weights"""
        for strategy_name, strategy_config in STRATEGIES.items():
            weights = strategy_config['weights']
            assert len(weights) == 4, f"{strategy_name} has {len(weights)} weights, expected 4"

    def test_all_strategies_weights_sum_to_one(self):
        """Test that all strategy weights sum to 1.0"""
        for strategy_name, strategy_config in STRATEGIES.items():
            weights = strategy_config['weights']
            total = sum(weights.values())
            assert abs(total - 1.0) < 0.01, f"{strategy_name} weights sum to {total}, not 1.0"

    def test_balanced_strategy_has_equal_weights(self):
        """Test that Balanced strategy has equal weights"""
        weights = STRATEGIES['Balanced']['weights']
        assert all(w == 0.25 for w in weights.values())

    def test_motion_priority_emphasizes_motion(self):
        """Test that Motion Priority has highest weight on motion"""
        weights = STRATEGIES['Motion Priority']['weights']
        assert weights['motion'] == max(weights.values())
        assert weights['motion'] >= 0.5  # At least 50%

    def test_visual_detail_emphasizes_complexity(self):
        """Test that Visual Detail has highest weight on complexity"""
        weights = STRATEGIES['Visual Detail']['weights']
        assert weights['complexity'] == max(weights.values())
        assert weights['complexity'] >= 0.5  # At least 50%

    def test_color_focus_emphasizes_saturation(self):
        """Test that Color Focus has highest weight on saturation"""
        weights = STRATEGIES['Color Focus']['weights']
        assert weights['saturation'] == max(weights.values())
        assert weights['saturation'] >= 0.45  # At least 45%
