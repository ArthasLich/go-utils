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
	ListenPort = port
	return nil
}

// Run 运行方法
func Run() error {
	// 启动web服务
	web := gin.Default()
	InitRoute(web)

	// 启动后台任务
	go Daemon()
	
	// 启动下载任务处理
	go InitDownload()

	return web.Run(fmt.Sprintf(":%d", ListenPort))
}


// InitDownload 初始化下载任务
func InitDownload() {

}