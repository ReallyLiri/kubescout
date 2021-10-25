package kubecontext

import "gopkg.in/yaml.v3"

func valueOf(mapNode *yaml.Node, key string) *yaml.Node {
	if mapNode.Kind != yaml.MappingNode {
		return nil
	}
	for i, node := range mapNode.Content {
		if i%2 == 0 && node.Kind == yaml.ScalarNode && node.Value == key {
			return mapNode.Content[i+1]
		}
	}
	return nil
}
