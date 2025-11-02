# Phase 4 Complete: Extract Candidate Selection Logic

**Date**: 2025-11-01
**Status**: ✅ COMPLETE
**Time Spent**: ~1 hour

---

## Summary

Successfully extracted candidate selection logic from the monolithic main() function into a dedicated, testable module. This enables testing the candidate generation algorithm without running FFmpeg or generating actual video frames.

## Module Created

### `smart_crop/core/candidates.py` (350 lines)
**Purpose**: Generate diverse candidate crop positions using multiple strategies

**Key Features**:
- Strategy-based candidate selection (top N from each scoring strategy)
- Spatial diversity (best candidate from each quadrant + center)
- Intelligent deduplication (keeps highest-scoring duplicate)
- Configurable candidate count and strategies

**Core Functions**:

```python
@dataclass
class ScoredCandidate:
    """Candidate with position, score, and strategy that selected it"""
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
    """Generate top N candidates using specific scoring strategy"""

def generate_spatial_candidates(
    positions: List[PositionMetrics],
    bounds: NormalizationBounds,
    video_width: int,
    video_height: int
) -> List[ScoredCandidate]:
    """Generate spatially diverse candidates from 5 regions"""

def deduplicate_candidates(
    candidates: List[ScoredCandidate],
    max_candidates: int = 10
) -> List[ScoredCandidate]:
    """Deduplicate and return top N unique candidates"""

def generate_candidates(
    positions: List[PositionMetrics],
    bounds: NormalizationBounds,
    video_width: int,
    video_height: int,
    max_candidates: int = 10,
    top_per_strategy: int = 5
) -> List[ScoredCandidate]:
    """Main entry point: Generate diverse candidate set"""
```

---

## Algorithm Details

### Candidate Generation Process

**Step 1: Strategy-Based Selection** (25 candidates)
```python
# For each of 5 scoring strategies
strategies = [
    'Subject Detection',  # 40% edges, 30% saturation
    'Motion Priority',    # 50% motion, 25% edges
    'Visual Detail',      # 50% complexity, 30% edges
    'Balanced',          # 25% each metric
    'Color Focus'        # 45% saturation, 30% edges
]

# Take top 5 positions from each strategy
for strategy in strategies:
    top_5 = score_all_positions_and_take_top_5(strategy)
    candidates.extend(top_5)
# Result: 25 candidates (5 strategies × 5 candidates)
```

**Step 2: Spatial Diversity** (up to 5 candidates)
```python
# Divide video into 5 regions
regions = [
    'Top-Left',      # x < center_x AND y < center_y
    'Top-Right',     # x >= center_x AND y < center_y
    'Bottom-Left',   # x < center_x AND y >= center_y
    'Bottom-Right',  # x >= center_x AND y >= center_y
    'Center'         # within 1/4 width/height of center
]

# Find best position in each region using Balanced strategy
for region in regions:
    best_in_region = find_best_position(region, 'Balanced')
    candidates.append(best_in_region)
# Result: Up to 5 additional candidates (one per region)
```

**Step 3: Deduplication** (→ 10 candidates)
```python
# Sort all candidates by score (descending)
# Remove duplicates (same x, y), keeping highest score
# Remove invalid positions (x=0 or y=0)
# Take top 10 unique candidates

unique_candidates = deduplicate_and_select_top_10(all_candidates)
# Result: 10 unique, high-scoring, spatially diverse candidates
```

### Why This Approach?

**Diversity**: 5 different scoring strategies ensure we don't miss good candidates due to a single scoring bias

**Spatial Coverage**: Quadrant-based selection ensures we sample different regions of the video, not just one area

**Quality**: Taking top 5 from each strategy focuses on high-scoring positions

**Deduplication**: Multiple strategies often select the same position; we keep the highest-scoring version

---

## Test Statistics

### New Unit Tests
- **Candidate Tests**: 38 tests
  - ScoredCandidate dataclass: 3 tests
  - Strategy candidate generation: 7 tests
  - Spatial candidate generation: 7 tests
  - Deduplication: 8 tests
  - Full generation pipeline: 10 tests
  - Integration tests: 3 tests

### Total Test Count
- **Phase 3**: 155 tests
- **Phase 4**: +38 tests
- **Total**: 193 unit tests

### Test Execution Time
- **All unit tests**: 0.09 seconds (193 tests)
- **Candidates alone**: 0.04 seconds (38 tests)
- **Previous phases**: 0.05 seconds (155 tests)

### Coverage Improvement
- **Phase 3 Coverage**: ~57%
- **Phase 4 Coverage**: ~62% (+5%)
- **Lines Made Testable**: ~200 lines (candidate generation)

---

## Key Achievements

### 1. Candidate Generation Now Testable

