package logic

import (
	"testing"
)

func TestQueryTask(t *testing.T) {
	err := Init("test.db", "/tmp", 0)
	if err != nil {
		t.Error(err)
		return
	}
	user := "liming6"
	task, err := DAO.QueryTask(&user, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	for _, v := range task {
		t.Logf("%+v", v)
	}
}

func TestSaveOrUpdateTask(t *testing.T) {
	err := Init("test.db", "/tmp", 0)
	if err != nil {
		t.Error(err)
		return
	}
	task := &ModelDownloadTask{
		ModelName: "deepseek-r1",
		SavePath:  "/public/llm-models",
		Status:    TaskStatusCreating,
		User:      "liming6",
		ModelSize: 123456789,
	}
	task, err = DAO.SaveOrUpdateTask(task)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(task)
}

func TestDeleteTask(t *testing.T) {
	err := Init("test.db", "/tmp", 0)
	if err != nil {
		t.Error(err)
		return
	}
	task := &ModelDownloadTask{}
	task.ID = 1
	err = DAO.DeleteTask(task)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("Delete task successfully")
}

func TestGetTaskByID(t *testing.T) {
	err := Init("test.db", "/tmp", 0)
	if err != nil {
		t.Error(err)
		return
	}
	task := DAO.GetTaskByID(2)
	if task != nil {
		t.Logf("%+v", task)
	}
}

func TestGetModelSize(t *testing.T) {
	model := "MiniMax/MiniMax-H4"
	size, err := QueryModelSize(model)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(size)
}

func TestQueryModel(t *testing.T) {
	model := "deepseek-r1"
	Init("test.db", "/tmp", 0)
	m, err := DAO.GetTaskByModelName(model)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(m)
}
