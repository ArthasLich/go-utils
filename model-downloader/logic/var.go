package logic

import (
	"sync"

	"gorm.io/gorm"
)

var (
	GlobalDB   *gorm.DB = nil        // 数据库对象
	DAO        *dao     = &dao{}     // 数据库操作对象
	SavePath   string   = ""         // 模型存放路径
	PythonCmd  string   = ""         // 查询模型大小使用的python3可执行文件名
	TaskMap    sync.Map = sync.Map{} // 在内存中的任务map，仅存放正在下载的和等待下载的任务
	ListenPort uint16   = 0          // 监听端口
)

const (
	// 下载模型的命令
	ModelDownCmd = "modelscope"
	QueryCmd     = "from modelscope.hub.api import HubApi; files=HubApi().get_model_files('MODEL_NAME'); print(f'{sum(f.get(\"Size\",0) for f in files)}')"
	OutputDir    = "/tmp/model-downloader"
)
