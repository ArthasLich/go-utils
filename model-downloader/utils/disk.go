package utils

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// GetDiskUsage 获取磁盘使用情况，返回总空间，可用空间和使用率，单位为字节和百分比
func GetDiskUsage(path string) (uint64, uint64, float64, error) {
	output, err := exec.Command("df", "--output=used,avail,size", path).Output()
	if err != nil {
		return 0, 0, 0, err
	}
	lines := strings.Split(strings.Trim(string(output), "\n"), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, errors.New("error: invalid disk info")
	}
	diskInfo := strings.Fields(lines[1])
	if len(diskInfo) != 3 {
		return 0, 0, 0, errors.New("error: invalid disk info")
	}
	used, _ := strconv.ParseUint(diskInfo[0], 10, 64)
	avail, _ := strconv.ParseUint(diskInfo[1], 10, 64)
	size, _ := strconv.ParseUint(diskInfo[2], 10, 64)

	return size * 1024, avail * 1024, float64(used) / float64(size) * 100, nil
}
