package blog

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"yuuki798/env-king/biz/blog/myutil"
)

const PageSize = 10

type Hexo struct {
	dirExists bool

	blogManager *Manager
	DraftFolder string

	curListPage int64

	os string
}

func NewHexo(manager *Manager, draftFolder string) *Hexo {
	// check if the draft folder exists
	dirExists := true
	_, err := os.Stat(draftFolder)
	if os.IsNotExist(err) {
		dirExists = false
	}
	return &Hexo{
		blogManager: manager,
		DraftFolder: draftFolder,
		dirExists:   dirExists,
		os:          strings.ToLower(runtime.GOOS),
	}
}

func (this *Hexo) Auth() error {
	// because hexo is a static site generator, no need to auth.
	return nil
}

func (this *Hexo) SaveToDraft() error {
	// to avoid modifying the original file, we save the file to draft folder
	if !this.dirExists {
		_ = myutil.Mkdir(this.DraftFolder)
	}

	for _, blog := range this.blogManager.BlogList {
		targetFilePath := this.DraftFolder + "/" + blog.Title + ".md"
		file, err := os.Create(targetFilePath)
		if err != nil {
			log.Printf("create file error for %s: %v", targetFilePath, err)
			continue
		}

		srcFilePath := this.blogManager.LocalPath + "/" + blog.Path
		srcFile, err := os.Open(srcFilePath)
		if err != nil {
			log.Printf("open source file error for %s: %v", srcFilePath, err)
			file.Close()
			continue
		}

		// before copy ,we need to justify the blogTree version
		err = this.blogManager.AddNewVersion(blog, blog.Path)
		if err != nil {
			log.Printf("add new version error for %s: %v", blog.Title, err)
			// continue even if versioning fails for now
		}

		// finally copy the file
		_, err = io.Copy(file, srcFile)
		if err != nil {
			log.Printf("copy file error for %s: %v", blog.Title, err)
		}

		file.Close()
		srcFile.Close()
	}

	log.Printf("Saved %d blogs to draft folder.", len(this.blogManager.BlogList))
	return nil
}

// PreProcess will do before commit/publish.
func (this *Hexo) PreProcess() error {
	for _, blog := range this.blogManager.BlogList {
		// tags split with ","
		tagsString := strings.Join(blog.Tags, ", ")
		header := fmt.Sprintf("---\n"+
			"title: %v\n"+
			"date: %v\n"+
			"tags: %v\n"+
			"categories: %v\n"+
			"---\n\n", blog.Title, blog.CreatedAt.Format("2006-01-02 15:04:05"), tagsString, blog.Category)

		// read the original content
		srcFilePath := this.blogManager.LocalPath + "/" + blog.Path
		content, err := os.ReadFile(srcFilePath)
		if err != nil {
			log.Printf("read file error for %s: %v", srcFilePath, err)
			continue
		}

		// 检测并处理已存在的Front Matter
		newContent := this.processFrontMatter(string(content), header)

		// write the new content to the original file
		err = os.WriteFile(srcFilePath, []byte(newContent), 0644)
		if err != nil {
			log.Printf("write file error for %s: %v", srcFilePath, err)
			continue
		}
	}
	log.Printf("Pre-processed %d blogs.", len(this.blogManager.BlogList))
	return nil
}

// processFrontMatter 处理Front Matter，如果已存在则智能合并，否则添加
func (this *Hexo) processFrontMatter(content, newHeader string) string {
	lines := strings.Split(content, "\n")

	// 检查是否已有Front Matter
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		// 查找Front Matter结束标记
		endIndex := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIndex = i
				break
			}
		}

		if endIndex != -1 {
			// 找到Front Matter，进行智能合并
			existingFrontMatter := lines[1:endIndex]
			mergedHeader := this.mergeFrontMatter(existingFrontMatter, newHeader)

			// 保留Front Matter后的内容
			remainingContent := strings.Join(lines[endIndex+1:], "\n")
			// 如果剩余内容不为空，确保有换行
			if remainingContent != "" && !strings.HasPrefix(remainingContent, "\n") {
				remainingContent = "\n" + remainingContent
			}
			return mergedHeader + remainingContent
		}
	}

	// 没有找到Front Matter或格式不正确，直接添加
	return newHeader + content
}

// mergeFrontMatter 智能合并Front Matter，保留已存在的date字段
func (this *Hexo) mergeFrontMatter(existing []string, newHeader string) string {
	// 解析已存在的Front Matter
	existingFields := make(map[string]string)
	for _, line := range existing {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			existingFields[key] = value
		}
	}

	// 解析新的Front Matter
	newLines := strings.Split(newHeader, "\n")
	newFields := make(map[string]string)
	for _, line := range newLines {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			newFields[key] = value
		}
	}

	// 合并字段，优先保留已存在的date字段
	mergedFields := make(map[string]string)

	// 先添加新字段
	for key, value := range newFields {
		mergedFields[key] = value
	}

	// 如果已存在date字段，则保留原有的date
	if existingDate, exists := existingFields["date"]; exists {
		mergedFields["date"] = existingDate
	}

	// 构建合并后的Front Matter
	var result strings.Builder
	result.WriteString("---\n")

	// 按固定顺序输出字段
	fieldOrder := []string{"title", "date", "tags", "categories"}
	for _, field := range fieldOrder {
		if value, exists := mergedFields[field]; exists {
			result.WriteString(fmt.Sprintf("%s: %s\n", field, value))
		}
	}

	result.WriteString("---\n")
	return result.String()
}

func (this *Hexo) Publish() error {
	if this.os == "windows" {
		err := os.Chdir(this.DraftFolder)
		if err != nil {
			log.Println("change dir error:", err)
			return err
		}
		// use hexo to publish the blog
		cmdGenerate := exec.Command("hexo", "g")
		cmdGenerate.Stdout = os.Stdout
		cmdGenerate.Stderr = os.Stderr
		err = cmdGenerate.Run()
		if err != nil {
			log.Println("hexo generate error:", err)
			return err
		}

		cmdDeploy := exec.Command("hexo", "d")
		cmdDeploy.Stdout = os.Stdout
		cmdDeploy.Stderr = os.Stderr
		err = cmdDeploy.Run()
		if err != nil {
			log.Println("hexo deploy error:", err)
			return err
		}
		return nil
	}
	return errors.New("not implemented for non-windows OS")
}

func (this *Hexo) GetList() error {
	if this.curListPage == 0 {
		this.curListPage = 1
	}
	list := this.blogManager.BlogList
	start := (this.curListPage - 1) * PageSize
	end := this.curListPage * PageSize
	if start >= int64(len(list)) {
		return errors.New("page out of range")
	}
	if end > int64(len(list)) {
		end = int64(len(list))
	}
	for i := start; i < end; i++ {
		singleBlog := list[i]
		fmt.Printf("Title: %v ", singleBlog.Title)
		fmt.Printf("Path: %v ", singleBlog.Path)
		fmt.Printf("Category: %v ", singleBlog.Category)
		fmt.Printf("Tags: %v ", strings.Join(singleBlog.Tags, ", "))
		fmt.Printf("UpdatedAt: %v\n", singleBlog.UpdatedAt)
		fmt.Println("-------------------------")
	}
	return nil
}

func (this *Hexo) Name() string {
	return "hexo"
}

var _ PlatformAdapter = (*Hexo)(nil)
