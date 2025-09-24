package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"gopkg.in/yaml.v3"
)

// buildStructure constrói a estrutura JSON a partir dos parâmetros
func (b *ConfigBuilder) buildStructure(params []types.Parameter, basePath string, stripPrefix, sortByDependencies bool) map[string]interface{} {
	if len(params) == 0 {
		return make(map[string]interface{})
	}

	// Organiza os parâmetros por nível
	levels := b.organizeParametersByLevel(params, basePath, stripPrefix)

	return b.buildGenericStructure(levels)
}

// buildGenericStructure constrói estrutura genérica sem ordenação por dependências
func (b *ConfigBuilder) buildGenericStructure(levels map[string]map[string]types.Parameter) map[string]interface{} {
	result := make(map[string]interface{})

	for levelKey, levelParams := range levels {
		if levelKey == "." {
			b.processRootLevel(result, levelParams)
		} else {
			b.processNestedLevel(result, levelKey, levelParams)
		}
	}

	return result
}

// processRootLevel processa parâmetros no nível raiz
func (b *ConfigBuilder) processRootLevel(result map[string]interface{}, levelParams map[string]types.Parameter) {
	if len(levelParams) > 1 {
		// Múltiplos parâmetros - array
		result["items"] = b.buildArrayFromMap(levelParams)
	} else {
		// Único parâmetro - objeto
		for paramName, param := range levelParams {
			result[paramName] = b.parseParameterValue(*param.Value)
		}
	}
}

// processNestedLevel processa parâmetros em níveis aninhados
func (b *ConfigBuilder) processNestedLevel(result map[string]interface{}, levelKey string, levelParams map[string]types.Parameter) {
	if len(levelParams) == 1 {
		// Único parâmetro
		for childPath, param := range levelParams {
			if childPath == "." {
				result[levelKey] = b.parseParameterValue(*param.Value)
			} else {
				result[levelKey] = b.buildNestedObject(childPath, param)
			}
		}
	} else {
		// Múltiplos parâmetros
		if b.shouldBeArray(levelParams) {
			result[levelKey] = b.buildArrayFromMap(levelParams)
		} else {
			result[levelKey] = b.buildNestedStructure(levelParams)
		}
	}
}

// buildNestedStructure constrói estrutura aninhada complexa
func (b *ConfigBuilder) buildNestedStructure(levelParams map[string]types.Parameter) map[string]interface{} {
	result := make(map[string]interface{})

	for childPath, param := range levelParams {
		pathParts := strings.Split(childPath, "/")
		current := result

		for i, part := range pathParts {
			if i == len(pathParts)-1 {
				// Última parte - valor final
				current[part] = b.parseParameterValue(*param.Value)
			} else {
				// Parte intermediária - navega ou cria
				if existing, exists := current[part]; exists {
					if childMap, ok := existing.(map[string]interface{}); ok {
						current = childMap
					} else {
						// Conflito - substitui por mapa
						newMap := make(map[string]interface{})
						current[part] = newMap
						current = newMap
					}
				} else {
					newMap := make(map[string]interface{})
					current[part] = newMap
					current = newMap
				}
			}
		}
	}

	return result
}

// buildNestedObject constrói objeto aninhado simples
func (b *ConfigBuilder) buildNestedObject(childPath string, param types.Parameter) map[string]interface{} {
	pathParts := strings.Split(childPath, "/")
	result := make(map[string]interface{})
	current := result

	for i, part := range pathParts {
		if i == len(pathParts)-1 {
			current[part] = b.parseParameterValue(*param.Value)
		} else {
			current[part] = make(map[string]interface{})
			current = current[part].(map[string]interface{})
		}
	}

	return result
}

// shouldBeArray verifica se os parâmetros devem formar um array
func (b *ConfigBuilder) shouldBeArray(levelParams map[string]types.Parameter) bool {
	if len(levelParams) <= 1 {
		return false
	}

	// Verifica se todos estão no mesmo nível de profundidade
	firstDepth := -1
	for childPath := range levelParams {
		depth := strings.Count(childPath, "/")
		if firstDepth == -1 {
			firstDepth = depth
		} else if depth != firstDepth {
			return false
		}
	}

	return firstDepth == 0
}

// buildArrayFromMap constrói array a partir de mapa de parâmetros
func (b *ConfigBuilder) buildArrayFromMap(params map[string]types.Parameter) []interface{} {
	result := make([]interface{}, 0, len(params))
	for _, param := range params {
		result = append(result, b.parseParameterValue(*param.Value))
	}
	return result
}

// organizeParametersByLevel organiza parâmetros por nível hierárquico
func (b *ConfigBuilder) organizeParametersByLevel(params []types.Parameter, basePath string, stripPrefix bool) map[string]map[string]types.Parameter {
	levels := make(map[string]map[string]types.Parameter)

	for _, param := range params {
		relativePath := b.extractRelativePath(*param.Name, basePath, stripPrefix)

		if relativePath == "" {
			// Parâmetro no nível raiz
			paramName := b.getLastPathSegment(*param.Name)
			if levels["."] == nil {
				levels["."] = make(map[string]types.Parameter)
			}
			levels["."][paramName] = param
		} else {
			pathParts := strings.Split(relativePath, "/")

			if len(pathParts) == 1 {
				// Parâmetro no primeiro nível
				levelKey := pathParts[0]
				if levels[levelKey] == nil {
					levels[levelKey] = make(map[string]types.Parameter)
				}
				levels[levelKey]["."] = param
			} else {
				// Parâmetro em nível aninhado
				parentLevel := pathParts[0]
				childPath := strings.Join(pathParts[1:], "/")

				if levels[parentLevel] == nil {
					levels[parentLevel] = make(map[string]types.Parameter)
				}
				levels[parentLevel][childPath] = param
			}
		}
	}

	return levels
}