**Before (Untestable)**:
```python
# Inside massive 628-line main() function
all_candidates = []
strategies = ['Subject Detection', 'Motion Priority', ...]

for strategy in strategies:
    scored = [(p, score_with_strategy(p, mins, maxs, strategy))
              for p in positions]
    scored.sort(key=lambda x: x[1], reverse=True)

    for pos, score in scored[:5]:
        candidate = ScoredCandidate(pos.x, pos.y, score, strategy)
        all_candidates.append(candidate)

# ... 30 more lines of quadrant logic ...
# ... 10 more lines of deduplication ...
```

**After (Fully Testable)**:
```python
# Clean, testable function
candidates = generate_candidates(
    positions=analyzed_positions,
    bounds=normalization_bounds,
    video_width=1920,
    video_height=1080,
    max_candidates=10,
    top_per_strategy=5
)

# Test with mock data
def test_candidate_generation():
    positions = [PositionMetrics(...), ...]
    bounds = NormalizationBounds.from_positions(positions)
    candidates = generate_candidates(positions, bounds, 1920, 1080)

    assert len(candidates) <= 10
    assert all_unique_positions(candidates)
    assert sorted_by_score(candidates)
```

### 2. Fast Unit Tests

**Speed Comparison**:
| Test Type | With FFmpeg | With Mock | Speedup |
|-----------|-------------|-----------|---------|
| Generate candidates (25 positions) | ~75s analysis + candidate selection | <0.001s | **Instant** |
| Test deduplication | N/A (untestable) | <0.001s | **New capability** |
| Test spatial diversity | N/A (untestable) | <0.001s | **New capability** |

**Real Example from Tests**:
```python
def test_realistic_scenario(self):
    """Test a realistic candidate generation scenario"""
    # Simulate 5x5 grid analysis
    positions = [...]  # 25 positions
    bounds = NormalizationBounds.from_positions(positions)

    candidates = generate_candidates(
        positions, bounds, 1920, 1080,
        max_candidates=10, top_per_strategy=5
    )

    # All assertions complete in <0.001s
    assert len(candidates) <= 10
    assert all_unique_positions(candidates)
    # Real execution would take 75+ seconds with FFmpeg
```

### 3. Algorithm Verification

Tests verify the candidate generation algorithm:

**Uniqueness**:
```python
def test_candidates_are_unique(self):
    """All returned candidates must be unique positions"""
    candidates = generate_candidates(...)
    positions_set = {(c.x, c.y) for c in candidates}
    assert len(positions_set) == len(candidates)
```

**Sorting**:
```python
def test_candidates_sorted_by_score(self):
    """Candidates must be sorted by score descending"""
    candidates = generate_candidates(...)
    for i in range(len(candidates) - 1):
        assert candidates[i].score >= candidates[i+1].score
```

**Deduplication**:
```python
def test_keeps_highest_score(self):
    """When duplicates exist, keep highest-scoring version"""
    candidates = [
        ScoredCandidate(100, 100, 80.0, 'Strategy 1'),
        ScoredCandidate(100, 100, 95.0, 'Strategy 2'),  # Highest
        ScoredCandidate(100, 100, 85.0, 'Strategy 3'),
    ]
    unique = deduplicate_candidates(candidates)
    assert len(unique) == 1
    assert unique[0].score == 95.0  # Kept highest
```

**Spatial Coverage**:
```python
def test_all_spatial_regions_covered(self):
    """All 5 spatial regions should be covered"""
    positions = [
        PositionMetrics(100, 100, ...),    # Top-left
        PositionMetrics(1500, 100, ...),   # Top-right
        PositionMetrics(100, 900, ...),    # Bottom-left
        PositionMetrics(1500, 900, ...),   # Bottom-right
        PositionMetrics(960, 540, ...),    # Center
    ]
    candidates = generate_spatial_candidates(...)

    strategies = {c.strategy for c in candidates}
    expected = {'Spatial:Top-Left', 'Spatial:Top-Right',
                'Spatial:Bottom-Left', 'Spatial:Bottom-Right',
                'Spatial:Center'}
    assert strategies == expected
```

### 4. Edge Case Handling

The module handles all edge cases:

**Empty positions**:
```python
with pytest.raises(ValueError, match="empty positions list"):
    generate_candidates([], bounds, 1920, 1080)
```

**Invalid dimensions**:
```python
with pytest.raises(ValueError, match="Invalid video dimensions"):
    generate_candidates(positions, bounds, 0, 1080)
```

**Single position** (deduplicates to 1):
```python
positions = [PositionMetrics(100, 100, ...)]
candidates = generate_candidates(positions, bounds, 1920, 1080)
assert len(candidates) == 1  # All strategies select same position
```

**Few positions** (returns available):
```python
positions = [PositionMetrics(100, 100, ...),
             PositionMetrics(200, 200, ...)]
candidates = generate_candidates(positions, bounds, 1920, 1080)
assert len(candidates) <= 2  # Can't return more than exist
```

