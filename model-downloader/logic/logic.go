package logic

import (
	"fmt"
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

// DownloadModel 下载模型
// func DownloadModel(task *ModelDownloadTask) (*ModelDownloadTask, error) {
// 	// modelscope download --model Tencent-Hunyuan/Hy3 --local_dir
// 	err := os.MkdirAll(task.SavePath, 0644)
// 	if err != nil {
// 		return task, fmt.Errorf("error: create directory failed: %v", err)
// 	}
// 	cmdStr, err := task.DownLoadCmd()
// 	if err != nil {
// 		return task, fmt.Errorf("error: generate download command failed: %v", err)
// 	}
// 	cmd := exec.Command("bash", "-c", cmdStr)
// 	err = cmd.Start()
// 	if err != nil {
// 		return task, fmt.Errorf("error: start command failed: %v", err)
// 	}
// 	pid := cmd.Process.Pid
// 	task.DownloadPid = &pid
// 	task.Status = TaskStatusDownloading
// 	task.DownloadProcess = cmd.Process

// 	go func(cmd *exec.Cmd, task *ModelDownloadTask) {
// 		err := cmd.Wait()
// 		if err != nil {
// 			select {
// 			case <-task.skipAutoChangeStatus:

// 			default:
// 				task.Status = TaskStatusFailed
// 			}
// 		} else {
// 			task.Status = TaskStatusCompleted
// 		}
// 		task.DownloadPid = nil
// 	}(cmd, task)

// 	return task, nil
// }
