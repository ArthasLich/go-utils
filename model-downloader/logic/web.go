package logic

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InitRoute 初始化路由
func InitRoute(engine *gin.Engine) {
	// 提交下载任务
	engine.POST("/api/task/commit", GinTaskCommit)
	// 查询下载任务
	engine.GET("/api/task/query", GinTaskQuery)
}

// GinTaskCommit 提交下载任务
func GinTaskCommit(c *gin.Context) {
	var query struct {
		ModelName string `json:"model_name"`
		User      string `json:"user"`
		SavePath  string `json:"save_path"`
	}
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, ApiResult[any]{Code: 400, Msg: err.Error(), Data: nil})
		return
	}
	task, err := NewTask(query.ModelName, query.User, query.SavePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResult[any]{Code: 500, Msg: err.Error(), Data: nil})
		return
	}
	DAO.QueryTask(nil, nil, &task.ModelName)
}

// GinTaskQuery 查询下载任务
func GinTaskQuery(c *gin.Context) {
}

// ApiResult API返回结果
type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}
