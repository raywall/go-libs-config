package builder

import (
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// ConfigBuilder - Construtor genérico de configurações
type ConfigBuilder struct {
	ssmClient *ssm.Client
}

// BuildOptions opções para construção da configuração
type BuildOptions struct {
	Prefixes           []string
	StripPrefix        bool
	JSONOutput         bool
	YAMLRules          bool // Nova opção para modo de regras YAML
	SortByDependencies bool
}
