# readme

cleanc： docker容器、镜像清理工具

用法：
```
# 显示帮助信息
cleanc --help
cleanc: clean old docker container and images
Usage of ./cleanc:
  -c, --container string   clean container before this time, format: <number>(d|w|m|y)
  -d, --dry-run            just show which container will delete without --dry-run, not delete
  -h, --help               show help
      --idle               delete idle running container
  -i, --image string       clean image before this time, format: <number>(d|w|m|y)

参数解释：
-c, --container <number>(d|w|m|y) 删除指定时间前创建的、当前已经退出的容器
-c 5d表示清除5天前创建的，现在是退出状态的容器

--idle 配合-c使用的，删除退出容器的同时，也会删除容器里只有一个进程，且进程是/bin/bash、/bin/sh或sleep的容器，这些容器通常是闲置的

-i, --image <number>(d|w|m|y) 尝试删除指定时间前下载的镜像
-i 4m表示尝试删除4个月以前下载的镜像，如果有容器在用，则跳过

-d, --dry-run 配合-c参数使用，仅显示符合筛选条件的容器而不删除

cleanc -d -c 5d --idle会打印5天前创建的，现在退出或空闲的容器，但不会删除
```

