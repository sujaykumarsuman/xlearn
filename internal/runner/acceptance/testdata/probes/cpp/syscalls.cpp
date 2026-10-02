// Section C (syscalls), C++: arg names one call from t3 §2.4 A1's dangerous set (or clone with a
// namespace flag). The exec filter must kill it with SIGSYS; "survived: ..." on fd 4 means the
// filter let it through (a failure, whatever the call then returned).
#include <sys/syscall.h>
#include <sys/uio.h>

class Solution {
public:
    string probe(string arg) {
        static char buf[512];
        long nr = -1;
        long a[6] = {0, 0, 0, 0, 0, 0};
        if (arg == "io_uring_setup") { nr = SYS_io_uring_setup; a[0] = 1; a[1] = (long)buf; }
        else if (arg == "bpf") { nr = SYS_bpf; a[0] = 0; a[1] = (long)buf; a[2] = 72; }
        else if (arg == "perf_event_open") { nr = SYS_perf_event_open; a[0] = (long)buf; a[2] = -1; a[3] = -1; }
        else if (arg == "userfaultfd") { nr = SYS_userfaultfd; a[0] = 0x80000; }
        else if (arg == "keyctl") { nr = SYS_keyctl; a[0] = 0; a[1] = -4; }
        else if (arg == "add_key") { nr = SYS_add_key; a[0] = (long)"user"; a[1] = (long)"xl-probe"; a[2] = (long)"x"; a[3] = 1; a[4] = -2; }
        else if (arg == "ptrace") { nr = SYS_ptrace; a[0] = 0; }
        else if (arg == "process_vm_readv") {
            static iovec iov{buf, 8};
            nr = SYS_process_vm_readv; a[0] = getpid(); a[1] = (long)&iov; a[2] = 1; a[3] = (long)&iov; a[4] = 1;
        }
        else if (arg == "mount") { nr = SYS_mount; a[0] = (long)"none"; a[1] = (long)"/w"; a[2] = (long)"tmpfs"; }
        else if (arg == "unshare_newuser") { nr = SYS_unshare; a[0] = 0x10000000; }
        else if (arg == "setns") { nr = SYS_setns; a[0] = -1; }
        else if (arg == "clone_newns") { nr = SYS_clone; a[0] = 0x00020000 | 17; }
        else if (arg == "socket_netlink") { nr = SYS_socket; a[0] = 16; a[1] = 3; a[2] = 0; }
        else return "bad probe " + arg;
        fprintf(stderr, "calling %s\n", arg.c_str());
        long rc = syscall(nr, a[0], a[1], a[2], a[3], a[4], a[5]);
        return "survived: " + arg + " rc=" + to_string(rc) + " errno=" + to_string(errno);
    }
};
