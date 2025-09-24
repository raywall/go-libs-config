package builder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"gopkg.in/yaml.v3"
)

// New cria uma nova instância do ConfigBuilder
func New(ssmClient *ssm.Client) *ConfigBuilder {
	return &ConfigBuilder{
		ssmClient: ssmClient,
	}
}

// BuildConfigFromPrefixes constrói a configuração a partir dos prefixos
func (b *ConfigBuilder) BuildConfigFromPrefixes(ctx context.Context, opts BuildOptions) ([]byte, error) {
	if opts.YAMLRules {
		// Modo YAML para regras
		configMap := make(map[string]interface{})

		for _, prefix := range opts.Prefixes {
			params, err := b.getParametersByPath(ctx, prefix)
			if err != nil {
				return nil, fmt.Errorf("erro ao buscar parâmetros do prefixo %s: %w", prefix, err)
			}

			prefixConfig, err := b.buildYAMLStructure(params, prefix, opts.StripPrefix)
			if err != nil {
				return nil, err
			}
			b.mergeMaps(configMap, prefixConfig)
		}

		// Para YAML, não aplicamos ordenação por dependências (específica para schemas JSON)
		return yaml.Marshal(configMap)
	}

	// Modo JSON padrão
	configMap := make(map[string]interface{})

	for _, prefix := range opts.Prefixes {
		params, err := b.getParametersByPath(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar parâmetros do prefixo %s: %w", prefix, err)
		}

		prefixConfig := b.buildStructure(params, prefix, opts.StripPrefix, opts.SortByDependencies)
		b.mergeMaps(configMap, prefixConfig)
	}

	if opts.SortByDependencies {
		err := sortTypesByDependency(&configMap)
		if err != nil {
			return nil, fmt.Errorf("erro ao ordenar tipos por dependência: %w", err)
		}
	}

	if opts.JSONOutput {
		return json.MarshalIndent(configMap, "", "  ")
	}

	return json.Marshal(configMap)
}

// BuildJsonFromPrefix método simplificado
func (b *ConfigBuilder) BuildJsonFromPrefix(ctx context.Context, prefix string, sortByDependencies bool) ([]byte, error) {
	opts := BuildOptions{
		Prefixes:           []string{prefix},
		StripPrefix:        true,
		JSONOutput:         true,
		YAMLRules:          false,
		SortByDependencies: sortByDependencies,
	}
	return b.BuildConfigFromPrefixes(ctx, opts)
}

// BuildYamlFromPrefix método simplificado
func (b *ConfigBuilder) BuildYamlFromPrefix(ctx context.Context, prefix string, sortByDependencies bool) ([]byte, error) {
	opts := BuildOptions{
		Prefixes:           []string{prefix},
		StripPrefix:        true,
		JSONOutput:         false,
		YAMLRules:          true,
		SortByDependencies: sortByDependencies,
	}
	return b.BuildConfigFromPrefixes(ctx, opts)
}
