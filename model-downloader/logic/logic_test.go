package logic

import (
	"os"
	"os/exec"
	"testing"
	"time"
)


func TestNewTask(t *testing.T) {
	PythonCmd = "python3.12"
	task, err := NewTask("microsoft/Mage-Flow", "test", "test")
	if err != nil {
		t.Error(err)
	}
	t.Log(task)
}

func TestKillTask(t *testing.T) {
	os.Mkdir("model", 0777)
	cmd := exec.Command("bash", "-c", "modelscope download --model XYZAILab/XYZ-Aquila-mini --local_dir model")
	err := cmd.Start()
	if err != nil {
		t.Errorf("error: start command failed: %v", err)
		return
	}
	t.Log("command started with pid:", cmd.Process.Pid)
	go func() {
		err := cmd.Wait()
		if err != nil {
			t.Errorf("error: command finished with error: %v", err)
		}
	}()
	time.Sleep(time.Second * 10)
	cmd.Process.Kill()
	t.Log("command killed")
}
