# docker-registry-cleaner
## QuickStart
### 0.环境依赖(Prerequisites)
已安装Golang(Golang Installed)
* https://golang.org/dl/
### 1.查看所有镜像(List All Images)
此操作不会实际进行删除动作(This operation will not actually do delete.)
```
# go run main.go -registry-url=http://127.0.0.1:5000 -list-all=true
2021/03/09 23:36:39 ubuntu:20.10
2021/03/09 23:36:39 ubuntu:16.04
2021/03/09 23:36:39 ubuntu:18.04
```

### 2.查看需要被删除的Tag(List Tags that should be deleted)
此操作不会实际进行删除动作(This operation will not actually do delete.)
```
//可自行更改需要保留的tag数(You can change tags-keep as need)
# go run main.go -registry-url=http://127.0.0.1:5000 -tags-keep=1
2021/03/09 23:37:03 will delete ubuntu:18.04, create time 2020-11-25T22:25:17.102901455Z
2021/03/09 23:37:03 will delete ubuntu:20.10, create time 2020-11-25T22:25:42.138514649Z
```

### 3.执行删除操作(Delete Tags)
请务必在删除前执行操作2确认删除列表(Please ensure delete list in Step 2)
```
# go run main.go -registry-url=http://127.0.0.1:5000 -tags-keep=1 --dry-run=false
2021/03/09 23:52:39 will delete ubuntu:18.04, create time 2020-11-25T22:25:17.102901455Z
2021/03/09 23:52:39 delete ubuntu:18.04 suceess
2021/03/09 23:52:39 will delete ubuntu:20.10, create time 2020-11-25T22:25:42.138514649Z
2021/03/09 23:52:39 delete ubuntu:20.10 suceess
```

