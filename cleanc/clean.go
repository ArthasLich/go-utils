package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/spf13/pflag"
)

var (
	containerBefore = pflag.StringP("container", "c", "", "clean container before this time, format: <number>(d|w|m|y)")
	imageBefore     = pflag.StringP("image", "i", "", "clean image before this time, format: <number>(d|w|m|y)")

	cleanIdel = pflag.Bool("idle", false, "delete idle running container")
	help      = pflag.BoolP("help", "h", false, "show help")
	dryRun    = pflag.BoolP("dry-run", "d", false, "just show which container will delete without --dry-run, not delete")

	// 匹配docker容器状态的时间字符串
	regDuration = regexp.MustCompile(`(?i)^.*[^0-9](\d+)\s+(seconds?|minutes?|hours?|days?|weeks?|months?|years?).*$`)

	// 匹配参数cleanBefore
	regArgBefore = regexp.MustCompile(`(?i)^(\d+)([dwmy])$`)
)

type TimeUnit int64

const (
	TUSeconds TimeUnit = 1
	TUMinutes TimeUnit = TUSeconds * 60
	TUHours   TimeUnit = TUMinutes * 60
	TUDays    TimeUnit = TUHours * 24
	TUWeeks   TimeUnit = TUDays * 7
	TUMonths  TimeUnit = TUDays * 30
	TUYears   TimeUnit = TUDays * 365
	TUUnknow  TimeUnit = 0
)

func TUFromStr(str string) TimeUnit {
	switch strings.ToLower(str) {
	case "seconds", "second":
		return TUSeconds
	case "minutes", "minute":
		return TUMinutes
	case "hours", "hour":
		return TUHours
	case "days", "day":
		return TUDays
	case "weeks", "week":
		return TUWeeks
	case "months", "month":
		return TUMonths
	case "years", "year":
		return TUYears
	default:
		return TUUnknow
	}
}

func (tu TimeUnit) String() string {
	switch tu {
	case TUSeconds:
		return "second(s)"
	case TUMinutes:
		return "minute(s)"
	case TUHours:
		return "hour(s)"
	case TUDays:
		return "day(s)"
	case TUWeeks:
		return "week(s)"
	case TUMonths:
		return "month(s)"
	case TUYears:
		return "year(s)"
	default:
		return "unknown"
	}
}

// DockerDuration docker status显示的时间
type DockerDuration struct {
	Num  int64    // 数字
	Unit TimeUnit // 单位
}

func (dd DockerDuration) String() string {
	return fmt.Sprintf("%d %s", dd.Num, dd.Unit.String())
}

func (dd DockerDuration) ToDuration() time.Duration {
	return time.Duration(dd.Num*int64(dd.Unit)) * time.Second
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	ID         string                   // 容器id
	Name       string                   // 容器名
	CreateTime time.Time                // 容器创建时间
	Status     container.ContainerState // 容器状态
	StatusTime *DockerDuration          // 状态持续时间
	PidNumber  int                      // 进程数
	ImageID    string                   // docker镜像id
	FirstCmd   string                   // 第一个容器进程的命令
}

func main() {
	pflag.Parse()

	if *help {
		fmt.Println("cleanc: clean old docker container and images")
		pflag.Usage()
		return
	}
	var before *DockerDuration = nil
	if len(*containerBefore) != 0 {
		tmp, err := parseBefore(*containerBefore)
		if err != nil {
			log.Fatal(err)
		}
		before = tmp
	}

	var ibefore *DockerDuration = nil
	if len(*imageBefore) != 0 {
		tmp, err := parseBefore(*imageBefore)
		if err != nil {
			log.Fatal(err)
		}
		ibefore = tmp
	}

	cli, err := client.New(client.WithAPIVersionFromEnv())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	ci, err := GetContainerInfo(cli)
	if err != nil {
		log.Fatalf("error: get container info failed: %v", err)
	}

	if before != nil {
		ctod := filterContainer(ci, before, cleanIdel)
		if *dryRun {
			PrintContainers(ctod)
			return
		}

		for k, v := range ctod {
			if v.Status == container.StateRunning {
				cli.ContainerKill(context.Background(), k, client.ContainerKillOptions{})
			}
			cli.ContainerRemove(context.Background(), k, client.ContainerRemoveOptions{})
			fmt.Printf("deleted %s %s \n", v.ID[:12], v.Name)
		}
	}

	if ibefore != nil {
		imgs, err := cli.ImageList(context.Background(), client.ImageListOptions{All: true})
		if err != nil {
			log.Fatalf("error: list docker image failed: %v", err)
		}
		now := time.Now()
		t := now.Add(ibefore.ToDuration() * -1)
		for _, v := range imgs.Items {
			if v.Containers > 0 {
				continue
			}
			if time.Unix(v.Created, 0).Before(t) {
				cli.ImageRemove(context.Background(), v.ID, client.ImageRemoveOptions{Force: false})
			}
		}
	}
}

