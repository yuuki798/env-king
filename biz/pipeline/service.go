package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Job struct {
	ID         string    `json:"id"`
	Repo       string    `json:"repo"`
	Branch     string    `json:"branch"`
	Status     string    `json:"status"`
	ImageTag   string    `json:"imageTag,omitempty"`
	HarborRepo string    `json:"harborRepo,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Logs       string    `json:"logs,omitempty"`
}

type Service struct {
	mu      sync.RWMutex
	jobs    []*Job
	workDir string
}

func NewService() *Service {
	workDir := viper.GetString("pipeline.work_dir")
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "env-king-pipeline")
	}
	_ = os.MkdirAll(workDir, 0755)
	return &Service{jobs: make([]*Job, 0), workDir: workDir}
}

func (s *Service) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, len(s.jobs))
	copy(out, s.jobs)
	return out
}

func (s *Service) Trigger(repo, branch string) (*Job, error) {
	repo = normalizeRepo(repo)
	job := &Job{
		ID:        uuid.New().String(),
		Repo:      repo,
		Branch:    branch,
		Status:    "pending",
		StartedAt: time.Now(),
	}
	s.mu.Lock()
	s.jobs = append([]*Job{job}, s.jobs...)
	if len(s.jobs) > 50 {
		s.jobs = s.jobs[:50]
	}
	s.mu.Unlock()

	go s.runJob(job)
	return job, nil
}

func normalizeRepo(repo string) string {
	repo = strings.TrimSpace(repo)
	if strings.HasPrefix(repo, "https://github.com/") {
		repo = strings.TrimPrefix(repo, "https://github.com/")
		repo = strings.TrimSuffix(repo, ".git")
	}
	if strings.HasPrefix(repo, "git@github.com:") {
		repo = strings.TrimPrefix(repo, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
	}
	return repo
}

func (s *Service) runJob(job *Job) {
	job.Status = "building"
	cloneDir := filepath.Join(s.workDir, job.ID)
	_ = os.MkdirAll(cloneDir, 0755)
	defer func() {
		job.FinishedAt = time.Now()
		os.RemoveAll(cloneDir)
	}()

	// 使用 GITHUB_TOKEN 或 SSH 拉取私有仓库
	token := viper.GetString("github.token")
	cloneURL := fmt.Sprintf("https://github.com/%s.git", job.Repo)
	if token != "" {
		cloneURL = fmt.Sprintf("https://%s@github.com/%s.git", token, job.Repo)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// git clone
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "-b", job.Branch, cloneURL, cloneDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		job.Status = "failed"
		job.Logs = string(out) + "\n" + err.Error()
		return
	}

	// 检测 Dockerfile
	dockerfile := filepath.Join(cloneDir, "Dockerfile")
	if _, err := os.Stat(dockerfile); err != nil {
		job.Status = "failed"
		job.Logs = "no Dockerfile found"
		return
	}

	// docker build
	harborHost := viper.GetString("harbor.host")
	harborProject := viper.GetString("harbor.project")
	if harborProject == "" {
		harborProject = "library"
	}
	parts := strings.Split(job.Repo, "/")
	imageName := parts[len(parts)-1]
	tag := fmt.Sprintf("%s-%s-%d", job.Branch, job.ID[:8], job.StartedAt.Unix())
	fullImage := fmt.Sprintf("%s/%s/%s:%s", harborHost, harborProject, imageName, tag)
	if harborHost == "" {
		fullImage = fmt.Sprintf("%s:%s", imageName, tag)
	}
	job.HarborRepo = fullImage

	cmd = exec.CommandContext(ctx, "docker", "build", "-t", fullImage, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		job.Status = "failed"
		job.Logs = string(out)
		return
	}

	job.Status = "pushing"
	if harborHost != "" {
		cmd = exec.CommandContext(ctx, "docker", "push", fullImage)
		if out, err := cmd.CombinedOutput(); err != nil {
			job.Status = "failed"
			job.Logs = string(out)
			return
		}
	}

	job.Status = "success"
	job.ImageTag = tag
}
