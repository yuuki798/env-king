package ek

import (
	"github.com/spf13/cobra"
	"yuuki798/env-king/biz/skills"
)

var SkillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "MCP & Skills 管理",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出已安装 Skills",
	RunE:  runSkillsList,
}

var skillsMcpListCmd = &cobra.Command{
	Use:   "mcp-list",
	Short: "列出已安装 MCP",
	RunE:  runMcpList,
}

func init() {
	SkillsCmd.AddCommand(skillsListCmd)
	SkillsCmd.AddCommand(skillsMcpListCmd)
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	list := skills.ListSkills()
	for _, s := range list {
		cmd.Println(s)
	}
	return nil
}

func runMcpList(cmd *cobra.Command, args []string) error {
	list := skills.ListMCP()
	for _, m := range list {
		cmd.Println(m)
	}
	return nil
}
