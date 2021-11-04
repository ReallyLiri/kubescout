package dedup

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_dedup(t *testing.T) {

	assert.True(t, AreSimilar("", "", 0))
	assert.True(t, AreSimilar("", "", 0.5))
	assert.True(t, AreSimilar("", "", 1))

	assert.True(t, AreSimilar("a", "", 0))
	assert.False(t, AreSimilar("a", "", 0.1))
	assert.False(t, AreSimilar("", "a", 0.1))

	assert.True(t, AreSimilar(
		`Event by kubelet: Failed x since , :
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,
		`Event by kubelet: Failed x since , :
	Error: ErrImagePull`,
		0.1,
	))
	assert.False(t, AreSimilar(
		`Event by kubelet: Failed x since , :
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,
		`Event by kubelet: Failed x since , :
	Error: ErrImagePull`,
		0.75,
	))
	assert.False(t, AreSimilar(
		`Event by kubelet: Failed x since , :
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,
		`Event by kubelet: Failed x since , :
	Error: ImagePullBackOff`,
		0.75,
	))

	assert.True(t, AreSimilar(
		`Event by kubelet: Failed x since , :
	Error: ErrImagePull`,
		`Event by kubelet: Failed x since , :
	Error: ImagePullBackOff`,
		0.6,
	))
	assert.False(t, AreSimilar(
		`Event by kubelet: Failed x since , :
	Error: ErrImagePull`,
		`Event by kubelet: Failed x since , :
	Error: ImagePullBackOff`,
		0.95,
	))

	assert.True(t, AreSimilar(
		`Event by kernel-monitor: TaskHung since , :
INFO: task runc:[2:INIT]:293016 blocked for more than 327 seconds.`,
		`Event by kernel-monitor: TaskHung since , :
INFO: task runc:[2:INIT]:309147 blocked for more than 327 seconds.`,
		0.8,
	))
}
