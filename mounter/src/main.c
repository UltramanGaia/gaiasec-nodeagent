#define _GNU_SOURCE

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <sched.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <sys/sysmacros.h>
#include <sys/types.h>
#include <unistd.h>

enum {
    EXIT_USAGE = 2,
    EXIT_IO = 3,
    EXIT_NS = 4,
    EXIT_MOUNT = 5,
};

struct mount_entry {
    int major_id;
    int minor_id;
    char mountpoint[PATH_MAX];
    char fstype[128];
};

struct mount_context {
    char tempdir[PATH_MAX];
    char devpath[PATH_MAX];
    char hostfs[PATH_MAX];
    int mounted_hostfs;
};

static void failf(int code, const char* fmt, ...) {
    va_list args;
    va_start(args, fmt);
    vfprintf(stderr, fmt, args);
    va_end(args);
    fputc('\n', stderr);
    exit(code);
}

static void fail_errno(int code, const char* action, const char* path) {
    if (path == NULL || path[0] == '\0') {
        failf(code, "%s: %s", action, strerror(errno));
    }
    failf(code, "%s %s: %s", action, path, strerror(errno));
}

static bool has_path_prefix(const char* path, const char* prefix) {
    size_t prefix_len = strlen(prefix);
    if (strncmp(path, prefix, prefix_len) != 0) {
        return false;
    }

    if (prefix_len == 1 && prefix[0] == '/') {
        return true;
    }

    return path[prefix_len] == '\0' || path[prefix_len] == '/';
}

static void ensure_absolute_path(const char* path, const char* flag_name) {
    if (path == NULL || path[0] != '/') {
        failf(EXIT_USAGE, "%s must be an absolute path", flag_name);
    }
}

static int mkdir_p(const char* path, mode_t mode) {
    char buf[PATH_MAX];
    char* cursor;

    if (snprintf(buf, sizeof(buf), "%s", path) >= (int)sizeof(buf)) {
        errno = ENAMETOOLONG;
        return -1;
    }

    for (cursor = buf + 1; *cursor != '\0'; cursor++) {
        if (*cursor == '/') {
            *cursor = '\0';
            if (mkdir(buf, mode) != 0 && errno != EEXIST) {
                return -1;
            }
            *cursor = '/';
        }
    }

    if (mkdir(buf, mode) != 0 && errno != EEXIST) {
        return -1;
    }

    return 0;
}

static int ensure_parent_dir(const char* path, mode_t mode) {
    char parent[PATH_MAX];
    char* slash;

    if (snprintf(parent, sizeof(parent), "%s", path) >= (int)sizeof(parent)) {
        errno = ENAMETOOLONG;
        return -1;
    }

    slash = strrchr(parent, '/');
    if (slash == NULL) {
        errno = EINVAL;
        return -1;
    }

    if (slash == parent) {
        return 0;
    }

    *slash = '\0';
    return mkdir_p(parent, mode);
}

static void ensure_dest_path(const char* dest, const struct stat* src_stat) {
    struct stat dest_stat;

    if (stat(dest, &dest_stat) == 0) {
        return;
    }
    if (errno != ENOENT) {
        fail_errno(EXIT_IO, "stat failed for", dest);
    }

    if (S_ISDIR(src_stat->st_mode)) {
        if (mkdir_p(dest, src_stat->st_mode & 0777) != 0) {
            fail_errno(EXIT_IO, "mkdir failed for", dest);
        }
        return;
    }

    if (ensure_parent_dir(dest, 0777) != 0) {
        fail_errno(EXIT_IO, "mkdir failed for parent of", dest);
    }

    int fd = open(dest, O_CREAT | O_CLOEXEC, src_stat->st_mode & 0777);
    if (fd < 0) {
        fail_errno(EXIT_IO, "create failed for", dest);
    }
    close(fd);
}