func filterContainer(ci map[string]*ContainerInfo, before *DockerDuration, cleanIdel *bool) map[string]*ContainerInfo {
	result := make(map[string]*ContainerInfo)
	now := time.Now()
	b := now.Add(-1 * before.ToDuration())
	for k, v := range ci {
		if v.CreateTime.After(b) {
			continue
		}
		switch v.Status {
		case container.StateExited:
			result[k] = v
		case container.StateRunning:
			if *cleanIdel && isIdleCmd(v.FirstCmd) && v.PidNumber == 1 {
				result[k] = v
			}
		}
	}
	return result
}

func ContainerName(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.TrimPrefix(s[0], "/")
}

func GetContainerInfo(cli *client.Client) (map[string]*ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	listResult, err := cli.ContainerList(ctx, client.ContainerListOptions{
		All:   true,
		Limit: 9999,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]*ContainerInfo)
	for _, v := range listResult.Items {
		ci := ContainerInfo{
			ID:         v.ID,
			Name:       ContainerName(v.Names),
			CreateTime: time.Unix(v.Created, 0),
			Status:     v.State,
			FirstCmd:   v.Command,
			ImageID:    v.ImageID,
		}
		if ci.Status == container.StateRunning {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			top, err := cli.ContainerTop(ctx, v.ID, client.ContainerTopOptions{Arguments: []string{"-o", "pid,cmd"}})
			cancel()
			if err != nil {
				continue
			}
			ci.PidNumber = len(top.Processes)
			if ci.PidNumber == 1 {
				ci.FirstCmd = top.Processes[0][1]
			}
		}
		switch v.State {
		case container.StateExited, container.StateRunning:
			s, _ := parseStatus(v.Status)
			if s != nil {
				ci.StatusTime = s
			}
		}
		result[v.ID] = &ci
	}
	return result, nil
}

// parseStatus 解析docker容器的status字段
func parseStatus(str string) (*DockerDuration, error) {
	str = strings.ToLower(str)
	var dd *DockerDuration
	if strings.Contains(str, "about a minute") {
		dd = &DockerDuration{
			Num:  1,
			Unit: TUMinutes,
		}
	} else {
		items := regDuration.FindStringSubmatch(str)
		if len(items) == 0 {
			return nil, fmt.Errorf("error: can't find duation string: %s", str)
		}
		unit := TUFromStr(items[2])
		num, err := strconv.ParseInt(items[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error: parse int error: %s", items[1])
		}
		dd = &DockerDuration{
			Num:  num,
			Unit: unit,
		}
	}
	return dd, nil
}

// isIdleCmd 判断是否为空占命令
func isIdleCmd(str string) bool {
	if strings.HasSuffix(str, "bash") || str == "sh" || strings.HasPrefix(str, "sleep") {
		return true
	}
	if strings.HasSuffix(str, "/bin/bash") || strings.HasSuffix(str, "/bin/sh") {
		return true
	}
	return false
}

func parseBefore(str string) (*DockerDuration, error) {
	items := regArgBefore.FindStringSubmatch(str)
	if len(items) == 0 {
		return nil, fmt.Errorf("error: parse arg error: %s", str)
	}
	i, err := strconv.ParseInt(items[1], 10, 64)
	if err != nil {
		return nil, err
	}
	result := DockerDuration{
		Num: i,
	}
	switch items[2] {
	case "d":
		result.Unit = TUDays
	case "w":
		result.Unit = TUWeeks
	case "m":
		result.Unit = TUMonths
	case "y":
		result.Unit = TUYears
	default:
		return nil, fmt.Errorf("error: arg format illegel: %s", str)
	}
	return &result, nil
}

// PrintContainers 打印要删除的容器信息
func PrintContainers(m map[string]*ContainerInfo) {
	if len(m) == 0 {
		return
	}
	/*
		ContainerID Name Create Status
	*/
	nameLen := 1
	l := make([]*ContainerInfo, 0, len(m))
	for _, v := range m {
		nameLen = max(nameLen, len(v.Name))
		l = append(l, v)
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].CreateTime.After(l[j].CreateTime)
	})
	space := max(nameLen-4, 1)
	fmt.Printf("ContainerID   Name%sCreate              Status\n", strings.Repeat(" ", space+2))
	ll := 0
	name := ""
	for _, v := range l {
		ll = len(v.Name)
		name = v.Name + strings.Repeat(" ", nameLen-ll+1)
		if v.StatusTime != nil {
			fmt.Printf("%s  %s %s %s %s\n", v.ID[:12], name, v.CreateTime.Format(time.DateTime), v.Status, v.StatusTime.String())
		} else {
			fmt.Printf("%s  %s %s\n", v.ID[:12], name, v.CreateTime.Format(time.DateTime))
		}
	}
}
