package blog

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"yuuki798/env-king/biz/blog/storage"
)

const VersionLen = 10

// Hub global singleton
type Hub struct {
	userManager  map[string]*User
	blogManagers map[string]*Manager
	store        *storage.Hub
}

type User struct {
	UserName  string
	Platforms []string
	// It fetches from config content, and will be passed to platform adapters
	PlatformConfigs []byte
	LocalPath       string
	HexoPath        string
}

func NewHub() *Hub {
	return &Hub{
		userManager:  map[string]*User{},
		blogManagers: map[string]*Manager{},
		store:        storage.NewHub(),
	}
}
func (this *Hub) RecoverFromStorage() error {
	// 获取或创建 blog bucket
	bucket, err := this.store.CreateOrOpenBucket("blog")
	if err != nil {
		return err
	}

	userListString, ok := bucket.Get("userList")
	if !ok {
		return nil // 没有用户数据，正常启动
	}
	userList := strings.Split(userListString, ",")
	for _, userName := range userList {
		if userName == "" {
			continue
		}
		userBytes, ok := bucket.Get("user:" + userName)
		if !ok {
			log.Printf("User data not found for: %s", userName)
			continue
		}
		var user User
		// 反序列化成user
		err := gob.NewDecoder(strings.NewReader(userBytes)).Decode(&user)
		if err != nil {
			log.Printf("gob decode user error for %s: %v", userName, err)
			continue
		}
		this.userManager[userName] = &user
	}
	return nil
}

func NewUser(userName string, platforms []string, platformConfigs []byte, localPath string, HexoPath string) *User {
	return &User{
		UserName:        userName,
		Platforms:       platforms,
		PlatformConfigs: platformConfigs,
		LocalPath:       localPath,
		HexoPath:        HexoPath,
	}
}

// AddUser 添加用户并持久化
func (this *Hub) AddUser(user *User) error {
	this.userManager[user.UserName] = user
	return this.saveUserToStorage(user)
}

