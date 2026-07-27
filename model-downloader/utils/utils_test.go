package utils

import "testing"

func TestGetDiskUsage(t *testing.T) {
	size, avail, usage, _ := GetDiskUsage("/data")
	t.Logf("size: %d, avail: %d, usage %.2f%%", size/1024/1024/1024, avail/1024/1024/1024, usage)
}
