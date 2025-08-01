package webspider

import (
	colly "github.com/gocolly/colly/v2"
	"log"
	"strings"
)

type DockerMirrorListHandler struct {
	url          string
	List         []string
	collyHandler *colly.Collector
}

func NewDockerMirrorListHandler(url string) *DockerMirrorListHandler {
	if url == "" {
		url = "https://www.coderjia.cn/archives/dba3f94c-a021-468a-8ac6-e840f85867ea"
	}
	return &DockerMirrorListHandler{
		url:  url,
		List: make([]string, 0),
		collyHandler: colly.NewCollector(
			colly.AllowedDomains("www.coderjia.cn", "coderjia.cn"), // 防止跳转到其他域
		),
	}
}

func (this *DockerMirrorListHandler) FetchList() bool {
	var results []MirrorStatus

	// 提取表格内容
	this.collyHandler.OnHTML("table tbody tr", func(ele *colly.HTMLElement) {
		cols := ele.DOM.Find("td")
		if cols.Length() >= 2 {
			address := strings.TrimSpace(cols.Eq(0).Text())
			status := strings.TrimSpace(cols.Eq(1).Text())

			results = append(results, MirrorStatus{
				Address: address,
				Status:  status,
			})
		}
	})

	// 错误处理
	this.collyHandler.OnError(func(r *colly.Response, err error) {
		log.Println("访问出错:", err)
	})

	// 启动访问
	err := this.collyHandler.Visit(this.url)
	if err != nil {
		log.Println("访问失败:", err)
		return false
	}
	for _, result := range results {
		if result.Status == "正常" {
			this.List = append(this.List, result.Address)
		}
	}
	return true
}
