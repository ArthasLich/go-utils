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
	GlobalDB          *gorm.DB                    = nil                                // 数据库对象
	DAO               *dao                        = &dao{}                             // 数据库操作对象
	SavePath          string                      = ""                                 // 模型存放路径
	TaskMap           map[uint]*ModelDownloadTask = make(map[uint]*ModelDownloadTask)  // 在内存中的任务map
	TaskMapLock       sync.RWMutex                = sync.RWMutex{}                     // 任务map读写锁
	ListenPort        uint16                      = 0                                  // 监听端口
	BackgroundEvent   chan *ModelDownloadTask     = make(chan *ModelDownloadTask, 256) // 背景事件
	FailedInfoMap     map[uint]*FailedInfo        = make(map[uint]*FailedInfo)         // 记录失败任务次数，用于自动恢复，key：taskId，val失败信息
	FailedInfoMapLock sync.RWMutex                = sync.RWMutex{}                     // 读写锁
)

// FailedInfo 记录下载失败信息
type FailedInfo struct {
	Times          uint
	Task           *ModelDownloadTask
	LastStartTime  *time.Time
	LastFailedTime [3]*time.Time // 记录失败时间，索引越大，越接近当前时间
}

func NewFailedInfo(task *ModelDownloadTask) *FailedInfo {
	result := FailedInfo{
		Times:          0,
		Task:           task,
		LastStartTime:  nil,
		LastFailedTime: [3]*time.Time{nil, nil, nil},
	}
	return &result
}

// Failed 记录失败
func (fi *FailedInfo) Failed() {
	fi.Times += 1
	t := time.Now()
	for k, v := range fi.LastFailedTime {
		if v == nil {
			fi.LastFailedTime[k] = &t
			return
		}
	}
	fi.LastFailedTime[0] = fi.LastFailedTime[1]
	fi.LastFailedTime[1] = fi.LastFailedTime[2]
	fi.LastFailedTime[2] = &t
}

// LastFail 获取上次失败的时间
func (fi *FailedInfo) LastFail() *time.Time {
	for index := 2; index >= 0; index-- {
		t := fi.LastFailedTime[index]
		if t != nil {
			return t
		}
	}
	return nil
}

// Check 判断是否要重试，返回true表示可以重试
// 失败次数大于10次，或
func (fi *FailedInfo) Check() bool {
	if fi.Times > 10 {
		return false
	}
	t := fi.LastFail()
	if t == nil {
		return true
	}
	if fi.LastStartTime != nil {
		start := *fi.LastStartTime
		end := *t
		// 表示没有记录上次失败时间
		if start.After(end) {
			return true
		}
		// 运行时间小于5秒
		if end.Sub(start) <= time.Second*5 {
			return false
		}
	}
	num := 0
	for _, v := range fi.LastFailedTime {
		if v != nil {
			num++
		}
	}
	switch num {
	case 0, 1:
		return true
	case 2:
		one := *(fi.LastFailedTime[0])
		two := *(fi.LastFailedTime[1])
		if two.Sub(one) <= time.Minute*2 {
			return false
		}
	case 3:
		one := *(fi.LastFailedTime[0])
		two := *(fi.LastFailedTime[1])
		three := *(fi.LastFailedTime[2])
		if three.Sub(two) <= time.Minute*2 || three.Sub(one) <= time.Minute*5 {
			return false
		}
	}
	return true
}

func (fi *FailedInfo) Start() {
	t := time.Now()
	fi.LastStartTime = &t
	err := fi.Task.StartDownload()
	if err != nil {
		fi.Failed()
		return
	}
}

const (
	// 下载模型的命令
	ModelDownCmd = "modelscope"
	// 日志输出文件夹
	OutputDir = "/tmp/model-downloader"
	// 魔塔社区token ms-7cabf546-64dc-42cc-829d-d930a8b74ce1
	MSToken = "ms-7cabf546-64dc-42cc-829d-d930a8b74ce1" // 魔塔社区API-token，用于查询模型大小
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
