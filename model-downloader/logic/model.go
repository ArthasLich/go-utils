package logic

import (
	"context"
	"errors"
	"fmt"
	"go-utils/model-downloader/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"gorm.io/gorm"
)

type dao struct{}

/*
	内存里仅存放状态为creating、downloading、failed和stopped的下载任务
*/

type TaskStatus string

const (
	TaskStatusCreating    TaskStatus = "creating"    // 创建中
	TaskStatusDownloading TaskStatus = "downloading" // 下载中
	TaskStatusCompleted   TaskStatus = "completed"   // 完成
	TaskStatusFailed      TaskStatus = "failed"      // 不知名错误
	TaskStatusCanceled    TaskStatus = "canceled"    // 取消
	TaskStatusStopped     TaskStatus = "stopped"     // 停止
)

type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ModelDownloadTask 下载任务
type ModelDownloadTask struct {
	Model                               // 基础字段
	ModelName        string             `json:"model_name"`                                 // 模型名称
	User             string             `json:"user"`                                       // 下载用户
	SavePath         string             `json:"save_path"`                                  // 模型下载路径
	Status           TaskStatus         `json:"status"`                                     // 状态
	ModelSize        uint64             `json:"model_size"`                                 // 模型大小
	DefaultPath      bool               `json:"default_path"`                               // 是否使用了默认路径
	DownloadedSize   uint64             `gorm:"download_size" json:"downloaded_size"`       // 已下载的大小
	DownloadProgress float64            `gorm:"download_progress" json:"download_progress"` // 下载进度，百分比
	DownloadPid      *int               `gorm:"-" json:"download_pid"`                      // 下载进程的pid
	DownloadProcess  *os.Process        `gorm:"-" json:"-"`                                 // 下载进程
	ctx              context.Context    `gorm:"-" json:"-"`                                 // 上下文，控制任务停止
	cancel           context.CancelFunc `gorm:"-" json:"-"`                                 // 取消方法，控制任务停止
	ctxUsed          bool               `gorm:"-" json:"-"`                                 // 上下文是否被使用
}

// NewTask 创建下载任务，如果查询模型大小失败则返回错误，如果savePath为空则使用默认路径
func NewTask(modelName string, user string, savePath string) (*ModelDownloadTask, error) {
	ctx, cancel := context.WithCancel(context.Background())
	result := ModelDownloadTask{
		ModelName: modelName,
		User:      user,
		SavePath:  savePath,
		Status:    TaskStatusCreating,
		ctx:       ctx,
		cancel:    cancel,
		ctxUsed:   false,
	}
	size, err := QueryModelSize(modelName)
	if err != nil {
		return nil, err
	}
	result.ModelSize = size
	if savePath == "" {
		result.SavePath = filepath.Join(SavePath, result.ModelShortName())
		result.DefaultPath = true
	} else {
		// 获取保存路径的绝对路径
		result.SavePath = savePath
	}
	return &result, nil
}

// LogFile 模型下载输出文件和错误文件
func (mdt *ModelDownloadTask) LogFile() (string, error) {
	err := os.MkdirAll(OutputDir, 0755)
	if err != nil {
		return "", err
	}
	shortName := mdt.ModelShortName()
	return fmt.Sprintf("%s/%s.out", OutputDir, shortName), nil
}

// StartDownload 启动下载任务
func (mdt *ModelDownloadTask) StartDownload() error {
	if mdt.Status == TaskStatusDownloading {
		return errors.New("task status is downloading")
	}
	// modelscope download --model Tencent-Hunyuan/Hy3 --local_dir
	err := os.MkdirAll(mdt.SavePath, 0644)
	if err != nil {
		return fmt.Errorf("error: create directory failed: %v", err)
	}
	out, err := mdt.LogFile()
	if err != nil {
		return fmt.Errorf("error: get download log file path failed: %v", err)
	}
	logfile, err := os.OpenFile(out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error: open or create log file failed: %v", err)
	}
	cmd := exec.Command(ModelDownCmd, "download", "--model", mdt.ModelName, "--local_dir", mdt.SavePath)
	cmd.Stdout = logfile
	cmd.Stderr = logfile
	err = cmd.Start()
	if err != nil {
		logfile.Close()
		return fmt.Errorf("error: start command failed: %v", err)
	}
	pid := cmd.Process.Pid
	mdt.DownloadPid = &pid
	mdt.Status = TaskStatusDownloading
	mdt.DownloadProcess = cmd.Process
	if mdt.ctxUsed {
		ctx, cancel := context.WithCancel(context.Background())
		mdt.ctx = ctx
		mdt.cancel = cancel
		mdt.ctxUsed = false
	}
	go func(cmd *exec.Cmd, task *ModelDownloadTask, f *os.File) {
		ch := make(chan error)
		skip := false
		go func() {
			err := cmd.Wait()
			if skip {
				return
			}
			ch <- err
		}()
		select {
		case <-task.ctx.Done(): // 任务人为取消
			skip = true
		case err := <-ch: // 任务完成或发生错误
			if err != nil {
				task.Status = TaskStatusFailed
			} else {
				task.Status = TaskStatusCompleted
			}
			task.DownloadPid = nil
			task.DownloadProcess = nil
			task.cancel()
			task.cancel = nil
			task.ctx = nil
			task.ctxUsed = true
			BackgroundEvent <- task // 通知背景任务自动落库
		}
		close(ch)
		f.Close()
	}(cmd, mdt, logfile)
	return nil
}

