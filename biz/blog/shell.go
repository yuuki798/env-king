package blog

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/common-nighthawk/go-figure"
)

// Shell 提供交互式的blog管理shell
type Shell struct {
	hub      *Hub
	manager  *Manager
	username string
	running  bool
}

// NewShell 创建新的shell实例
func NewShell(hub *Hub, username string) *Shell {
	return &Shell{
		hub:      hub,
		username: username,
		running:  true,
	}
}

// Start 启动交互式shell
func (s *Shell) Start() error {
	// 初始化用户管理器
	user, exists := s.hub.GetUser(s.username)
	if !exists {
		return fmt.Errorf("user %s not found", s.username)
	}
	if user.LocalPath == "" {
		return fmt.Errorf("user %s has no local path configured", s.username)
	}

	// 创建博客管理器
	s.manager = NewManager(user.LocalPath, s.username, user.Platforms, s.hub.GetStore(), user.HexoPath)
	for i := range user.Platforms {
		err := s.manager.AddPlatformAdapter(user.Platforms[i])
		if err != nil {
			return fmt.Errorf("failed to add platform adapter %s: %v", user.Platforms[i], err)
		}
	}

	// 从存储恢复数据
	err := s.hub.RecoverFromStorage()
	if err != nil {
		fmt.Printf("Warning: Failed to recover from storage: %v\n", err)
	}

	// 显示欢迎信息
	s.showWelcome()

	// 启动交互循环
	return s.runInteractiveLoop()
}

// showWelcome 显示欢迎信息
func (s *Shell) showWelcome() {
	fmt.Println("=" + strings.Repeat("=", 50))

	myFigure := figure.NewFigure("ek-blog", "", true)
	myFigure.Print()

	fmt.Printf("  Welcome to Blog Shell - User: %s\n", s.username)
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Available commands:")
	fmt.Println("  article list     - List all articles")
	fmt.Println("  article commit   - Commit changes to local storage")
	fmt.Println("  article push     - Push articles to Platforms")
	fmt.Println("  help             - Show this help message")
	fmt.Println("  exit             - Exit the shell")
	fmt.Println("user info - Show current user info")
	fmt.Println("=" + strings.Repeat("=", 50))
}

// runInteractiveLoop 运行交互循环
func (s *Shell) runInteractiveLoop() error {
	scanner := bufio.NewScanner(os.Stdin)

	for s.running {
		fmt.Print("blog> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		err := s.processCommand(line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return scanner.Err()
}

// processCommand 处理用户命令
func (s *Shell) processCommand(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "article":
		return s.handleArticleCommand(args)
	case "help":
		s.showWelcome()
		return nil
	case "user":
		if len(args) > 0 && args[0] == "info" {
			user, exists := s.hub.GetUser(s.username)
			if !exists {
				fmt.Printf("User %s not found\n", s.username)
				return nil
			}
			fmt.Printf("Username: %s\n", user.UserName)
			fmt.Printf("Local Path: %s\n", user.LocalPath)
			fmt.Printf("Platforms: %s\n", strings.Join(user.Platforms, ", "))
			return nil
		}
		fmt.Println("Usage: user info - Show current user info")
		return nil
	case "exit":
		s.running = false
		fmt.Println("Goodbye!")
		return nil
	default:
		fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		return nil
	}
}

// handleArticleCommand 处理article子命令
func (s *Shell) handleArticleCommand(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: article <command>")
		fmt.Println("Commands: list, commit, push")
		return nil
	}

	subCommand := args[0]

	switch subCommand {
	case "list":
		return s.listArticles()
	case "commit":
		return s.commitArticles()
	case "push":
		return s.pushArticles()
	default:
		fmt.Printf("Unknown article command: %s\n", subCommand)
		fmt.Println("Available commands: list, commit, push")
		return nil
	}
}

// listArticles 列出所有文章
func (s *Shell) listArticles() error {
	if s.manager == nil {
		return fmt.Errorf("manager not initialized")
	}

	// 清空现有博客列表，避免重复添加
	s.manager.BlogList = []*Blog{}

	// 从本地读取文章
	err := s.manager.ReadFromLocal()
	if err != nil {
		return fmt.Errorf("failed to read from local: %v", err)
	}

	blogs := s.manager.BlogList
	if len(blogs) == 0 {
		fmt.Println("No articles found.")
		return nil
	}

	fmt.Printf("\nFound %d articles:\n", len(blogs))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-50s %-20s %-10s\n", "Title", "Category", "Updated")
	fmt.Println(strings.Repeat("-", 80))

	for _, blog := range blogs {
		updatedStr := blog.UpdatedAt.Format("2006-01-02 15:04")
		fmt.Printf("%-50s %-20s %-10s\n",
			truncateString(blog.Title, 50),
			truncateString(blog.Category, 20),
			updatedStr)
	}
	fmt.Println(strings.Repeat("-", 80))

	return nil
}

// commitArticles 提交文章到本地存储
func (s *Shell) commitArticles() error {
	if s.manager == nil {
		return fmt.Errorf("manager not initialized")
	}

	fmt.Println("Committing articles to local storage...")

	// 清空现有博客列表，避免重复添加
	s.manager.BlogList = []*Blog{}

	// 从本地读取文章
	err := s.manager.ReadFromLocal()
	if err != nil {
		return fmt.Errorf("failed to read from local: %v", err)
	}
	// 保存到存储
	for i := range s.manager.platformAdapters {
		err := s.manager.platformAdapters[i].PreProcess()
		if err != nil {
			return fmt.Errorf("preprocess failed for platform %s: %v", s.manager.platformAdapters[i].Name(), err)
		}
		err = s.manager.platformAdapters[i].SaveToDraft()
		if err != nil {
			return fmt.Errorf("failed to save to draft for platform %s: %v", s.manager.platformAdapters[i].Name(), err)
		}
	}
	blogCount := len(s.manager.BlogList)
	fmt.Printf("Successfully committed %d articles to local storage.\n", blogCount)

	return nil
}

// pushArticles 推送文章到平台
func (s *Shell) pushArticles() error {
	if s.manager == nil {
		return fmt.Errorf("manager not initialized")
	}

	fmt.Println("Pushing articles to Platforms...")

	// 获取用户配置的平台
	user, exists := s.hub.GetUser(s.username)
	if !exists {
		return fmt.Errorf("user %s not found", s.username)
	}

	if len(user.Platforms) == 0 {
		fmt.Println("No Platforms configured for this user.")
		return nil
	}

	// 清空现有博客列表，避免重复添加
	s.manager.BlogList = []*Blog{}

	// 从本地读取文章
	err := s.manager.ReadFromLocal()
	if err != nil {
		return fmt.Errorf("failed to read from local: %v", err)
	}

	for i := range s.manager.platformAdapters {
		err = s.manager.platformAdapters[i].Publish()
		if err != nil {
			return fmt.Errorf("publish failed for platform %s: %v", s.manager.platformAdapters[i].Name(), err)
		}
	}

	fmt.Println("Articles pushed successfully!")

	return nil
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
