# Containerized Testing for smart-crop-video

Run the entire test suite **without installing Python, Playwright, or any dependencies locally**. Only Docker and Docker Compose required!

## Quick Start

### Prerequisites

- **Docker** (https://docs.docker.com/get-docker/)
- **Docker Compose** (https://docs.docker.com/compose/install/)

That's it! No Python, no Playwright, no pip packages needed on your machine.

### Run Tests

**Option 1: Using the shell script** (recommended)
```bash
./run-tests.sh container    # Fast validation (15 tests, ~40s)
./run-tests.sh quick        # Container + diagnostic tests
./run-tests.sh all          # Full test suite
```

**Option 2: Using Make**
```bash
make test-container    # Fast validation
make test-quick        # Quick smoke test
make test              # Full test suite
```

**Option 3: Using Docker Compose directly**
```bash
docker-compose -f docker-compose.test.yml run --rm tests
```

## Available Commands

### Shell Script (`./run-tests.sh`)

| Command | Description | Duration |
|---------|-------------|----------|
| `container` | Container integration tests | ~40s |
| `quick` | Fast smoke test (container + diagnostic) | ~50s |
| `api` | API endpoint tests | ~3-5 min |
| `ui` | Web UI tests | ~3-5 min |
| `focused` | Focused web UI tests | ~3-5 min |
| `all` | Complete test suite | ~5-10 min |
| `shell` | Open bash shell in test container | Interactive |
| `build` | Build test container image | ~2-5 min |
| `clean` | Remove test artifacts | ~5s |

### Examples

```bash
# Quick validation before committing
./run-tests.sh container

# Full validation before release
./run-tests.sh all

# Debug a failing test
./run-tests.sh shell
# Then inside container:
pytest tests/test_container.py::test_docker_image_builds -v -s

# Clean up everything
./run-tests.sh clean
```

## How It Works

### Test Container Architecture

```
┌─────────────────────────────────────┐
│   Your Machine                      │
│   (Only needs Docker)               │
│                                     │
│  ┌───────────────────────────────┐ │
│  │  Test Container                │ │
│  │  - Python 3.11                 │ │
│  │  - pytest                      │ │
│  │  - Playwright + Chromium       │ │
│  │  - Docker client               │ │
│  │                                │ │
│  │  Creates & Tests ──────────┐   │ │
│  └────────────────────────────│───┘ │
│                               │     │
│  ┌────────────────────────────▼───┐ │
│  │  smart-crop-video Container  │ │
│  │  - Flask app on port 8765    │ │
│  │  - FFmpeg video processing   │ │
│  └──────────────────────────────┘ │
└─────────────────────────────────────┘
```

The test container:
1. Mounts your workspace (`/workspace`)
2. Accesses Docker socket (`/var/run/docker.sock`)
3. Builds and runs smart-crop-video containers
4. Runs Playwright browser tests
5. Reports results

### Volume Mounts

- **Workspace**: `.:/workspace` - Your code is live-mounted
- **Docker Socket**: `/var/run/docker.sock` - Allows creating containers
- **Pytest Cache**: `pytest-cache:/workspace/.pytest_cache` - Speeds up reruns

### First Run

The first time you run tests:
1. Docker pulls Python 3.11 base image (~200 MB)
2. Installs apt packages (Docker client, etc.)
3. Installs Python dependencies (pytest, playwright, etc.)
4. Downloads Chromium browser (~150 MB)
5. **Total**: ~5 minutes, ~400 MB

Subsequent runs are **much faster** (just test execution time).

## Troubleshooting

### "Cannot connect to Docker daemon"

**Problem**: Test container can't access Docker
**Solution**: Ensure Docker daemon is running
```bash
docker ps  # Should work before running tests
```

### "Permission denied: /var/run/docker.sock"

**Problem**: Test container needs permission to access Docker socket
**Solution**: This is handled automatically (container runs as root)

On Linux, if still having issues:
```bash
sudo chmod 666 /var/run/docker.sock  # Temporary fix
# OR add your user to docker group (permanent)
sudo usermod -aG docker $USER
```

### "Image build failed"

**Problem**: Network issues or disk space
**Solution**:
```bash
# Check disk space
df -h

# Rebuild from scratch
docker-compose -f docker-compose.test.yml build --no-cache tests

# Check network
docker pull python:3.11-slim
```

### Tests are slow

**First run**: Building test image takes 3-5 minutes (one-time cost)
**Subsequent runs**: Only test execution time

To speed up tests:
```bash
# Run only fast tests
./run-tests.sh container    # 15 tests, ~40s

# Or use pytest markers
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/ -m "not slow"
```

### "Port 8765 already in use"

**Problem**: Another process using port 8765
**Solution**:
```bash
# Find what's using the port
lsof -i :8765   # macOS/Linux
netstat -ano | findstr :8765   # Windows

# Kill the process or change the port in tests
```

## Advanced Usage

### Run Specific Tests

```bash
# Single test file
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/test_container.py -v

# Single test function
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/test_container.py::test_docker_image_builds -v

# Tests matching pattern
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/ -k "container" -v
```

### Interactive Debugging

```bash
# Open shell in test container
./run-tests.sh shell

# Inside container, you can:
pytest tests/test_container.py -v -s  # Run with output
pytest --pdb  # Drop into debugger on failure
python  # Interactive Python with all deps available
```

### Custom pytest Arguments

```bash
# Pass any pytest args
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/ -v --tb=short --maxfail=1

# Show slowest tests
docker-compose -f docker-compose.test.yml run --rm tests \
    pytest tests/ --durations=10
```

### Update Test Dependencies

Edit `tests/requirements.txt`, then:
```bash
# Rebuild test image
./run-tests.sh build

# Or with docker-compose
docker-compose -f docker-compose.test.yml build --no-cache tests
```

## CI/CD Integration

### GitHub Actions

The test container works great in CI:

```yaml
- name: Run containerized tests
  run: |
    docker-compose -f docker-compose.test.yml build tests
    docker-compose -f docker-compose.test.yml run --rm tests
```

### GitLab CI

```yaml
test:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker-compose -f docker-compose.test.yml build tests
    - docker-compose -f docker-compose.test.yml run --rm tests
```

## Benefits

✅ **No local dependencies** - Only Docker required
✅ **Consistent environment** - Same Python/Playwright version everywhere
✅ **Portable** - Works on macOS, Linux, Windows
✅ **CI-ready** - Same container in dev and CI/CD
✅ **Isolated** - Tests don't pollute your system
✅ **Reproducible** - Exact same setup every time

## Performance

| Test Suite | First Run | Subsequent |
|------------|-----------|-----------|
| Image build | 3-5 min | Cached |
| Container tests | ~40s | ~40s |
| API tests | 3-5 min | 3-5 min |
| Web UI tests | 3-5 min | 3-5 min |
| Full suite | 5-10 min | 5-10 min |

## Files

- `tests/Dockerfile` - Test container image definition
- `docker-compose.test.yml` - Orchestration config
- `run-tests.sh` - Convenience script
- `Makefile` - Make targets for testing
- `tests/requirements.txt` - Python dependencies
- `.gitignore` - Excludes test artifacts

## Next Steps

1. **Quick validation**: `./run-tests.sh container`
2. **Check current status**: `./run-tests.sh quick`
3. **Full test suite**: `./run-tests.sh all`
4. **Debug failures**: `./run-tests.sh shell`

Need help? Check `./run-tests.sh help` or `make help`
