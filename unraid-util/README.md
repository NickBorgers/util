# unraid-util

I have started using unraid as a converged data storage and workload environment. 
The simplicity it provides comes at a cost, which is that I feel the command line interface on the base OS is not well suited to diagnosing problems.

An example is that `tcpdump` is not available, which makes it very difficult for me to diagnose problems.

I got around this by creating a container image which slept forever, giving me something to `exec` into and take actions including running `tcpdump`.

This is a published version of that image so others can use it and I can publish it in the unraid app store.

## How to use
First start the container. There are no configuration options. I suggest naming the container "util".

Next, go to your unraid terminal and run:
```
docker exec -it util bash
```

This will drop you into a bash shell in the util container. You can now run commands such as tcpdump like:
```
tcpdump -i any
```

If you want to install additional utilities or packages you can do so like:
```
apt install dnsutils
```

It's an Ubuntu instance, do what you will.

## Intel NPU Monitoring

This container includes [nputop](https://github.com/ZoLArk173/nputop), a terminal-based monitoring tool for Intel Neural Processing Units (NPUs):

```bash
nputop
```

This displays real-time NPU usage similar to how `nvtop` works for NVIDIA GPUs. Useful for monitoring AI/ML workloads on Intel systems with NPU hardware.

**Note**: Requires Intel NPU hardware and appropriate host device passthrough to the container.

## Disk Management Scripts

This container includes the [unraid-diskmv](https://github.com/trinapicot/unraid-diskmv) scripts for moving files between disks:

### diskmv
Move files or directories between disks within a user share:
```bash
# Test mode (default - no files moved)
diskmv /path/to/share disk1 disk2

# Force mode (actually moves files)
diskmv -f /path/to/share disk1 disk2
```

Options:
- `-f` Force execution (default is test mode)
- `-s SIZE` Only move files larger than SIZE (e.g., `-s 1G`)
- `-e EXT` Only move files with extension EXT
- `-c` Clobber/overwrite existing files
- `-v` Verbose output
- `-q` Quiet output

### consld8
Consolidate a user share directory from multiple disks onto a single disk:
```bash
# Test mode (default)
consld8 /path/to/share

# Force mode with specific destination disk
consld8 -f /path/to/share disk3
```

If no destination disk is specified, it automatically selects one based on current usage and available space.

**Warning**: These tools move files. Always back up important data before use.

