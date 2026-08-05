package logic

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestNewTask(t *testing.T) {
	task, err := NewTask("microsoft/Mage-Flow", "test", "test")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", task)
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

func TestGetModelSizeByAPI(t *testing.T) {
	var result struct {
		Code int `json:"Code"`
		Data struct {
			StorageSize uint64 `json:"StorageSize"`
		} `json:"Data"`
		Message   string `json:"Message"`
		RequestId string `json:"RequestId"`
		Success   bool   `json:"Success"`
	}
	cli := resty.New().SetTimeout(time.Second).SetAuthToken(fmt.Sprintf("Bearer %s", MSToken))
	_, err := cli.R().SetResult(&result).ExpectContentType("application/json").Get("https://modelscope.cn/api/v1/models/moonshotai/Kimi-K3")
	if err != nil {
		t.Error(err)
		return
	}
	s := result.Data.StorageSize
	t.Log(s)
	t.Log(s / 1024 / 1024 / 1024)
}

func TestEasyTime(t *testing.T) {
	es := EasyTime{
		Time: time.Now(),
	}
	b, _ := es.MarshalJSON()
	t.Log(string(b))
	var ess EasyTime
	ess.UnmarshalJSON(b)
	t.Log(ess)
}