// saveUserToStorage 保存用户到存储
func (this *Hub) saveUserToStorage(user *User) error {
	bucket, err := this.store.CreateOrOpenBucket("blog")
	if err != nil {
		return err
	}

	// 序列化用户数据
	var userBytes strings.Builder
	err = gob.NewEncoder(&userBytes).Encode(user)
	if err != nil {
		return err
	}

	// 保存用户数据
	err = bucket.Put("user:"+user.UserName, userBytes.String())
	if err != nil {
		return err
	}

	// 更新用户列表
	userListString, _ := bucket.Get("userList")
	userList := strings.Split(userListString, ",")

	// 检查用户是否已存在
	exists := false
	for _, u := range userList {
		if u == user.UserName {
			exists = true
			break
		}
	}

	if !exists {
		if userListString == "" {
			userListString = user.UserName
		} else {
			userListString += "," + user.UserName
		}
		err = bucket.Put("userList", userListString)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetUser 获取用户
func (this *Hub) GetUser(userName string) (*User, bool) {
	user, ok := this.userManager[userName]
	return user, ok
}

// AddBlogManager 添加博客管理器
func (this *Hub) AddBlogManager(userName string, manager *Manager) {
	this.blogManagers[userName] = manager
}

// GetBlogManagers 获取博客管理器映射
func (this *Hub) GetBlogManagers() map[string]*Manager {
	return this.blogManagers
}

// GetUserManager 获取用户管理器映射
func (this *Hub) GetUserManager() map[string]*User {
	return this.userManager
}

// GetStore 获取存储
func (this *Hub) GetStore() *storage.Hub {
	return this.store
}

// Manager manages blogs for a single user
type Manager struct {
	// We use a tree structure to store blogs
	blogTreeRoot *DirectoryNode
	// BlogList is a flat list of all blogs, used for listing,ordered by UpdatedAt desc
	BlogList       []*Blog
	ChosenBlogList []*Blog
	// LocalPath means the root directory where blogs are stored locally
	LocalPath        string
	HexoPath         string
	platforms        []string
	platformAdapters map[string]PlatformAdapter
	userName         string
	// 存储引用，用于持久化
	store *storage.Hub
}

func NewManager(localPath string, userName string, platforms []string, store *storage.Hub, HexoPath string) *Manager {
	return &Manager{
		blogTreeRoot:     &DirectoryNode{category: "default", children: map[string]*DirectoryNode{}},
		LocalPath:        localPath,
		platforms:        platforms,
		platformAdapters: map[string]PlatformAdapter{},
		userName:         userName,
		store:            store,
		HexoPath:         HexoPath,
	}
}
func (this *Manager) AddPlatformAdapter(name string) error {
	if _, exists := this.platformAdapters[name]; exists {
		return nil // already exists
	}
	var adapter PlatformAdapter
	switch strings.ToLower(name) {
	case "hexo":
		adapter = NewHexo(this, this.HexoPath)
	default:
		return errors.New("unsupported platform: " + name)
	}
	this.platformAdapters[name] = adapter
	log.Println("Added platform adapter:", name)
	return nil
}

// AddBlog 添加新博客并持久化
func (this *Manager) AddBlog(blog *Blog) error {
	// 添加到树结构
	err := this.addBlogToTree(blog)
	if err != nil {
		return err
	}

	// 添加到列表
	this.BlogList = append(this.BlogList, blog)

	// 持久化到存储
	return this.saveBlogToStorage(blog)
}

// addBlogToTree 将博客添加到树结构中
func (this *Manager) addBlogToTree(blog *Blog) error {
	parts := strings.Split(blog.Path, "/")
	curNode := this.blogTreeRoot

	// 创建目录路径
	for i, part := range parts {
		if i == len(parts)-1 {
			break // 最后一部分是文件名
		}

		if _, exists := curNode.children[part]; !exists {
			curNode.children[part] = &DirectoryNode{
				category: part,
				parent:   curNode,
				children: map[string]*DirectoryNode{},
				blogs:    []*Blog{},
			}
		}
		curNode = curNode.children[part]
	}

	// 添加博客到当前节点
	curNode.blogs = append(curNode.blogs, blog)
	return nil
}

// saveBlogToStorage 保存博客到存储
func (this *Manager) saveBlogToStorage(blog *Blog) error {
	bucket, err := this.store.CreateOrOpenBucket("blog")
	if err != nil {
		return err
	}

	// 序列化博客数据
	var blogBytes strings.Builder
	err = gob.NewEncoder(&blogBytes).Encode(blog)
	if err != nil {
		return err
	}

	// 保存博客数据
	key := "blog:" + this.userName + ":" + blog.Path
	return bucket.Put(key, blogBytes.String())
}

// AddNewVersion used only when updating.
func (this *Manager) AddNewVersion(newVersion *Blog, path string) error {
	// add the last version to history and replace the current version as newVersion
	cur, err := this.findBlogByPath(path)
	if err != nil {
		log.Println(this.BlogList[0].Title)
		log.Println(this.blogTreeRoot.blogs)
		log.Println("blog not found:", path)
		return err
	}

	// 创建当前版本的副本作为历史版本，并清空其OldVersions避免循环引用
	oldVersion := *cur
	oldVersion.OldVersions = nil // 清空OldVersions避免循环引用

	// slice is better to append than prepend, because of memory copy,
	// we keep the latest at the end of the slice
	cur.OldVersions = append(cur.OldVersions, &oldVersion)
	// keep only the last 10 versions
	if len(cur.OldVersions) > VersionLen {
		cur.OldVersions = cur.OldVersions[len(cur.OldVersions)-VersionLen:]
	}
	// replace the current version
	cur = newVersion

	// 持久化更新
	return this.saveBlogToStorage(cur)
}

func (this *Manager) findBlogByPath(path string) (*Blog, error) {
	// path example: "category1/category2/blog-title.md"
	parts := strings.Split(path, "/")
	curNode := this.blogTreeRoot
	for i, x := range parts {
		if i == len(parts)-1 {
			break
		}
		// children array is  map
		node, ok := curNode.children[x]
		if !ok {
			log.Println("directory not found:", x)
			return nil, errors.New("directory not found: " + x)
		}
		curNode = node
	}
	// blog array is slice
	for _, blog := range curNode.blogs {
		if blog.Path == path {
			return blog, nil
		}
	}
	return nil, errors.New("blog not found: " + path)
}

// FindBlogByPath 公共方法：根据路径查找博客
func (this *Manager) FindBlogByPath(path string) (*Blog, error) {
	return this.findBlogByPath(path)
}

// ReadFromLocal 公共方法：从本地读取
func (this *Manager) ReadFromLocal() error {
	return this.readFromLocal()
}

type DirectoryNode struct {
	// Each directory corresponds to a category, the root directory is "default". If directory is not the
	// 2nd level, it can connect with "/" to form a complete category path.
	category string
	parent   *DirectoryNode
	// the key is the directory title, also the category title
	children map[string]*DirectoryNode
	blogs    []*Blog
}
type Blog struct {
	// creation time
	CreatedAt time.Time
	// last updated time
	UpdatedAt time.Time
	Category  string
	Tags      []string
	Title     string
	Content   []byte
	Path      string
	// Old versions store the previous versions of the Blog except the current one.
	// It always keeps the last 10 versions and ordered by UpdatedAt desc.
	OldVersions []*Blog
}

// readFromLocal 从本地路径读取所有博客
func (this *Manager) readFromLocal() error {
	return this.scanDirectory(this.LocalPath, "")
}

// scanDirectory 递归扫描目录
func (this *Manager) scanDirectory(dirPath, relativePath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// 递归扫描子目录
			subPath := filepath.Join(dirPath, entry.Name())
			subRelativePath := filepath.Join(relativePath, entry.Name())
			err = this.scanDirectory(subPath, subRelativePath)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// 处理 Markdown 文件
			filePath := filepath.Join(dirPath, entry.Name())
			blogPath := filepath.Join(relativePath, entry.Name())
			err = this.processMarkdownFile(filePath, blogPath)
			if err != nil {
				log.Printf("Error processing file %s: %v", filePath, err)
				continue
			}
		}
	}

	return nil
}

