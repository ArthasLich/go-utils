package logic

import "testing"

func TestDownloadModel(t *testing.T) {
	task := &ModelDownloadTask{
		ModelName: "test",
		SavePath:  "test",
	}
	task, err := DownloadModel(task)
	if err != nil {
		t.Error(err)
	}
}

func TestNewTask(t *testing.T) {
	PythonCmd = "python3.12"
	task, err := NewTask("microsoft/Mage-Flow", "test", "test")
	if err != nil {
		t.Error(err)
	}
	t.Log(task)
}
