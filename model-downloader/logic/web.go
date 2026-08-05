package logic

import (
	"fmt"
	"net/http"
	"os"

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
	var query struct {
		TaskID uint   `json:"task_id"` // 任务ID
		User   string `json:"user"`    // 用户，用户只能关闭自己创建的任务，root可以关闭所有人的任务
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: parse body failed: %v", err), Data: nil})
		return
	}
	rl := TaskMapLock.RLocker()
	rl.Lock()
	task, have := TaskMap[query.TaskID]
	rl.Unlock()
	if !have {
		c.JSON(http.StatusOK, ApiResult[any]{Code: 200, Msg: "task not found", Data: nil})
		return
	}
	// 检查状态是否为正在下载
	if task.Status != TaskStatusDownloading || task.DownloadProcess == nil {
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: "task is not downloading", Data: task})
		return
	}
	// 检查人员是否匹配
	if task.User != query.User && query.User != "root" {
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: "task user not match", Data: task})
		return
	}
	err := task.StopDownload()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	err = task.UpdateDownloadProcess()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	// 落库
	task, err = DAO.SaveOrUpdateTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	TaskMapLock.Lock()
	TaskMap[task.ID] = task
	TaskMapLock.Unlock()
	c.JSON(http.StatusOK, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "ok", Data: task})
}

// GinTaskStart 开始 中断、失败的下载任务
func GinTaskStart(c *gin.Context) {
	var query struct {
		TaskID uint   `json:"task_id"` // 任务ID
		User   string `json:"user"`    // 用户名
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: parse body failed: %v", err), Data: nil})
		return
	}
	rl := TaskMapLock.RLocker()
	rl.Lock()
	task, have := TaskMap[query.TaskID]
	rl.Unlock()
	if !have {
		c.JSON(http.StatusOK, ApiResult[any]{Code: 200, Msg: "task not found", Data: nil})
		return
	}
	if task.User != query.User && query.User != "root" {
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: "task user not match", Data: task})
		return
	}
	switch task.Status {
	case TaskStatusCanceled, TaskStatusCompleted, TaskStatusDownloading:
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: fmt.Sprintf("task status is %s", task.Status), Data: task})
		return
	}
	// 检查磁盘剩余空间
	enough, err := task.CheckDiskSize()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	// 预留10G的额外空间
	if !enough {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: "not enough disk space", Data: nil})
		return
	}
	task.cancel = nil
	task.ctx = nil
	task.DownloadProcess = nil

	// 开始下载
	err = task.StartDownload()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	err = task.UpdateDownloadProcess()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
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
	TaskMapLock.Lock()
	TaskMap[task.ID] = task
	TaskMapLock.Unlock()

	c.JSON(http.StatusAccepted, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "ok", Data: task})
}

// GinTaskCancel 取消下载任务
func GinTaskCancel(c *gin.Context) {
	var query struct {
		TaskID uint   `json:"task_id"`
		User   string `json:"user"`
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: parse body failed: %v", err), Data: nil})
		return
	}
	rl := TaskMapLock.RLocker()
	rl.Lock()
	task, have := TaskMap[query.TaskID]
	rl.Unlock()
	if !have {
		c.JSON(http.StatusOK, ApiResult[any]{Code: 200, Msg: "task not found", Data: nil})
		return
	}
	if task.User != query.User && query.User != "root" {
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: "task user not match", Data: task})
		return
	}
	switch task.Status {
	case TaskStatusCanceled, TaskStatusCompleted:
		c.JSON(http.StatusBadRequest, ApiResult[*ModelDownloadTask]{Code: 400, Msg: fmt.Sprintf("task status is %s", task.Status), Data: task})
		return
	case TaskStatusDownloading:
		err := task.StopDownload()
		if err != nil {
			c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
			return
		}
	}
	task.Status = TaskStatusCanceled
	// 删除输出日志
	outfile, _ := task.LogFile()
	os.Remove(outfile)
	// 修改状态为取消
	_, err := DAO.SaveOrUpdateTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	// 删除内存中的数据，删除数据库中的数据
	err = DAO.DeleteTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	TaskMapLock.Lock()
	delete(TaskMap, task.ID)
	TaskMapLock.Unlock()
	c.JSON(http.StatusOK, ApiResult[any]{Code: 200, Msg: "ok", Data: nil})
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
		rl := TaskMapLock.RLocker()
		rl.Lock()
		task, have := TaskMap[model.ID]
		rl.Unlock()
		if have && task != nil {
			task.UpdateDownloadProcess()
			c.JSON(http.StatusOK, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "task already exists", Data: task})
			return
		}
		c.JSON(http.StatusOK, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "task already exists", Data: model})
		return
	}

	// 检查磁盘空间是否满足
	enough, err := task.CheckDiskSize()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	// 预留10G的额外空间
	if !enough {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: "not enough disk space", Data: nil})
		return
	}

	// 开始下载模型
	err = task.StartDownload()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}

	err = task.UpdateDownloadProcess()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
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
	TaskMapLock.Lock()
	TaskMap[task.ID] = task
	TaskMapLock.Unlock()

	c.JSON(http.StatusAccepted, ApiResult[*ModelDownloadTask]{Code: 200, Msg: "ok", Data: task})
}

// GinTaskQuery 查询下载任务
func GinTaskQuery(c *gin.Context) {
	var query struct {
		User   string       `json:"user"`             // 用户，为空表示所有用户
		After  EasyTime     `json:"after"`            // 查询截止时间，在此之前的不考虑
		Status []TaskStatus `json:"status,omitempty"` // 任务状态，为空表示所有状态
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: parse body failed: %v", err), Data: nil})
		return
	}
	result := make([]*ModelDownloadTask, 0, 64)
	tasks, err := DAO.QueryTasks(query.Status, query.After.Time, query.User)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: fmt.Sprintf("error: query database failed: %v", err), Data: nil})
		return
	}
	rl := TaskMapLock.RLocker()
	rl.Lock()
	for _, v := range tasks {
		task, have := TaskMap[v.ID]
		if have {
			result = append(result, task)
		} else {
			result = append(result, &v)
		}
	}
	rl.Unlock()
	for _, v := range result {
		v.UpdateDownloadProcess()
	}
	c.JSON(http.StatusOK, ApiResult[[]*ModelDownloadTask]{Code: 200, Msg: "ok", Data: result})
}

// ApiResult API返回结果
type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}
