# readme

model-downloader，模型下载器，用于下载和记录LLM模型的工具，该工具仅下载魔塔社区的模型

该工具分为两个部分：

- 服务端(model-downloader-server,mds)：一个后台任务，监听一个TCP端口，以接收、执行下载任务、向客户端展示下载任务
- 客户端(model-downloader-client,mdc)：一个命令行工具，可以向服务端提交、查看下载任务进度

特性：

- 内嵌sqlite3数据库，即使服务端中断，重启后也能自动开始中断的下载任务
- 会监视下载位置所属的文件系统剩余空间，一旦空闲空间小于10G，自动中断下载任务，不会写满磁盘
- 允许手动中断任务


## 使用示例

```bash
# 启动服务端，并将默认下载路径设为当前文件夹
mds -s .

# 让服务端开始下载 Qwen/Qwen3.5-4B 模型，下载位置是/public4/cache/Qwen3.5-4B
mdc pull Qwen/Qwen3.5-4B
> commit task success
> ID Model           Size    Status      DownloadProgress User SavePath
> 1  Qwen/Qwen3.5-4B 8.70GiB downloading 0.00%            root /public4/cache/Qwen3.5-4B

# 查看已有的下载任务
mdc ls
> list task success
> ID Model           Size    Status      DownloadProgress User SavePath
> 1  Qwen/Qwen3.5-4B 8.70GiB downloading 50.19%           root /public4/cache/Qwen3.5-4B

# 中断下载任务
mdc stop 1
> stop task success
> ID Model           Size    Status  DownloadProgress User SavePath
> 1  Qwen/Qwen3.5-4B 8.70GiB stopped 50.19%           root /public4/cache/Qwen3.5-4B

# 开始下载任务
mdc start 1
> start task success
> ID Model           Size    Status      DownloadProgress User SavePath
> 1  Qwen/Qwen3.5-4B 8.70GiB downloading 52.92%           root /public4/cache/Qwen3.5-4B

# 取消下载任务，注意：取消任务后，已经下载的部分不会删除，需要手动删除
mdc cancel 1
> cancel task success
```

## 编译方法

```
# 系统没有upx命令也没关系，主要是为了减少二进制文件体积，不影响实际使用
bash build.sh
```

## 注意

目前mdc里硬编码了服务端的地址，如需修改，编辑bin/client/main.go内的ServerAddr变量的值，然后重新编译即可


## todo

支持webhook，下载完成时自动发送消息提醒