**Invalid positions** (x=0 or y=0 excluded):
```python
candidates = [
    ScoredCandidate(0, 100, 95.0, 'Invalid'),  # x=0
    ScoredCandidate(100, 0, 90.0, 'Invalid'),  # y=0
    ScoredCandidate(100, 100, 85.0, 'Valid')
]
unique = deduplicate_candidates(candidates)
assert len(unique) == 1  # Only valid position
```

---

## Code Examples

### Example 1: Generate Candidates for Analysis Results

```python
from smart_crop.core.candidates import generate_candidates
from smart_crop.core.scoring import PositionMetrics, NormalizationBounds

# After analyzing all positions with FFmpeg/MockAnalyzer
analyzed_positions = [
    PositionMetrics(100, 100, motion=10.0, complexity=8.0, edges=9.0, saturation=7.0),
    PositionMetrics(200, 200, motion=5.0, complexity=12.0, edges=6.0, saturation=8.0),
    # ... 23 more positions from 5x5 grid
]

# Calculate normalization bounds
bounds = NormalizationBounds.from_positions(analyzed_positions)

# Generate 10 diverse candidates
candidates = generate_candidates(
    positions=analyzed_positions,
    bounds=bounds,
    video_width=1920,
    video_height=1080,
    max_candidates=10,
    top_per_strategy=5
)

# candidates now contains up to 10 ScoredCandidate objects
for i, candidate in enumerate(candidates, 1):
    print(f"{i}. Position ({candidate.x}, {candidate.y})")
    print(f"   Score: {candidate.score:.1f}")
    print(f"   Strategy: {candidate.strategy}")
```

### Example 2: Testing Candidate Selection Strategy

```python
def test_motion_priority_favors_high_motion():
    """Verify Motion Priority strategy selects high-motion positions"""
    high_motion = PositionMetrics(100, 100, motion=50.0, complexity=1.0,
                                   edges=5.0, saturation=5.0)
    low_motion = PositionMetrics(200, 200, motion=1.0, complexity=50.0,
                                  edges=8.0, saturation=8.0)

    positions = [high_motion, low_motion]
    bounds = NormalizationBounds.from_positions(positions)

    # Generate candidates using Motion Priority
    candidates = generate_strategy_candidates(
        positions, bounds, 'Motion Priority', top_n=1
    )

    # Should select high-motion position
    assert candidates[0].x == 100  # high_motion position
    assert candidates[0].y == 100
```

### Example 3: Testing Spatial Diversity

```python
def test_spatial_diversity_covers_all_quadrants():
    """Verify spatial diversity selects from all regions"""
    # Create positions in each quadrant
    positions = [
        PositionMetrics(100, 100, 10.0, 5.0, 8.0, 7.0),    # Top-left
        PositionMetrics(1500, 100, 8.0, 8.0, 9.0, 9.0),    # Top-right
        PositionMetrics(100, 900, 7.0, 7.0, 7.0, 7.0),     # Bottom-left
        PositionMetrics(1500, 900, 6.0, 6.0, 6.0, 6.0),    # Bottom-right
        PositionMetrics(960, 540, 9.0, 9.0, 8.0, 8.0),     # Center
    ]
    bounds = NormalizationBounds.from_positions(positions)

    candidates = generate_spatial_candidates(
        positions, bounds, 1920, 1080
    )

    # Should have one candidate from each region
    assert len(candidates) == 5

    strategies = {c.strategy for c in candidates}
    assert 'Spatial:Top-Left' in strategies
    assert 'Spatial:Top-Right' in strategies
    assert 'Spatial:Bottom-Left' in strategies
    assert 'Spatial:Bottom-Right' in strategies
    assert 'Spatial:Center' in strategies
```

---

## Architecture Benefits

### Before Phase 4
```
Main Script (1783 lines)
├── main() function (628 lines)
│   ├── ... earlier logic ...
│   ├── Candidate generation (50 lines) ❌ Untestable
│   │   ├── Strategy scoring loops
│   │   ├── Spatial diversity logic
│   │   └── Deduplication
│   └── ... later logic ...
```

### After Phase 4
```
smart_crop/core/candidates.py
├── generate_strategy_candidates() ✅ Testable
├── generate_spatial_candidates() ✅ Testable
├── deduplicate_candidates() ✅ Testable
└── generate_candidates() ✅ Testable (orchestrates all)

Main Script
└── main() function
    └── candidates = generate_candidates(...) ✅ Simple call
```

---

## Files Created

### Source Code
```
smart-crop-video/
├── smart_crop/
│   ├── core/
│   │   ├── candidates.py         (350 lines) ✅ NEW
│   │   ├── dimensions.py         (167 lines) ✅ Phase 1
│   │   ├── grid.py              (141 lines) ✅ Phase 1
│   │   └── scoring.py           (312 lines) ✅ Phase 1
```

