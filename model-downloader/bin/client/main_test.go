package main

import (
	"go-utils/model-downloader/logic"
	"testing"
)

func TestRegDuration(t *testing.T) {
	RegDuration.MatchString("39m")

}

func TestPrintTask(t *testing.T) {
	task := logic.ModelDownloadTask{}
	task.ID = 4000
	task.ModelName = "deepseek/DeepSeek-V4-Pro"
	task.User = "liming6"
	task.ModelSize = 123456789
	task.DownloadProgress = 99.887
	task.SavePath = "/public/opendas/DL_DATA/llm-models/DeepSeek-V4-Pro"
	PrintTask(&task)

}


func TestPrintTasks(t *testing.T) {
	task := logic.ModelDownloadTask{}
	task.ID = 4000
	task.ModelName = "deepseek/DeepSeek-V4-Pro"
	task.User = "liming6"
	task.ModelSize = 1234
	task.DownloadProgress = 99.887
	task.SavePath = "/public/opendas/DL_DATA/llm-models/DeepSeek-V4-Pro"
	tasks := make([]*logic.ModelDownloadTask,0,4)

	task1 := task
	task1.ModelSize = 987654321
	tasks = append(tasks, &task, &task1)


	PrintTasks(tasks)

}