package store

import "fmt"

type EntityName struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

func (entityName *EntityName) String() string {
	if entityName.Namespace == "" {
		return fmt.Sprintf("%v/%v", entityName.Kind, entityName.Name)
	}
	return fmt.Sprintf("%v/%v/%v", entityName.Kind, entityName.Namespace, entityName.Name)
}
