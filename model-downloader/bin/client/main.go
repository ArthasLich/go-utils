package main

import (
	"errors"
	"fmt"
	"go-utils/model-downloader/logic"
	"go-utils/model-downloader/utils"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
)

const (
	Version    = "1.0.0"
	ServerAddr = "http://10.16.1.201:9986"
)

var (
	rootCmd = &cobra.Command{
		Use:     "mdc",
		Short:   "model downloader client",
		Long:    "model downloader client",
		Version: Version,
	}

	pullCmd = &cobra.Command{
		Use:     "pull",
		Short:   "pull llm model from modelscope",
		Long:    `pull llm model from modelscope`,
		Version: Version,
		RunE:    handlePullCmd,
		Example: "mdc pull MiniMax/MiniMax-H3 (create a model download task and start download)",
	}

	listCmd = &cobra.Command{
		Use:     "ls",
		Short:   "list model download task",
		Long:    "list model download task",
		Version: Version,
		RunE:    handleListCmd,
		Example: "mdc ls",
	}

	stopCmd = &cobra.Command{
		Use:     "stop",
		Short:   "stop a model download task",
		Long:    "stop a model download task",
		Version: Version,
		RunE:    handleStopCmd,
		Example: "mdc stop 1 (stop the model download task which id is 1)",
	}

	startCmd = &cobra.Command{
		Use:     "start",
		Short:   "start a model download task",
		Long:    "start a model download task",
		Version: Version,
		RunE:    handleStartCmd,
		Example: "mds start 1 (start the model download task which id is 1)",
	}

	cancelCmd = &cobra.Command{
		Use:     "cancel",
		Short:   "cancel a model download task",
		Long:    "cancel a model download task",
		Version: Version,
		RunE:    handleCancelCmd,
		Example: "mds cancel 1 (cancel the model download task which id is 1)",
	}
	RegDuration = regexp.MustCompile(`([1-9][0-9]*)(d|w|m|y)`)
)

