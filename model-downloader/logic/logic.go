package logic

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// QueryModelSize 查询模型大小，单位是字节
func QueryModelSize(model string) (uint64, error) {
	output, err := exec.Command(PythonCmd, "-c", strings.ReplaceAll(QueryCmd, "MODEL_NAME", model)).Output()
	if err != nil {
		return 0, fmt.Errorf("error: run shell command `%s` failed: %v", fmt.Sprintf("%s -c %s", PythonCmd, strings.ReplaceAll(QueryCmd, "MODEL_NAME", model)), err)
	}
	size := strings.Trim(string(output), "\n")
	usize, err := strconv.ParseUint(size, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error: parse uint64 failed: %v", err)
	}
	return usize, nil
}

// DownloadModel 下载模型，返回下载命令
func DownloadModel(task *ModelDownloadTask) error {
	// modelscope download --model Tencent-Hunyuan/Hy3 --local_dir
	err := os.MkdirAll(task.SavePath, 0644)
	if err != nil {
		return fmt.Errorf("error: create directory failed: %v", err)
	}
	cmdStr, err := task.DownLoadCmd()
	if err != nil {
		return fmt.Errorf("error: generate download command failed: %v", err)
	}
	cmd := exec.Command("bash", "-c", cmdStr)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error: start command failed: %v", err)
	}
	pid := cmd.Process.Pid
	task.DownloadPid = &pid
	task.Status = TaskStatusDownloading
	return cmd.Wait()
}
