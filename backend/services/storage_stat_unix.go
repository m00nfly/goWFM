//go:build !windows

package services

import (
	"os"
	"strconv"
	"syscall"
)

func diskSpace(path string) (available uint64, total uint64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), stat.Blocks * uint64(stat.Bsize), nil
}

func fileDiskUsage(_ string, info os.FileInfo) (int64, string) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Blocks > 0 {
		identity := strconv.FormatUint(uint64(stat.Dev), 10) + ":" + strconv.FormatUint(uint64(stat.Ino), 10)
		return stat.Blocks * 512, identity
	}
	return info.Size(), ""
}
