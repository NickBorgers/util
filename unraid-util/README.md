# unraid-util

I have started using unraid as a converged data storage and workload environment. 
The simplicity it provides comes at a cost, which is that I feel the command line interface on the base OS is not well suited to diagnosing problems.

An example is that `tcpdump` is not available, which makes it very difficult for me to diagnose problems.

I got around this by creating a container image which slept forever, giving me something to `exec` into and take actions including running `tcpdump`.

This is a published version of that image so others can use it and I can publish it in the unraid app store.
