package utils

import "testing"

func TestGetDiskUsage(t *testing.T) {
	size, avail, usage, _ := GetDiskUsage("/data")
	t.Logf("size: %d, avail: %d, usage %.2f%%", size/1024/1024/1024, avail/1024/1024/1024, usage)
}

func TestGetSysUserNameByUid(t *testing.T) {
	user, err := GetSysUserNameByUid(0)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(user)
}

func TestRegexp(t *testing.T) {
	m := regUID.MatchString("uid=1000(sugon) gid=1000(sugon) groups=1000(sugon)")
	t.Log(m)
}

func TestSize(t *testing.T) {
	size := ParseUintI(10238710024)
	t.Logf("%+v", size.String())
}