// extractRelativePath extrai o caminho relativo ao basePath
func (b *ConfigBuilder) extractRelativePath(fullPath, basePath string, stripPrefix bool) string {
	if stripPrefix {
		basePath = strings.TrimSuffix(basePath, "/")
		relative := strings.TrimPrefix(fullPath, basePath)
		return strings.Trim(relative, "/")
	}
	return fullPath
}

// getLastPathSegment retorna o último segmento de um path
func (b *ConfigBuilder) getLastPathSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// parseParameterValue parse o valor do parâmetro
func (b *ConfigBuilder) parseParameterValue(value string) interface{} {
	var result interface{}
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return value
	}
	return result
}

// mergeMaps faz merge de dois maps recursivamente
func (b *ConfigBuilder) mergeMaps(dest, src map[string]interface{}) {
	for key, srcValue := range src {
		if destValue, exists := dest[key]; exists {
			if destMap, ok := destValue.(map[string]interface{}); ok {
				if srcMap, ok := srcValue.(map[string]interface{}); ok {
					b.mergeMaps(destMap, srcMap)
					continue
				}
			}
			if destArray, ok := destValue.([]interface{}); ok {
				if srcArray, ok := srcValue.([]interface{}); ok {
					dest[key] = append(destArray, srcArray...)
					continue
				}
			}
		}
		dest[key] = srcValue
	}
}

// getParametersByPath recupera parâmetros recursivamente
func (b *ConfigBuilder) getParametersByPath(ctx context.Context, path string) ([]types.Parameter, error) {
	var allParams []types.Parameter
	var nextToken *string

	for {
		input := &ssm.GetParametersByPathInput{
			Path:      aws.String(path),
			Recursive: aws.Bool(true),
			NextToken: nextToken,
		}

		result, err := b.ssmClient.GetParametersByPath(ctx, input)
		if err != nil {
			return nil, err
		}

		allParams = append(allParams, result.Parameters...)
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	// Ordena por nome para consistência
	sort.Slice(allParams, func(i, j int) bool {
		return *allParams[i].Name < *allParams[j].Name
	})

	return allParams, nil
}

// sortTypesByDependencies reordena os tipos com base nas dependências
func sortTypesByDependencies(schema map[string]interface{}) error {
	types, ok := schema["types"].(interface{})
	if !ok {
		return fmt.Errorf("'types' não encontrado ou não é uma lista")
	}

	// Mapeia os tipos por nome e suas dependências
	typeMap := make(map[string]map[string]interface{})
	dependencyMap := make(map[string][]string)

	for _, t := range types {
		typeObj := t.(map[string]interface{})
		name := typeObj["name"].(string)
		typeMap[name] = typeObj

		// Captura dependências (ofType) nos campos
		fields, ok := typeObj["fields"].([]interface{})
		if ok {
			for _, f := range fields {
				field := f.(map[string]interface{})
				if ofType, exists := field["ofType"].(string); exists {
					dependencyMap[name] = append(dependencyMap[name], ofType)
				}
			}
		}
	}

	// Ordenação topológica
	var sortedTypes []map[string]interface{}
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("ciclo detectado com o tipo: %s", name)
		}
		if visited[name] {
			return nil
		}
		inStack[name] = true
		for _, dep := range dependencyMap[name] {
			if _, exists := typeMap[dep]; exists {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		inStack[name] = false
		visited[name] = true
		sortedTypes = append(sortedTypes, typeMap[name])
		return nil
	}

	for name := range typeMap {
		if err != visit(name); err != nil {
			return err
		}
	}

	// Substitui os tipos ordenados do schema
	schema["types"] = sortedTypes
	return nil
}

// buildYAMLStructure constrói a estrutura YAML a partir dos parâmetros
func (b *ConfigBuilder) buildYAMLStructure(params []types.Parameter, basePath string, stripPrefix bool) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, param := range params {
		value := *param.Value
		relative := b.extractRelativePath(*param.Name, basePath, stripPrefix)
		if strings.Contains(relative, "/") {
			return nil, fmt.Errorf("parâmetros aninhados não são suportados para regras YAML: %s", *param.Name)
		}
		if relative == "" {
			relative = b.getLastPathSegment(*param.Name)
		}

		// Tenta parsear como map (para YAML completo ou submapas)
		var m map[string]interface{}
		err := yaml.Unmarshal([]byte(value), &m)
		if err == nil {
			b.mergeMaps(result, m)
			continue
		}

		// Se não, tenta parsear como lista (para regra individual)
		var l []interface{}
		err = yaml.Unmarshal([]byte(value), &l)
		if err != nil {
			return nil, fmt.Errorf("falha ao parsear YAML como map ou lista em %s: %w", *param.Name, err)
		}

		// Verifica duplicados
		if _, exists := result[relative]; exists {
			return nil, fmt.Errorf("chave de regra duplicada: %s", relative)
		}

		result[relative] = l
	}

	return result, nil
}