### Tests
```
tests/
├── unit/
│   ├── test_candidates.py        (500 lines, 38 tests) ✅ NEW
│   ├── test_dimensions.py        (27 tests) ✅ Phase 1
│   ├── test_grid.py             (28 tests) ✅ Phase 1
│   ├── test_scoring.py          (48 tests) ✅ Phase 1
│   ├── test_mock_analyzer.py    (19 tests) ✅ Phase 2
│   └── test_parallel.py         (33 tests) ✅ Phase 3
```

---

## Test Categories

### ScoredCandidate Tests (3 tests)
- Creation
- Equality
- Different positions

### Strategy Candidate Generation (7 tests)
- Basic generation
- Sorting by score
- top_n limits results
- top_n > available positions
- Empty positions error
- Invalid top_n error
- Different strategies give different results

### Spatial Candidate Generation (7 tests)
- Basic generation
- All regions covered
- Missing regions skipped
- Best position selected per region
- Empty positions error
- Invalid dimensions error
- Center region calculation

### Deduplication (8 tests)
- Basic deduplication
- Keeps highest score
- Sorted by score
- max_candidates limit
- Excludes zero positions
- Empty list returns empty
- Invalid max_candidates error
- All duplicates collapse to one

### Full Generation Pipeline (10 tests)
- Basic generation
- Candidates are unique
- Sorted by score
- Includes strategy and spatial candidates
- Empty positions error
- Invalid dimensions error
- Invalid max_candidates error
- Invalid top_per_strategy error
- max_candidates respected
- top_per_strategy affects results

### Integration Tests (3 tests)
- Realistic scenario (5x5 grid)
- Edge case: few positions
- Edge case: single position

---

## Success Metrics

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| New modules created | 1 | 1 | ✅ Met |
| Unit tests added | 30+ | 38 | ✅ Exceeded |
| Test coverage gain | +4% | +5% | ✅ Exceeded |
| Test execution time | < 0.1s | 0.04s | ✅ Met |
| Backward compatibility | 100% | 100% | ✅ Met |
| All tests passing | Yes | 193/193 | ✅ Met |

---

## Benefits Unlocked

### Immediate Benefits
1. ✅ Candidate generation now testable without FFmpeg
2. ✅ Algorithm correctness verified with tests
3. ✅ Edge cases handled and tested
4. ✅ Deduplication logic isolated and verified
5. ✅ Spatial diversity logic testable

### Future Benefits
1. 🔮 Easy to add new scoring strategies
2. 🔮 Can experiment with different candidate counts
3. 🔮 Can test strategy effectiveness independently
4. 🔮 Can optimize deduplication algorithm
5. 🔮 Can add ML-based candidate ranking

---

## Next Steps (Phase 5)

Phase 5 will extract scene detection logic:

1. **Create `smart_crop/analysis/scenes.py`**
   - Scene boundary detection using FFmpeg
   - Acceleration-friendly region identification
   - Scene timestamp extraction

2. **Benefits**:
   - Testable scene detection without video files
   - Separate concerns: scene detection vs analysis
   - Enable unit testing of scene logic

3. **Estimated Coverage Gain**: +4% (~150 lines)

---

## Lessons Learned

### What Worked Well
1. **Clear algorithm decomposition** - Breaking candidate generation into 3 steps (strategy, spatial, dedup) made testing easy
2. **ScoredCandidate dataclass** - Simple, clear representation of candidates
3. **Comprehensive edge case testing** - Empty lists, single positions, invalid dimensions all covered
4. **Integration tests** - Realistic scenarios verify the full pipeline

### Challenges Overcome
1. **Spatial region calculation** - Needed careful testing to verify quadrant boundaries
2. **Deduplication strategy** - Decided to keep highest score, not first seen
3. **Invalid position handling** - x=0, y=0 are invalid crop positions, needed to filter

---

## Code Quality

### Type Safety
- ✅ All functions have type hints
- ✅ Return types documented
- ✅ Clear dataclass definitions

### Documentation
- ✅ Comprehensive docstrings with examples
- ✅ Algorithm explanation in module docstring
- ✅ Test names describe behavior

### Testability
- ✅ 100% of public functions tested
- ✅ Edge cases covered
- ✅ Integration examples provided
- ✅ Realistic scenarios tested

---

**Phase 4: COMPLETE ✅**

We now have a fully testable candidate generation system with:
- **193 unit tests** (0.09s execution)
- **62% code coverage** (+5% from Phase 3)
- **Instant testing** without FFmpeg
- **Full backward compatibility**

The core business logic is now extracted and testable. Only scene detection and the main orchestration logic remain in the monolithic script.

Ready to proceed with Phase 5: Extract Scene Detection Logic!
