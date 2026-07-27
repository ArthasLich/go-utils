package logic

import (
	"context"
	"errors"
	"fmt"
	"go-utils/model-downloader/utils"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

type dao struct{}

type TaskStatus string

const (
	TaskStatusCreating    TaskStatus = "creating"
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
)

// ModelDownloadTask 下载任务
type ModelDownloadTask struct {
	gorm.Model                  // 基础字段
	ModelName        string     // 模型名称
	User             string     // 下载用户
	SavePath         string     // 模型下载路径
	Status           TaskStatus // 状态
	ModelSize        uint64     // 模型大小
	DownloadedSize   *int64     `gorm:"-"` // 已下载的大小
	DownloadPid      *int       `gorm:"-"` // 下载进程的pid
	DownloadProgress *float64   `gorm:"-"` // 下载进度，百分比
}

func NewTask(modelName string, user string, savePath string) (*ModelDownloadTask, error) {
	result := ModelDownloadTask{
		ModelName: modelName,
		User:      user,
		SavePath:  savePath,
		Status:    TaskStatusCreating,
	}
	size, err := QueryModelSize(modelName)
	if err != nil {
		return nil, err
	}
	result.ModelSize = size
	return &result, nil
}

// DownLoadCmd 生成下载命令
func (mdt *ModelDownloadTask) DownLoadCmd() (string, error) {
	f, err := mdt.LogFile()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s download --model %s --local_dir %s 2>>%s >>%s", ModelDownCmd, mdt.ModelName, mdt.SavePath, f, f), nil
}

// LogFile 模型下载输出文件
func (mdt *ModelDownloadTask) LogFile() (string, error) {
	err := os.MkdirAll(OutputDir, 0755)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s.log", OutputDir, strings.ReplaceAll(mdt.ModelName, "/", "_")), nil
}

// CheckDiskSize 检查磁盘空间是否满足需求
func (mdt *ModelDownloadTask) CheckDiskSize() (bool, error) {
	path, err := filepath.EvalSymlinks(mdt.SavePath)
	if err != nil {
		return false, err
	}
	_, avail, _, err := utils.GetDiskUsage(path)
	if err != nil {
		return false, err
	}
	if avail < mdt.ModelSize {
		return false, nil
	}
	return true, nil
}

func (d *dao) QueryTask(user *string, status *TaskStatus, model *string) ([]ModelDownloadTask, error) {
	db := gorm.G[ModelDownloadTask](GlobalDB)
	var chain gorm.ChainInterface[ModelDownloadTask]
	if user != nil {
		chain = db.Where("user = ?", *user)
	}
	if status != nil {
		if chain != nil {
			chain = chain.Where("status = ?", *status)
		} else {
			chain = db.Where("status = ?", *status)
		}
	}
	if model != nil {
		if chain != nil {
			chain = chain.Where("model_name = ?", *model)
		} else {
			chain = db.Where("model_name = ?", *model)
		}
	}
	if chain != nil {
		return chain.Order("created_at desc").Find(context.Background())
	}
	return db.Order("created_at desc").Find(context.Background())
}

func (d *dao) SaveOrUpdateTask(task *ModelDownloadTask) (*ModelDownloadTask, error) {
	if task == nil {
		return nil, errors.New("error: arg nil")
	}
	if task.ID == 0 {
		err := gorm.G[ModelDownloadTask](GlobalDB).Create(context.Background(), task)
		return task, err
	} else {
		_, err := gorm.G[ModelDownloadTask](GlobalDB).Where("id = ?", task.ID).Updates(context.Background(), *task)
		return task, err
	}
}

func (d *dao) DeleteTask(task *ModelDownloadTask) error {
	_, err := gorm.G[ModelDownloadTask](GlobalDB).Where("id = ?", task.ID).Delete(context.Background())
	return err
}

func (d *dao) GetTaskByID(id uint) *ModelDownloadTask {
	task, err := gorm.G[ModelDownloadTask](GlobalDB).Where("id = ?", id).First(context.Background())
	if err != nil {
		return nil
	}
	return &task
}

func (d *dao) GetTaskByModelName(modelName string) (*ModelDownloadTask, error) {
	var task ModelDownloadTask
	tx := GlobalDB.Unscoped().Where("model_name = ?", modelName).First(&task)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if task.ID == 0 {
		return nil, nil
	}
	return &task, nil
}
