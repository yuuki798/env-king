package ek

import (
	"github.com/spf13/cobra"
	"yuuki798/env-king/biz/agent"
)

var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "智能 Agent（沙盒、cron、自我修改、部署）",
}

var agentChatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "与 Agent 对话",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAgentChat,
}

var agentCronListCmd = &cobra.Command{
	Use:   "cron-list",
	Short: "列出 cron 任务",
	RunE:  runAgentCronList,
}

var agentDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "打包并部署新 env-king",
	RunE:  runAgentDeploy,
}

func init() {
	AgentCmd.AddCommand(agentChatCmd)
	AgentCmd.AddCommand(agentCronListCmd)
	AgentCmd.AddCommand(agentDeployCmd)
}

func runAgentChat(cmd *cobra.Command, args []string) error {
	reply, err := agent.Chat(args[0])
	if err != nil {
		return err
	}
	cmd.Println(reply)
	return nil
}

func runAgentCronList(cmd *cobra.Command, args []string) error {
	jobs, err := agent.CronList()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		cmd.Printf("%s %s\n", j.Spec, j.Name)
	}
	return nil
}

func runAgentDeploy(cmd *cobra.Command, args []string) error {
	return agent.Deploy()
}
