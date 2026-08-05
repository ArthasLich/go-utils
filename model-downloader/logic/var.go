package logic

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	GlobalDB        *gorm.DB                    = nil                                // 数据库对象
	DAO             *dao                        = &dao{}                             // 数据库操作对象
	SavePath        string                      = ""                                 // 模型存放路径
	TaskMap         map[uint]*ModelDownloadTask = make(map[uint]*ModelDownloadTask)  // 在内存中的任务map
	TaskMapLock     sync.RWMutex                = sync.RWMutex{}                     // 任务map读写锁
	ListenPort      uint16                      = 0                                  // 监听端口
	BackgroundEvent chan *ModelDownloadTask     = make(chan *ModelDownloadTask, 256) // 背景事件

	MSToken = "ms-7cabf546-64dc-42cc-829d-d930a8b74ce1" // 魔塔社区API-token
)

/*
	MiniMax/MiniMax-H3查询模型大小不对
*/

const (
	// 下载模型的命令
	ModelDownCmd = "modelscope"

	// 日志输出文件夹
	OutputDir = "/tmp/model-downloader"

	// 魔塔社区token ms-7cabf546-64dc-42cc-829d-d930a8b74ce1
)

// 自定义的简单时间
type EasyTime struct {
	time.Time
}

func (et EasyTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, et.Time.Format(time.DateTime))), nil
}

func (et *EasyTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return errors.New("input is empty")
	}
	t, err := time.Parse(time.DateTime, strings.Trim(string(b), `"`))
	if err != nil {
		return err
	}
	et.Time = t
	return nil
}
