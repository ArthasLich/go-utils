package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

const (
	testStr = `Created
Up About a minute
Up 2 days
Up 2 weeks
Exited (0) 2 weeks ago
Up 2 weeks (healthy)
Up 8 days
Up 7 weeks
Up 7 weeks (healthy)
Up 7 weeks (healthy)
Exited (0) 7 weeks ago
Exited (0) 7 weeks ago
Exited (0) 7 weeks ago
Exited (0) 7 weeks ago
Up 7 weeks
Up 7 weeks
Exited (255) 2 months ago
Exited (255) 2 months ago
Exited (255) 2 months ago
Exited (255) 2 months ago
Created
Exited (0) 3 months ago
Up 7 weeks
Exited (0) 2 months ago
Exited (0) 7 weeks ago
Exited (0) 2 months ago
Exited (255) 2 months ago
Exited (1) 2 weeks ago
Up 7 weeks
Up 7 weeks
Exited (255) 2 months ago
Up 7 weeks
Exited (137) 3 months ago
Exited (137) 3 months ago
Up 7 weeks
Exited (255) 3 months ago
Exited (255) 3 months ago
Exited (255) 3 months ago
Exited (130) 4 months ago
Exited (1) 5 months ago
Exited (127) 6 months ago
Exited (255) 3 months ago
Exited (255) 3 months ago
Exited (129) 11 months ago
Up 7 weeks
Up 7 weeks
Up 7 weeks
Up 7 weeks
Up 7 weeks
Up 7 weeks
Exited (128) 3 months ago
Exited (128) 3 months ago
Up 40 seconds (health: starting)
Exited (128) 3 months ago
Exited (128) 7 weeks ago
Up 7 weeks (healthy)
Exited (128) 3 months ago
Exited (128) 7 weeks ago
Up 7 weeks (healthy)
Exited (255) 15 months ago
Up 7 weeks
Up 7 weeks`
)

func TestDocker(t *testing.T) {
	cli, err := client.New(client.WithAPIVersionFromEnv())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cli.Close()

	result, err := cli.ContainerList(context.Background(), client.ContainerListOptions{
		All:   true,
		Limit: 9999,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, c := range result.Items {
		tt := time.Unix(c.Created, 0)
		if c.State == container.StateRunning {
			top, err := cli.ContainerTop(t.Context(), c.ID, client.ContainerTopOptions{Arguments: []string{"-o", "pid,cmd"}})
			if err != nil {
				t.Error(err)
			}
			pids := len(top.Processes)
			t.Log(tt, pids)
			if pids == 1 {
				t.Log("  ", top.Processes[0][1])
			}
		}

	}
}

func TestGetContainerInfo(t *testing.T) {
	cli, err := client.New(client.WithAPIVersionFromEnv())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cli.Close()
	ci, err := GetContainerInfo(cli)
	if err != nil {
		t.Error(err)
		return
	}
	for k, v := range ci {
		t.Log(string(k[:12]), v.Name, v.StatusTime.String())
	}
}

func TestParseBefore(t *testing.T) {
	dd, err := parseBefore("4m")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(dd)
	t.Log(dd.ToDuration().String())
}

func TestPrintContainers(t *testing.T) {
	cli, err := client.New(client.WithAPIVersionFromEnv())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cli.Close()
	ci, err := GetContainerInfo(cli)
	if err != nil {
		t.Error(err)
		return
	}
	PrintContainers(ci)
}

func TestPrintImages(t *testing.T) {
	cli, err := client.New(client.WithAPIVersionFromEnv())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cli.Close()
	imgs, err := cli.ImageList(context.Background(), client.ImageListOptions{All: true})
	if err != nil {
		t.Error(err)
		return

	}
	for _, v := range imgs.Items {
		t.Logf("id: %s, userd container: %d,create: %v repotag: %v\n", v.ID[7:19], v.Containers, time.Unix(v.Created, 0).Format(time.DateTime), v.RepoTags)
	}
	imgMap := make(map[string]*image.Summary)
	for _, v := range imgs.Items {
		imgMap[v.ID] = &v
	}
	PrintImages(imgMap)
}
