# Test execution environment for smart-crop-video
# Includes Python, pytest, Playwright, and Docker client
# Can be run without installing anything locally except Docker

FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    # FFmpeg for video processing and comprehensive tests
    ffmpeg \
    # Docker client (for running containers from tests)
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
      $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    # Cleanup
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace

# Copy test requirements
COPY tests/requirements.txt /workspace/tests/requirements.txt

# Install Python test dependencies
RUN pip install --no-cache-dir -r /workspace/tests/requirements.txt

# Install Playwright browsers
RUN playwright install chromium && \
    playwright install-deps chromium

# Copy test files (done at runtime via volume mount for development)
# COPY . /workspace/

# Default command: run all tests
CMD ["pytest", "tests/", "-v"]
