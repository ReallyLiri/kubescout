package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config, err := DefaultConfig()
	require.Nil(t, err)
	require.NotNil(t, config)
	require.NotEqual(t, int64(0), config.PodLogsTail)
	require.NotEqual(t, int64(0), config.EventsLimit)
	require.Equal(t, log.InfoLevel, log.GetLevel())
	require.NotEqual(t, time.Duration(0), config.MessagesDeduplicationDuration)
	require.NotEqual(t, "", config.OutputMode)
	require.NotNil(t, config.Locale)
	require.Equal(t, time.UTC, config.Locale)
	require.True(t, strings.HasSuffix(config.KubeconfigFilePath, "/.kube/config"))
}

func TestFromArgs(t *testing.T) {
	config, err := FromArgs([]string{
		"executable",
		"--vv",
		"-k",
		"path/kubeconfig",
		"--locale",
		"Asia/Kolkata",
		"--exclude-ns",
		"ns1,ns2",
		"--include-ns",
		"ns3",
		"-d",
		"17",
		"-s",
		"",
		"-o",
		"foo",
	})
	require.Nil(t, err)
	require.NotNil(t, config)
	require.NotEqual(t, int64(0), config.PodLogsTail)
	require.NotEqual(t, int64(0), config.EventsLimit)
	require.Equal(t, log.TraceLevel, log.GetLevel())
	require.Equal(t, time.Duration(17)*time.Minute, config.MessagesDeduplicationDuration)
	require.NotEqual(t, "pretty", config.OutputMode)
	require.NotNil(t, config.Locale)
	require.Equal(t, time.UTC, config.Locale)
	require.Equal(t, "path/kubeconfig", config.KubeconfigFilePath)
	require.Equal(t, []string{"ns1", "ns2"}, config.ExcludeNamespaces)
	require.Equal(t, []string{"ns3"}, config.IncludeNamespaces)
}