func init() {
	rootFS := rootCmd.PersistentFlags()
	rootFS.BoolP("version", "v", false, "Show mdc version")

	pullFS := pullCmd.Flags()
	pullFS.StringP("path", "p", "", "model save path")

	listFS := listCmd.Flags()
	listFS.String("since", "3m", "set query task before this duration, format: <number>[d|w|m|y], d = day w = week m = month y = year")
	listFS.StringSliceP("status", "s", []string{string(logic.TaskStatusDownloading), string(logic.TaskStatusFailed), string(logic.TaskStatusStopped), string(logic.TaskStatusCompleted)}, "set query task status")
	listFS.StringP("user", "u", "", "set query task of user, set all to query all user's download task")

	rootCmd.AddCommand(pullCmd, listCmd, stopCmd, cancelCmd, startCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func handlePullCmd(cmd *cobra.Command, args []string) error {
	uid := os.Getuid()
	user, err := utils.GetSysUserNameByUid(uid)
	if err != nil {
		return fmt.Errorf("get user name of %d failed: %v", uid, err)
	}
	if len(args) != 1 {
		return fmt.Errorf("command format error")
	}
	path, err := cmd.Flags().GetString("path")
	if err != nil {
		return fmt.Errorf("parse arg failed: %v", err)
	}

	var query struct {
		ModelName string `json:"model_name"`          // 模型名称，
		User      string `json:"user"`                // 用户名，
		SavePath  string `json:"save_path,omitempty"` // 保存路径，如果为空，就使用默认路径
	}
	query.ModelName = args[0]
	query.User = user
	if len(path) != 0 {
		// 校验目录
		p, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("eval path %s failed: %v", path, err)
		}
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("get path %s info failed: %v", p, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a dir", p)
		}
		query.SavePath = p
	} else {
		query.SavePath = path
	}
	var result logic.ApiResult[*logic.ModelDownloadTask]
	cli := resty.New().SetTimeout(time.Second * 5)
	resp, err := cli.R().SetBody(&query).SetResult(&result).SetError(&result).Post(fmt.Sprintf("%s/api/task/commit", ServerAddr))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	code := resp.StatusCode()
	if code == http.StatusAccepted {
		fmt.Println("commit task success")
	} else {
		fmt.Printf("commit task failed: %s\n", result.Msg)
	}
	if result.Data != nil {
		PrintTask(result.Data)
	}
	return nil
}

func handleStopCmd(cmd *cobra.Command, args []string) error {
	uid := os.Getuid()
	user, err := utils.GetSysUserNameByUid(uid)
	if err != nil {
		return fmt.Errorf("get user name of %d failed: %v", uid, err)
	}
	if len(args) != 1 {
		return fmt.Errorf("command format error")
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("parse task id '%s' failed: %v", args[0], err)
	}
	var query struct {
		TaskID uint   `json:"task_id"` // 任务ID
		User   string `json:"user"`    // 用户，用户只能关闭自己创建的任务，root可以关闭所有人的任务
	}
	query.User = user
	query.TaskID = uint(id)
	var result logic.ApiResult[*logic.ModelDownloadTask]
	cli := resty.New().SetTimeout(time.Second * 2)
	resp, err := cli.R().SetBody(&query).SetResult(&result).SetError(&result).Post(fmt.Sprintf("%s/api/task/stop", ServerAddr))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode() == 200 {
		fmt.Println("stop task success")
	} else {
		fmt.Printf("stop task failed: %s\n", result.Msg)
	}
	if result.Data != nil {
		PrintTask(result.Data)
	}
	return nil
}

func handleCancelCmd(cmd *cobra.Command, args []string) error {
	uid := os.Getuid()
	user, err := utils.GetSysUserNameByUid(uid)
	if err != nil {
		return fmt.Errorf("get user name of %d failed: %v", uid, err)
	}
	if len(args) != 1 {
		return fmt.Errorf("command format error")
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("parse task id '%s' failed: %v", args[0], err)
	}
	var query struct {
		TaskID uint   `json:"task_id"` // 任务ID
		User   string `json:"user"`    // 用户，用户只能关闭自己创建的任务，root可以关闭所有人的任务
	}
	query.User = user
	query.TaskID = uint(id)
	var result logic.ApiResult[*logic.ModelDownloadTask]
	cli := resty.New().SetTimeout(time.Second * 2)
	resp, err := cli.R().SetBody(&query).SetResult(&result).SetError(&result).Post(fmt.Sprintf("%s/api/task/cancel", ServerAddr))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode() == 200 {
		fmt.Println("cancel task success")
	} else {
		fmt.Printf("cancel task failed: %s \n", result.Msg)
	}
	if result.Data != nil {
		PrintTask(result.Data)
	}
	return nil
}

func handleListCmd(cmd *cobra.Command, args []string) error {
	// 解析参数
	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return fmt.Errorf("parse arg user failed: %v", err)
	}
	if len(user) == 0 {
		uid := os.Getuid()
		u, err := utils.GetSysUserNameByUid(uid)
		if err != nil {
			return fmt.Errorf("get user name of %d failed: %v", uid, err)
		}
		user = u
	}
	sinceStr, err := cmd.Flags().GetString("since")
	if err != nil {
		return fmt.Errorf("parse arg since failed: %v", err)
	}
	dura, err := ParseDurationStr(sinceStr)
	if err != nil {
		return fmt.Errorf("error: parse arg sinec failed: %v", err)
	}
	statusStrs, err := cmd.Flags().GetStringSlice("status")
	if err != nil {
		return fmt.Errorf("get arg status failed: %v", err)
	}
	status := make([]logic.TaskStatus, 0, len(statusStrs))
	for _, v := range statusStrs {
		s, err := logic.TaskStatusFromStr(v)
		if err != nil {
			return fmt.Errorf("parse status failed: %v", err)
		}
		status = append(status, s)
	}
	var query struct {
		User   string             `json:"user"`             // 用户，为空表示所有用户
		After  logic.EasyTime     `json:"after"`            // 查询截止时间，在此之前的不考虑
		Status []logic.TaskStatus `json:"status,omitempty"` // 任务状态，为空表示所有状态
	}
	query.User = user
	query.After = logic.EasyTime{
		Time: time.Now().Add(-1 * *dura),
	}
	query.Status = status
	var result logic.ApiResult[[]*logic.ModelDownloadTask]
	cli := resty.New().SetTimeout(time.Second * 2)
	resp, err := cli.R().SetBody(&query).SetResult(&result).SetError(&result).Post(fmt.Sprintf("%s/api/task/query", ServerAddr))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode() == 200 {
		fmt.Println("list task success")
	} else {
		fmt.Printf("list task failed: %s\n", result.Msg)
	}
	if result.Data != nil {
		PrintTasks(result.Data)
	}
	return nil
}

func handleStartCmd(cmd *cobra.Command, args []string) error {
	uid := os.Getuid()
	user, err := utils.GetSysUserNameByUid(uid)
	if err != nil {
		return fmt.Errorf("get user name of %d failed: %v", uid, err)
	}
	if len(args) != 1 {
		return fmt.Errorf("command format error")
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("parse task id '%s' failed: %v", args[0], err)
	}
	var query struct {
		TaskID uint   `json:"task_id"` // 任务ID
		User   string `json:"user"`    // 用户，用户只能关闭自己创建的任务，root可以关闭所有人的任务
	}
	query.User = user
	query.TaskID = uint(id)
	var result logic.ApiResult[*logic.ModelDownloadTask]
	cli := resty.New().SetTimeout(time.Second * 2)
	resp, err := cli.R().SetBody(&query).SetResult(&result).SetError(&result).Post(fmt.Sprintf("%s/api/task/start", ServerAddr))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	if resp.StatusCode() == 200 {
		fmt.Println("start task success")
	} else {
		fmt.Printf("start task failed: %s\n", result.Msg)
	}
	if result.Data != nil {
		PrintTask(result.Data)
	}
	return nil
}

