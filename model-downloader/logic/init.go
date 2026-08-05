package logic

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Init 初始化方法
func Init(dbfile string, savePath string, port uint16) error {
	fileInfo, err := os.Stat(savePath)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("error: save path is not dir: %s", savePath)
	}
	SavePath = savePath

	db, err := gorm.Open(sqlite.Open(dbfile), &gorm.Config{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&ModelDownloadTask{})
	if err != nil {
		return err
	}
	GlobalDB = db
	ListenPort = port
	err = InitDownload()
	if err != nil {
		// 这个错误不打断启动
		log.Printf("error: init download failed: %v", err)
	}
	return nil
}

// Run 运行方法
func Run() error {
	// 启动web服务
	web := gin.Default()
	InitRoute(web)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动后台任务
	go Daemon(ctx)

	return web.Run(fmt.Sprintf(":%d", ListenPort))
}

// InitDownload 初始化下载任务
func InitDownload() error {
	stat := []TaskStatus{TaskStatusDownloading, TaskStatusFailed, TaskStatusStopped}
	task, err := DAO.ListTaskByStatus(stat)
	if err != nil {
		return err
	}
	errs := make([]error, 0, 4)
	TaskMapLock.Lock()
	for _, v := range task {
		TaskMap[v.ID] = &v
		switch v.Status {
		case TaskStatusDownloading:
			v.Status = TaskStatusFailed
			err := v.StartDownload()
			if err != nil {
				errs = append(errs, err)
			}
		case TaskStatusFailed:
			err := v.StartDownload()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	TaskMapLock.Unlock()
	return errors.Join(errs...)
}
