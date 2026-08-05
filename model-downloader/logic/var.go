package logic

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
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

	MSToken    = "ms-7cabf546-64dc-42cc-829d-d930a8b74ce1" // 魔塔社区API-token
	HTTPClient = resty.New().SetTimeout(time.Second).SetAuthToken(fmt.Sprintf("Bearer %s", MSToken))
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
