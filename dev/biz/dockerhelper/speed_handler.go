package dockerhelper

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
	"yuuki798/env-king/biz/webspider"
)

type SpeedHandler struct {
	WebSpider *webspider.Hub

	Url2Duration     []Url2Duration // 存储 URL 和对应的下载速度
	LockUrl2Duration sync.Mutex     // 用于保护 list 的并发访问
}

func NewSpeedHandler() *SpeedHandler {
	return &SpeedHandler{
		WebSpider:    webspider.NewHub(),
		Url2Duration: make([]Url2Duration, 0),
	}
}

func (this *SpeedHandler) FetchList() bool {
	// list  存到内存
	// 使用 WebSpider 获取镜像站列表
	ok := this.WebSpider.DockerMirrorListHandler.FetchList()
	if !ok {
		return false
	}
	this.LockUrl2Duration.Lock()
	for _, url := range this.WebSpider.DockerMirrorListHandler.List {
		// 初始化每个 URL 的速度为 0
		this.Url2Duration = append(this.Url2Duration, Url2Duration{
			Url:      url,
			Duration: 10000,
		})
	}
	this.LockUrl2Duration.Unlock()
	return true
}

func (this *SpeedHandler) SpeedTest() {
	// 并发拉取镜像
	ch := make(chan Url2Duration, len(this.Url2Duration))
	var wg sync.WaitGroup

	for _, url := range this.Url2Duration {
		wg.Add(1)
		go pullImage(url.Url, ch, &wg)
	}

	// 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(ch)
	}()

	// 收集结果
	for result := range ch {
		this.LockUrl2Duration.Lock()
		for i, url := range this.Url2Duration {
			if url.Url == result.Url {
				this.Url2Duration[i].Duration = result.Duration
				break
			}
		}
		this.LockUrl2Duration.Unlock()
	}
	sort.Slice(this.Url2Duration, func(i, j int) bool {
		return this.Url2Duration[i].Duration < this.Url2Duration[j].Duration
	})
}

func pullImage(url string, ch chan<- Url2Duration, wg *sync.WaitGroup) (time.Duration, bool) {
	defer wg.Done()
	myImg := generateImageName(url, "busybox:1.37.0")
	// 先删除缓存保证公平性
	cmd := exec.Command("docker", "rmi", "-f", myImg)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run() // 忽略错误，有可能镜像本来就不存在

	// 开始测速
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	cmd = exec.CommandContext(ctx, "docker", "pull", myImg)
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err != nil {
		return 0, false // 拉取失败
	}
	duration := time.Since(start)
	ch <- Url2Duration{
		Url:      url,
		Duration: duration.Milliseconds(),
	}
	return duration, true
}

func generateImageName(url string, name string) string {
	// 生成镜像名称，格式为 registry/namespace/image:tag
	// 例如: docker.io/library/ubuntu:latest
	// 但是url是https地址，需要转换为docker地址
	if url == "" {
		return name
	}
	// 去掉协议部分
	if strings.HasPrefix(url, "https://") {
		url = url[len("https://"):]
	}
	if strings.HasPrefix(url, "http://") {
		url = url[len("http://"):]
	}
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1] // 去掉末尾的斜杠
	}
	return url + "/" + name
}
