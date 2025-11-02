"""
Video analysis package.

Provides abstraction layer for video analysis operations, allowing multiple
implementations (FFmpeg, mocks, future GPU implementations, etc.).
"""
from smart_crop.analysis.analyzer import VideoAnalyzer

__all__ = ['VideoAnalyzer']