static void find_mount_for_source(const char* src, struct mount_entry* out) {
    FILE* file = fopen("/proc/self/mountinfo", "r");
    if (file == NULL) {
        fail_errno(EXIT_IO, "open failed for", "/proc/self/mountinfo");
    }

    char* line = NULL;
    size_t cap = 0;
    bool found = false;
    size_t best_len = 0;

    while (getline(&line, &cap, file) != -1) {
        char* separator = strstr(line, " - ");
        if (separator == NULL) {
            continue;
        }

        *separator = '\0';
        char* right = separator + 3;

        int major_id = 0;
        int minor_id = 0;
        char mountpoint[PATH_MAX];
        if (sscanf(line, "%*s %*s %d:%d %*s %4095s %*s", &major_id, &minor_id, mountpoint) != 3) {
            continue;
        }

        if (!has_path_prefix(src, mountpoint)) {
            continue;
        }

        size_t mount_len = strlen(mountpoint);
        if (found && mount_len <= best_len) {
            continue;
        }

        char fstype[128];
        char device[PATH_MAX];
        if (sscanf(right, "%127s %4095s %*s", fstype, device) < 2) {
            continue;
        }

        out->major_id = major_id;
        out->minor_id = minor_id;
        snprintf(out->mountpoint, sizeof(out->mountpoint), "%s", mountpoint);
        snprintf(out->fstype, sizeof(out->fstype), "%s", fstype);
        found = true;
        best_len = mount_len;
    }

    free(line);
    fclose(file);

    if (!found) {
        failf(EXIT_MOUNT, "no mount point found for %s", src);
    }
}

static int enter_mount_namespace(int pid) {
    char path[64];
    snprintf(path, sizeof(path), "/proc/%d/ns/mnt", pid);

    int fd = open(path, O_RDONLY | O_CLOEXEC);
    if (fd < 0) {
        return -1;
    }

    int result = syscall(__NR_setns, fd, 0);
    close(fd);
    return result;
}

static int root_is_readonly(void) {
    FILE* file = fopen("/proc/self/mountinfo", "r");
    if (file == NULL) {
        return 0;
    }

    char* line = NULL;
    size_t cap = 0;
    int readonly = 0;

    while (getline(&line, &cap, file) != -1) {
        char mountpoint[PATH_MAX];
        char options[PATH_MAX];
        if (sscanf(line, "%*s %*s %*s %*s %4095s %4095s", mountpoint, options) != 2) {
            continue;
        }

        if (strcmp(mountpoint, "/") != 0) {
            continue;
        }

        if (strcmp(options, "ro") == 0 || strncmp(options, "ro,", 3) == 0) {
            readonly = 1;
        }
        break;
    }

    free(line);
    fclose(file);
    return readonly;
}

static void remount_root_if_ro(void) {
    if (!root_is_readonly()) {
        return;
    }

    if (mount(NULL, "/", NULL, MS_REMOUNT | MS_RELATIME, NULL) != 0) {
        fail_errno(EXIT_MOUNT, "remount failed for", "/");
    }
}

static void cleanup_context(struct mount_context* ctx) {
    if (ctx->mounted_hostfs) {
        umount(ctx->hostfs);
        ctx->mounted_hostfs = 0;
    }

    if (ctx->hostfs[0] != '\0') {
        rmdir(ctx->hostfs);
    }
    if (ctx->devpath[0] != '\0') {
        unlink(ctx->devpath);
    }
    if (ctx->tempdir[0] != '\0') {
        rmdir(ctx->tempdir);
    }
}

static void prepare_context(struct mount_context* ctx) {
    memset(ctx, 0, sizeof(*ctx));

    snprintf(ctx->tempdir, sizeof(ctx->tempdir), "/tmp/mounter.XXXXXX");
    if (mkdtemp(ctx->tempdir) == NULL) {
        fail_errno(EXIT_IO, "mkdtemp failed for", "/tmp/mounter.XXXXXX");
    }

    snprintf(ctx->devpath, sizeof(ctx->devpath), "%s/block-dev", ctx->tempdir);
    snprintf(ctx->hostfs, sizeof(ctx->hostfs), "%s/rootfs", ctx->tempdir);
    if (mkdir(ctx->hostfs, 0700) != 0) {
        cleanup_context(ctx);
        fail_errno(EXIT_IO, "mkdir failed for", ctx->hostfs);
    }
}

