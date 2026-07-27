package logic

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Init 初始化方法
func Init(dbfile string, savePath string, pythonCmd string, port uint16) error {
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

	PythonCmd = pythonCmd

	web := gin.Default()
	InitRoute(web)

	return nil
}

// Run 运行方法
func Run() error {
	// 启动web服务

	// 启动磁盘容量检查

	// 启动下载任务处理

	return nil
}
