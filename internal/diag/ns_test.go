package diag

import (
	"github.com/reallyliri/kubescout/internal/kubeclient"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetNamespaces(t *testing.T) {
	namespaces, err := kubeclient.GetNamespaces(t, "namespaces.json")
	require.Nil(t, err)
	require.NotNil(t, namespaces)
	require.Equal(t, 8, len(namespaces))

	for _, namespace := range namespaces {
		require.NotEmpty(t, namespace.Name)
	}
}
