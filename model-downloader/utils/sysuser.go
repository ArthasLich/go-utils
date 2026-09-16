package utils

import (
	"os/exec"
	"strconv"
	"strings"
)

func GetSysUserNameByUid(uid int) (string, error) {
	out, err := exec.Command("id", strconv.Itoa(uid), "-un").Output()
	if err != nil {
		return "", err
	}
	str := strings.Trim(string(out), "\n")
	return str, nil
}
