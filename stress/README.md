# stress

This is a container image and wrapper script for `stress`: https://linux.die.net/man/1/stress

This is for easy execution of `stress` without installing the package (e.g. on a distro where the package is not in the repos).

It also includes a script which auto-identifies CPU count and free memory, then hogs it. You can obviously bypass it if you don't like how it works.
