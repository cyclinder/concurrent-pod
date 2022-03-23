# concurrent-pod
Concurrent creation of pods for multus test

# **QuickStart**
```shell
git clone https://github.com/cyclinder/concurrent-pod.git
cd concurrent-pod
go run main.go
```

**option flag**
- -f

the file path of pod yaml file(only pod)
- -n

the number of pods created
- -g

number of concurrent goroutines

**example**
```shell
go run main.go -f /root/test/pod.yaml -n 1 -g 1
```
