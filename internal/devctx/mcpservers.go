package devctx

import (
	"regexp"

	"gopkg.in/yaml.v3"
)

var frontmatterExtractRe = regexp.MustCompile(`(?s)\A---\n(.*?)\n---`)

type MCPServer struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
}

type contextFrontmatter struct {
	MCPServers map[string]MCPServer `yaml:"mcp_servers"`
}

func ParseMCPServers(content []byte) map[string]MCPServer {
	m := frontmatterExtractRe.FindSubmatch(content)
	if m == nil {
		return nil
	}
	var fm contextFrontmatter
	if err := yaml.Unmarshal(m[1], &fm); err != nil {
		return nil
	}
	return fm.MCPServers
}
