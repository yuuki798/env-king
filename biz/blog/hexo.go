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
			"---\n\n", blog.Title, blog.UpdatedAt, tagsString, blog.Category)

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

// processFrontMatter 处理Front Matter，如果已存在则覆盖，否则添加
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
			// 找到Front Matter，进行替换
			// 保留Front Matter后的内容
			remainingContent := strings.Join(lines[endIndex+1:], "\n")
			// 如果剩余内容不为空，确保有换行
			if remainingContent != "" && !strings.HasPrefix(remainingContent, "\n") {
				remainingContent = "\n" + remainingContent
			}
			return newHeader + remainingContent
		}
	}

	// 没有找到Front Matter或格式不正确，直接添加
	return newHeader + content
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
