package logic

import (
	"context"
	"go-utils/model-downloader/utils"
	"log"
	"os"
	"time"
)

/*
	定义后台任务逻辑：
	- 定时检查磁盘空间
	- 定时检查模型下载进度并更新数据库和内存
	- 定时清理下载日志文件
	- 定期删除不需要在内存里存放的任务
	- 使用管道进行任务状态更新（failed、completed）

	内存里仅存放状态为creating、downloading、failed和stopped的下载任务
	数据库中仅删除状态为canceled的下载任务
*/

// Daemon 后台任务逻辑
func Daemon(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case task := <-BackgroundEvent: // 接收到背景任务
			_, err := DAO.SaveOrUpdateTask(task)
			if err != nil {
				log.Printf("error: background update task failed: %v", err)
			}
			// 如果是下载完成，则删除日志
			if task.Status == TaskStatusCompleted {
				outf, _ := task.LogFile()
				os.Remove(outf)
			}
		case <-ticker.C:
			background()
		case <-ctx.Done():
			log.Println("info: Daemon ctx done")
			return
		}
	}
}

// background
func background() {
	// 非默认路径的模型下载任务
	savePathMap := make(map[string]*ModelDownloadTask)

	// 默认路径的模型下载任务
	defaultPathTask := make([]*ModelDownloadTask, 0, 32)

	taskToDelFromMem := make([]*ModelDownloadTask, 0, 32)
	rl := TaskMapLock.RLocker()
	rl.Lock()
	for _, task := range TaskMap {
		switch task.Status {
		case TaskStatusCreating, TaskStatusFailed, TaskStatusStopped:
			// 不执行动作
		case TaskStatusDownloading:
			// 记录下载路径
			if task.Status == TaskStatusDownloading {
				if task.DefaultPath {
					defaultPathTask = append(defaultPathTask, task)
				} else {
					savePathMap[task.SavePath] = task
				}
			}
		case TaskStatusCompleted, TaskStatusCanceled:
			// 从内存中删除
			taskToDelFromMem = append(taskToDelFromMem, task)
		}
	}
	rl.Unlock()

	taskToUpdate := make([]*ModelDownloadTask, 0, 64)

	// 更新下载进度
	for _, v := range defaultPathTask {
		if err := v.UpdateDownloadProcess(); err != nil {
			log.Printf("error: update download process failed: %v", err)
		}
		taskToUpdate = append(taskToUpdate, v)
	}
	for _, v := range savePathMap {
		if err := v.UpdateDownloadProcess(); err != nil {
			log.Printf("error: update download process failed: %v", err)
		}
		taskToUpdate = append(taskToUpdate, v)
	}

	// 停止磁盘不足的下载任务
	_, alive, _, err := utils.GetDiskUsage(SavePath)
	if err != nil {
		log.Printf("error: get disk usage failed, path: %s, err: %v", SavePath, err)
		return
	}
	if alive < ((1 << 30) * 10) {
		// 剩余空间小于10G
		for _, v := range defaultPathTask {
			v.UpdateDownloadProcess()
			v.StopDownload()
		}
	}
	for k, v := range savePathMap {
		_, alive, _, err := utils.GetDiskUsage(k)
		if err != nil {
			log.Printf("error: get disk usage failed, path: %s, err: %v", k, err)
			continue
		}
		if alive < ((1 << 30) * 10) {
			// 剩余空间小于10G
			v.UpdateDownloadProcess()
			v.StopDownload()
		}
	}
	// 落库
	_, err = DAO.UpdateTaskBatch(taskToUpdate)
	if err != nil {
		log.Printf("error: update database failed: %v", err)
	}

	// 删除不需要的内存
	TaskMapLock.Lock()
	for _, v := range taskToDelFromMem {
		delete(TaskMap, v.ID)
	}
	TaskMapLock.Unlock()
}
