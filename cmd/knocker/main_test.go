package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestApplyConfigDefaultsSetsPersistentFlagsFromConfig(t *testing.T) {
	v := viper.New()

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Int("ttl", 0, "")
	require.NoError(t, v.BindPFlag("ttl", root.PersistentFlags().Lookup("ttl")))

	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)

	v.Set("ttl", 60)

	applyConfigDefaults(child, v)

	ttl, err := root.PersistentFlags().GetInt("ttl")
	require.NoError(t, err)
	require.Equal(t, 60, ttl)
	require.Equal(t, 60, v.GetInt("ttl"))
}
