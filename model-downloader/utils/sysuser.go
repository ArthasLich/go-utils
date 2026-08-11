package utils

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func GetSysUserNameByUid(uid int) (string, error) {
	out, err := exec.Command("id", strconv.Itoa(uid)).Output()
	if err != nil {
		return "", err
	}
	str := strings.Trim(string(out), "\n")
	items := regUID.FindStringSubmatch(str)
	if len(items) == 0 {
		return "", errors.New("id output match regexp failed")
	}
	return items[1], nil
}