static void bind_mount_path(const char* src, const char* dest, const struct mount_entry* entry) {
    struct mount_context ctx;
    prepare_context(&ctx);

    if (mknod(ctx.devpath, S_IFBLK | 0600, makedev(entry->major_id, entry->minor_id)) != 0) {
        cleanup_context(&ctx);
        fail_errno(EXIT_MOUNT, "mknod failed for", ctx.devpath);
    }

    if (mount(ctx.devpath, ctx.hostfs, entry->fstype, MS_MGC_VAL, NULL) != 0) {
        cleanup_context(&ctx);
        fail_errno(EXIT_MOUNT, "mount failed for", ctx.hostfs);
    }
    ctx.mounted_hostfs = 1;

    char source_in_ns[PATH_MAX];
    if (snprintf(source_in_ns, sizeof(source_in_ns), "%s%s", ctx.hostfs, src) >= (int)sizeof(source_in_ns)) {
        cleanup_context(&ctx);
        failf(EXIT_MOUNT, "source path is too long after namespace remap");
    }

    if (mount(source_in_ns, dest, NULL, MS_BIND | MS_SILENT, NULL) != 0) {
        cleanup_context(&ctx);
        fail_errno(EXIT_MOUNT, "bind mount failed for", dest);
    }

    cleanup_context(&ctx);
}

static void cmd_remount_root_if_ro(int target_pid) {
    if (enter_mount_namespace(target_pid) != 0) {
        fail_errno(EXIT_NS, "setns failed for pid", "");
    }
    remount_root_if_ro();
}

static void cmd_ensure_path_visible(int target_pid, const char* src, const char* dest) {
    ensure_absolute_path(src, "--src");
    ensure_absolute_path(dest, "--dest");

    struct stat src_stat;
    if (stat(src, &src_stat) != 0) {
        fail_errno(EXIT_IO, "stat failed for", src);
    }

    struct mount_entry entry;
    memset(&entry, 0, sizeof(entry));
    find_mount_for_source(src, &entry);

    if (enter_mount_namespace(target_pid) != 0) {
        fail_errno(EXIT_NS, "setns failed for pid", "");
    }

    remount_root_if_ro();
    ensure_dest_path(dest, &src_stat);
    bind_mount_path(src, dest, &entry);
}

static void usage(void) {
    fprintf(stderr,
            "Usage:\n"
            "  mounter ensure-path-visible --target-pid <pid> --src <path> --dest <path>\n"
            "  mounter remount-root-if-ro --target-pid <pid>\n");
}

int main(int argc, char** argv) {
    if (argc < 2) {
        usage();
        return EXIT_USAGE;
    }

    const char* command = argv[1];
    int target_pid = -1;
    const char* src = NULL;
    const char* dest = NULL;

    for (int i = 2; i < argc; i++) {
        if (strcmp(argv[i], "--target-pid") == 0 && i + 1 < argc) {
            target_pid = atoi(argv[++i]);
        } else if (strcmp(argv[i], "--src") == 0 && i + 1 < argc) {
            src = argv[++i];
        } else if (strcmp(argv[i], "--dest") == 0 && i + 1 < argc) {
            dest = argv[++i];
        } else {
            usage();
            return EXIT_USAGE;
        }
    }

    if (target_pid <= 0) {
        failf(EXIT_USAGE, "--target-pid is required");
    }

    if (strcmp(command, "ensure-path-visible") == 0) {
        if (src == NULL || dest == NULL) {
            failf(EXIT_USAGE, "--src and --dest are required");
        }
        cmd_ensure_path_visible(target_pid, src, dest);
        return 0;
    }

    if (strcmp(command, "remount-root-if-ro") == 0) {
        cmd_remount_root_if_ro(target_pid);
        return 0;
    }

    usage();
    return EXIT_USAGE;
}
