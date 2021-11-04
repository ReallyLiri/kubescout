package diag

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_splitToWords(t *testing.T) {
	assert.Equal(t, "", splitToWords(""))
	assert.Equal(t, "a", splitToWords("a"))
	assert.Equal(t, "a a", splitToWords("a a"))
	assert.Equal(t, "a A", splitToWords("a A"))
	assert.Equal(t, "a A B b", splitToWords("a A B b"))
	assert.Equal(t, "Pod Is Too Broken", splitToWords("PodIsTooBroken"))
	assert.Equal(t, "Kubelet Has Sufficient PID", splitToWords("KubeletHasSufficientPID"))
}

func Test_formatUnitsSize(t *testing.T) {
	assert.Equal(t, "", formatUnitsSize(""))
	assert.Equal(t, "abc", formatUnitsSize("abc"))
	assert.Equal(
		t,
		"Pod is in Failed phase due to Evicted: The node was low on resource: inodes.",
		formatUnitsSize("Pod is in Failed phase due to Evicted: The node was low on resource: inodes."),
	)
	assert.Equal(
		t,
		"Pod is in Failed phase due to Evicted: The node was low on resource: memory. Container memory-bomb-container was using 24GB, which exceeds its request of 0.",
		formatUnitsSize("Pod is in Failed phase due to Evicted: The node was low on resource: memory. Container memory-bomb-container was using 23313696Ki, which exceeds its request of 0."),
	)
	assert.Equal(
		t,
		"Pod is in Failed phase due to Evicted: The node was low on resource: memory. Container memory-bomb-container was using 3.0GB, which exceeds its request of 2.1GB.",
		formatUnitsSize("Pod is in Failed phase due to Evicted: The node was low on resource: memory. Container memory-bomb-container was using 2890108Ki, which exceeds its request of 2000Mi."),
	)
	assert.Equal(
		t,
		"Failed Create: pods \"app8-757699f8f9-lcfkd\" is forbidden: exceeded quota: resource-quota, requested: cpu=0.5, used: cpu=6.5, limited: cpu=7 (last transition: 7 minutes ago)",
		formatUnitsSize("Failed Create: pods \"app8-757699f8f9-lcfkd\" is forbidden: exceeded quota: resource-quota, requested: cpu=500m, used: cpu=6550m, limited: cpu=7 (last transition: 7 minutes ago)"),
	)
}

func Test_asTime(t *testing.T) {
	asTime := asTime("2021-10-05T13:27:43Z")
	assert.Equal(t, int64(1633440463), asTime.Unix())
}

func Test_formatResourceUsage(t *testing.T) {
	assert.Equal(t, "", formatResourceUsage(19, 20, "CPU", 0.9))
	assert.Equal(t, "Excessive usage of CPU: 19/20 (95.0% usage)", formatResourceUsage(1, 20, "CPU", 0.9))
	assert.Equal(t, "", formatResourceUsage(48433408, 53485824, "Memory", 0.75))
	assert.Equal(t, "Excessive usage of Memory: 48MB/54MB (90.6% usage)", formatResourceUsage(5052416, 53485824, "Memory", 0.75))
}