func (mdt *ModelDownloadTask) StopDownload() error {
	if mdt.Status != TaskStatusDownloading {
		return errors.New("task status is not downloading")
	}
	if mdt.DownloadProcess == nil {
		return errors.New("download process is nil, something went wrong")
	}
	mdt.cancel()
	mdt.ctxUsed = true
	proc, err := process.NewProcess(int32(mdt.DownloadProcess.Pid))
	if err != nil {
		return fmt.Errorf("error: create process failed: %v", err)
	}
	child, err := proc.Children()
	if err != nil {
		return fmt.Errorf("error: get children processes failed: %v", err)
	}
	for _, c := range child {
		c.Kill()
	}
	proc.Kill()
	mdt.DownloadPid = nil
	mdt.DownloadProcess = nil
	mdt.Status = TaskStatusStopped
	return nil
}

// CheckDiskSize 检查磁盘空间是否满足需求
func (mdt *ModelDownloadTask) CheckDiskSize() (bool, error) {
	var path string
	var err error
	if mdt.DefaultPath {
		path, err = filepath.EvalSymlinks(SavePath)
	} else {
		path, err = filepath.EvalSymlinks(mdt.SavePath)
	}
	if err != nil {
		return false, err
	}
	_, avail, _, err := utils.GetDiskUsage(path)
	if err != nil {
		return false, err
	}
	return avail > (mdt.ModelSize + (1<<30)*10), nil
}

func (mdt *ModelDownloadTask) ModelShortName() string {
	items := strings.Split(mdt.ModelName, "/")
	return items[len(items)-1]
}

// UpdateDownloadStatus 更新下载进度
func (mdt *ModelDownloadTask) UpdateDownloadProcess() error {
	if mdt.Status != TaskStatusDownloading {
		return nil
	}
	output, err := exec.Command("du", "-sb", mdt.SavePath).Output()
	if err != nil {
		return err
	}
	items := strings.Fields(strings.Trim(string(output), "\n"))
	if len(items) != 2 {
		return errors.New("error: invalid output")
	}
	size, err := strconv.ParseUint(items[0], 10, 64)
	if err != nil {
		return err
	}
	mdt.DownloadedSize = size
	percent := (float64(size) / float64(mdt.ModelSize) * 100)
	mdt.DownloadProgress = percent
	return nil
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
	task, err := gorm.G[ModelDownloadTask](GlobalDB).Where("model_name = ?", modelName).First(context.Background())
	return &task, err
}

// UpdateTaskBatch 批量更新
func (d *dao) UpdateTaskBatch(tasks []*ModelDownloadTask) ([]*ModelDownloadTask, error) {
	if len(tasks) == 0 {
		return nil, nil
	}
	err := GlobalDB.Transaction(func(tx *gorm.DB) error {
		for _, v := range tasks {
			m := ModelDownloadTask{
				Model: Model{
					ID: v.ID,
				},
			}
			err := tx.Model(&m).Updates(v).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	return tasks, err
}

// ListTaskByStatus 列出指定状态的任务
func (d *dao) ListTaskByStatus(status []TaskStatus) ([]ModelDownloadTask, error) {
	if len(status) == 0 {
		return nil, nil
	}
	return gorm.G[ModelDownloadTask](GlobalDB).Where("status IN ?", status).Find(context.Background())
}
