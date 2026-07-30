package logic

import (
	"fmt"
	"go-utils/model-downloader/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// InitRoute 初始化路由
func InitRoute(engine *gin.Engine) {
	// 提交下载任务
	engine.POST("/api/task/commit", GinTaskCommit)
	// 查询下载任务
	engine.POST("/api/task/query", GinTaskQuery)
	// 取消任务
	engine.POST("/api/task/cancel", GinTaskCancel)
	// 暂停任务	
	engine.POST("/api/task/stop", GinTaskStop)
	// 开始任务
	engine.POST("/api/task/start", GinTaskStart)
}

func GinTaskStop(c *gin.Context) {
}

func GinTaskStart(c *gin.Context) {
}

func GinTaskCancel(c *gin.Context) {
}

// GinTaskCommit 提交下载任务
func GinTaskCommit(c *gin.Context) {
	var query struct {
		ModelName string `json:"model_name"`          // 模型名称，
		User      string `json:"user"`                // 用户名，
		SavePath  string `json:"save_path,omitempty"` // 保存路径，如果为空，就使用默认路径
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: parse body failed: %v", err), Data: nil})
		return
	}
	task, err := NewTask(query.ModelName, query.User, query.SavePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	model, err := DAO.GetTaskByModelName(task.ModelName)
	if err == nil && model != nil {
		// 任务已存在，返回任务信息
		c.JSON(http.StatusOK, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "task already exists", Data: model})
		return
	}

	// 检查磁盘空间是否满足
	var alive uint64
	if task.DefaultPath {
		_, alive, _, err = utils.GetDiskUsage(SavePath)
	} else {
		_, alive, _, err = utils.GetDiskUsage(task.SavePath)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	if alive < (task.ModelSize + 10*1<<30) {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: "not enough disk space", Data: nil})
		return
	}

	// 落库
	tempTask, err := DAO.SaveOrUpdateTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	task = tempTask
	// 内存里也保存一份
	TaskMap.Store(task.ID, task)

	// 开始下载模型
	task, err = DownloadModel(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	c.JSON(http.StatusAccepted, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "ok", Data: task})
}

// GinTaskQuery 查询下载任务
func GinTaskQuery(c *gin.Context) {
	result := make([]*ModelDownloadTask, 0)
	TaskMap.Range(func(key, value any) bool {
		result = append(result, value.(*ModelDownloadTask))
		return true
	})
	c.JSON(http.StatusOK, ApiResult[[]*ModelDownloadTask]{Code: 200, Msg: "ok", Data: result})
}

// ApiResult API返回结果
type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}