// processMarkdownFile 处理单个 Markdown 文件
func (this *Manager) processMarkdownFile(filePath, blogPath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 解析文件内容，提取元数据
	blog := this.parseMarkdownFile(content, blogPath)

	// 添加到管理器
	return this.AddBlog(blog)
}

// parseMarkdownFile 解析 Markdown 文件，提取元数据
func (this *Manager) parseMarkdownFile(content []byte, path string) *Blog {
	now := time.Now()
	blog := &Blog{
		Path:      path,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{},
	}

	// 提取文件名作为标题
	fileName := filepath.Base(path)
	blog.Title = strings.TrimSuffix(fileName, ".md")

	// 从路径提取分类
	dir := filepath.Dir(path)
	if dir != "." {
		blog.Category = strings.ReplaceAll(dir, "/", " > ")
	} else {
		blog.Category = "default"
	}

	// 解析 Front Matter（如果存在）
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		// 查找结束标记
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				// 解析 Front Matter
				this.parseFrontMatter(lines[1:i], blog)
				break
			}
		}
	}

	return blog
}

// parseFrontMatter 解析 Front Matter
func (this *Manager) parseFrontMatter(lines []string, blog *Blog) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "title":
			// 保持title就是文件名，不覆盖
			// blog.Title = value
		case "date":
			if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
				blog.UpdatedAt = t
			}
		case "tags":
			// 解析标签，支持多种格式
			if strings.Contains(value, ",") {
				tags := strings.Split(value, ",")
				for _, tag := range tags {
					blog.Tags = append(blog.Tags, strings.TrimSpace(tag))
				}
			} else {
				blog.Tags = append(blog.Tags, value)
			}
		case "categories":
			blog.Category = value
		}
	}
}

// GobEncode 自定义序列化方法，确保OldVersions被正确序列化
func (b *Blog) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// 手动序列化每个字段
	if err := enc.Encode(b.CreatedAt); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.UpdatedAt); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.Category); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.Tags); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.Title); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.Content); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.Path); err != nil {
		return nil, err
	}
	if err := enc.Encode(b.OldVersions); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GobDecode 自定义反序列化方法
func (b *Blog) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	// 手动反序列化每个字段
	if err := dec.Decode(&b.CreatedAt); err != nil {
		return err
	}
	if err := dec.Decode(&b.UpdatedAt); err != nil {
		return err
	}
	if err := dec.Decode(&b.Category); err != nil {
		return err
	}
	if err := dec.Decode(&b.Tags); err != nil {
		return err
	}
	if err := dec.Decode(&b.Title); err != nil {
		return err
	}
	if err := dec.Decode(&b.Content); err != nil {
		return err
	}
	if err := dec.Decode(&b.Path); err != nil {
		return err
	}
	if err := dec.Decode(&b.OldVersions); err != nil {
		return err
	}

	return nil
}

type IBlogManager interface {
	// readFromLocal init all blogs from local path to a Blog tree
	readFromLocal() error

	Commit() error
	Push() error
	List() error
	History(page int, path string) error
}
