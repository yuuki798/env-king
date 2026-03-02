package ek

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"yuuki798/env-king/biz/pipeline"
)

var PipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "构建流水线（GitHub → Docker → Harbor）",
}

var pipelineTriggerCmd = &cobra.Command{
	Use:   "trigger [repo]",
	Short: "触发构建：从 GitHub 拉取，构建镜像并推送到 Harbor",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPipelineTrigger,
}

func init() {
	PipelineCmd.AddCommand(pipelineTriggerCmd)
	pipelineTriggerCmd.Flags().String("repo", "", "GitHub 仓库 owner/repo")
	pipelineTriggerCmd.Flags().String("branch", "main", "分支")
}

func runPipelineTrigger(cmd *cobra.Command, args []string) error {
	_ = viper.ReadInConfig()
	repo, _ := cmd.Flags().GetString("repo")
	branch, _ := cmd.Flags().GetString("branch")
	if repo == "" && len(args) > 0 {
		repo = args[0]
	}
	if repo == "" {
		cmd.Println("用法: env-king pipeline trigger --repo owner/repo [--branch main]")
		return nil
	}
	if branch == "" {
		branch = "main"
	}
	svc := pipeline.NewService()
	job, err := svc.Trigger(repo, branch)
	if err != nil {
		return err
	}
	cmd.Printf("已触发构建 job=%s repo=%s branch=%s\n", job.ID, job.Repo, job.Branch)
	return nil
}
