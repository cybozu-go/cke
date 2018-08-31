package placemat

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cybozu-go/log"
)

var (
	// cgroupV1ctrls helps identifying the v1 controllers.
	// http://manpages.ubuntu.com/manpages/bionic/en/man7/cgroups.7.html
	cgroupV1ctrls = map[string]bool{
		"cpu":        true,
		"cpuacct":    true,
		"cpuset":     true,
		"memory":     true,
		"devices":    true,
		"freezer":    true,
		"net_cls":    true,
		"blkio":      true,
		"perf_event": true,
		"net_prio":   true,
		"hugetlb":    true,
		"pids":       true,
	}
)

const rootPath = "/placemat-root"

func umount(mp string) error {
	return exec.Command("umount", mp).Run()
}

func bindMount(src, dest string) error {
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}
	log.Info("bind mount", map[string]interface{}{
		"src":  src,
		"dest": dest,
	})
	c := exec.Command("mount", "--bind", src, dest)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return c.Run()
}

func mount(fs, dest, options string) error {
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}
	log.Info("mount", map[string]interface{}{
		"fs":   fs,
		"dest": dest,
	})
	c := exec.Command("mount", "-t", fs, "-o", options, fs, dest)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return c.Run()
}

func makeCgroupSymlinks(dest string, opts []string) error {
	ctrls := make([]string, 0, 3)
	for _, opt := range opts {
		if cgroupV1ctrls[opt] {
			ctrls = append(ctrls, opt)
		}
	}

	if len(ctrls) < 2 {
		return nil
	}

	dir := filepath.Dir(dest)
	base := filepath.Base(dest)
	for _, ctrl := range ctrls {
		sym := filepath.Join(dir, ctrl)
		_, err := os.Stat(sym)
		if err == nil {
			// something exists
			continue
		}

		err = os.Symlink(base, sym)
		if err != nil {
			return err
		}
	}

	return nil
}

// Rootfs is a fake root filesystem in order to fool rkt
// into believing that the system is running without systemd
// by hiding /run/systemd/system directory.
type Rootfs struct {
	root        string
	mountPoints []string
}

// Path returns the absolute filesystem path to the fake rootfs.
func (r *Rootfs) Path() string {
	return r.root
}

// Destroy unmounts the root filesystem and remove the mount point directory.
func (r *Rootfs) Destroy() error {
	var err error

	l := len(r.mountPoints)
	for i := 0; i < l; i++ {
		e := umount(r.mountPoints[l-i-1])
		if e != nil {
			err = e
		}
	}

	if err != nil {
		log.Error("failed to umount", map[string]interface{}{
			"root":      r.root,
			log.FnError: err,
		})
		return err
	}

	return os.RemoveAll(r.root)
}

// NewRootfs creates a new root filesystem.
func NewRootfs() (*Rootfs, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(rootPath, 0700)
	if err != nil {
		return nil, err
	}

	err = bindMount("/", rootPath)
	if err != nil {
		return nil, err
	}

	mountPoints := []string{rootPath}
	defer func() {
		l := len(mountPoints)
		for i := 0; i < l; i++ {
			umount(mountPoints[l-i-1])
		}
	}()

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		mp := fields[1]
		fs := fields[2]
		options := fields[3]
		opts := strings.Split(options, ",")
		dest := filepath.Join(rootPath, mp)

		switch {
		case mp == "/":
			continue
		case strings.HasPrefix(mp, rootPath):
			continue
		case strings.HasPrefix(mp, "/boot"):
			continue
		}

		readonly := false
		for _, opt := range opts {
			if opt == "ro" {
				readonly = true
				break
			}
		}
		if fs == "tmpfs" && readonly {
			for i := range opts {
				if opts[i] == "ro" {
					opts[i] = "rw"
					break
				}
			}
			options = strings.Join(opts, ",")
		}

		switch fs {
		case "tmpfs", "proc", "sysfs", "securityfs", "cgroup", "cgroup2", "debugfs", "fusectl", "configfs":
			err = mount(fs, dest, options)
			if err != nil {
				return nil, err
			}
			mountPoints = append(mountPoints, dest)

		case "autofs", "pstore", "efivarfs", "fuse.lxcfs":
			// ignore

		default:
			err = bindMount(mp, dest)
			if err != nil {
				return nil, err
			}
			mountPoints = append(mountPoints, dest)
		}

		if fs == "cgroup" {
			err = makeCgroupSymlinks(dest, opts)
			if err != nil {
				return nil, err
			}
		}
	}

	ret := &Rootfs{rootPath, mountPoints}
	mountPoints = nil
	return ret, nil
}

// CleanupRootfs unmount all remaining mounts.
func CleanupRootfs() error {
	data, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}

	var paths []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if strings.HasPrefix(fields[1], rootPath) {
			paths = append(paths, fields[1])
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, p := range paths {
		log.Info("unmount", map[string]interface{}{
			"target": p,
		})
		err := umount(p)
		if err != nil {
			return err
		}
	}

	return nil
}
