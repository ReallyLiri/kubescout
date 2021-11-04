package kubeclient

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const pageSize = 32

var metadataAccessor = meta.NewAccessor()

func pagedGet(
	initialOpts *metaV1.ListOptions,
	listFunc func(metaV1.ListOptions) (runtime.Object, error),
) error {
	opts := initialOpts
	if opts == nil {
		opts = &metaV1.ListOptions{
			Limit: pageSize,
		}
	}
	for {
		list, err := listFunc(*opts)
		if err != nil {
			return err
		}
		nextContinueToken, _ := metadataAccessor.Continue(list)
		if len(nextContinueToken) == 0 {
			return nil
		}
		opts.Continue = nextContinueToken
	}
}