// PrintTasks 打印多个任务
func PrintTasks(tasks []*logic.ModelDownloadTask) {
	/*
		ID  Model Size Status DownloadProgress User SavePath
	*/
	if len(tasks) == 0 {
		return
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.Before(tasks[j].UpdatedAt)
	})
	title := [7]string{"ID", "Model", "Size", "Status", "DownloadProgress", "User", "SavePath"}
	lens := [7]int{0, 0, 0, 0, 0, 0, 0}
	values := make([][]string, 0, len(tasks))
	for _, v := range tasks {
		value := make([]string, 0, 7)
		value = append(value, strconv.FormatUint(uint64(v.ID), 10))
		value = append(value, v.ModelName)
		value = append(value, utils.ParseUintI(v.ModelSize).String())
		value = append(value, string(v.Status))
		if v.Status == logic.TaskStatusCompleted {
			value = append(value, "100%")
		} else {
			value = append(value, fmt.Sprintf("%0.2f%%", v.DownloadProgress))
		}
		value = append(value, v.User)
		value = append(value, v.SavePath)
		values = append(values, value)
	}
	for k, v := range title {
		lens[k] = len(v)
		for _, vv := range values {
			lens[k] = max(lens[k], len(vv[k]))
		}
	}
	fmt.Printf("ID%sModel%sSize%sStatus%sDownloadProgress%sUser%sSavePath\n",
		strings.Repeat(" ", max(1, lens[0]-1)),
		strings.Repeat(" ", max(1, lens[1]-4)),
		strings.Repeat(" ", max(1, lens[2]-3)),
		strings.Repeat(" ", max(lens[3]-5, 1)),
		strings.Repeat(" ", max(lens[4]-15, 1)),
		strings.Repeat(" ", max(lens[5]-3, 1)))
	for _, v := range values {
		fmt.Printf("%s%s%s%s%s%s%s%s%s%s%s%s%s\n", v[0], strings.Repeat(" ", lens[0]-len(v[0])+1),
			v[1], strings.Repeat(" ", lens[1]-len(v[1])+1),
			v[2], strings.Repeat(" ", lens[2]-len(v[2])+1),
			v[3], strings.Repeat(" ", lens[3]-len(v[3])+1),
			v[4], strings.Repeat(" ", lens[4]-len(v[4])+1),
			v[5], strings.Repeat(" ", lens[5]-len(v[5])+1),
			v[6])
	}

}

// PrintTask 打印单个任务
func PrintTask(task *logic.ModelDownloadTask) {
	/*
		ID  Model Size Status DownloadProgress User SavePath
	*/
	if task == nil {
		return
	}
	title := [7]string{"ID", "Model", "Size", "Status", "DownloadProgress", "User", "SavePath"}
	lens := [7]int{0, 0, 0, 0, 0, 0, 0}
	values := make([]string, 0, 7)
	values = append(values, strconv.FormatUint(uint64(task.ID), 10))
	values = append(values, task.ModelName)
	values = append(values, utils.ParseUintI(task.ModelSize).String())
	values = append(values, string(task.Status))
	if task.Status == logic.TaskStatusCompleted {
		values = append(values, "100%")
	} else {
		values = append(values, fmt.Sprintf("%0.2f%%", task.DownloadProgress))
	}
	values = append(values, task.User)
	values = append(values, task.SavePath)
	for k, v := range values {
		lens[k] = max(len(title[k]), len(v))
	}
	fmt.Printf("ID%sModel%sSize%sStatus%sDownloadProgress%sUser%sSavePath\n",
		strings.Repeat(" ", max(1, lens[0]-1)),
		strings.Repeat(" ", max(1, lens[1]-4)),
		strings.Repeat(" ", max(1, lens[2]-3)),
		strings.Repeat(" ", max(lens[3]-5, 1)),
		strings.Repeat(" ", max(lens[4]-15, 1)),
		strings.Repeat(" ", max(lens[5]-3, 1)))
	fmt.Printf("%s%s%s%s%s%s%s%s%s%s%s%s%s\n",
		values[0], strings.Repeat(" ", lens[0]-len(values[0])+1),
		values[1], strings.Repeat(" ", lens[1]-len(values[1])+1),
		values[2], strings.Repeat(" ", lens[2]-len(values[2])+1),
		values[3], strings.Repeat(" ", lens[3]-len(values[3])+1),
		values[4], strings.Repeat(" ", lens[4]-len(values[4])+1),
		values[5], strings.Repeat(" ", lens[5]-len(values[5])+1),
		values[6])
}

func ParseDurationStr(str string) (*time.Duration, error) {
	items := RegDuration.FindStringSubmatch(str)
	if len(items) == 0 {
		return nil, errors.New("arg format illegal")
	}
	num, err := strconv.ParseInt(items[1], 10, 64)
	if err != nil {
		return nil, err
	}
	var unit time.Duration
	switch items[2] {
	case "d":
		unit = time.Hour * 24
	case "w":
		unit = time.Hour * 24 * 7
	case "m":
		unit = time.Hour * 24 * 7 * 30
	case "y":
		unit = time.Hour * 24 * 7 * 365
	default:
		return nil, errors.New("error format, sholud match <number>(d|w|m|y)")
	}
	result := unit * time.Duration(num)
	return &result, nil
}
