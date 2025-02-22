package delete

import (
	"time"

	"github.com/weaveworks/eksctl/pkg/actions/cluster"

	"github.com/kris-nova/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
	"github.com/weaveworks/eksctl/pkg/printers"
)

func deleteClusterCmd(cmd *cmdutils.Cmd) {
	deleteClusterWithRunFunc(cmd, func(cmd *cmdutils.Cmd, force bool, disableNodegroupEviction bool, parallel int) error {
		return doDeleteCluster(cmd, force, disableNodegroupEviction, parallel)
	})
}

func deleteClusterWithRunFunc(cmd *cmdutils.Cmd, runFunc func(cmd *cmdutils.Cmd, force bool, disableNodegroupEviction bool, parallel int) error) {
	cfg := api.NewClusterConfig()
	cmd.ClusterConfig = cfg

	cmd.SetDescription("cluster", "Delete a cluster", "")

	var (
		force                    bool
		disableNodegroupEviction bool
		parallel                 int
	)
	cmd.CobraCommand.RunE = func(_ *cobra.Command, args []string) error {
		cmd.NameArg = cmdutils.GetNameArg(args)
		return runFunc(cmd, force, disableNodegroupEviction, parallel)
	}

	cmd.FlagSetGroup.InFlagSet("General", func(fs *pflag.FlagSet) {
		fs.StringVarP(&cfg.Metadata.Name, "name", "n", "", "EKS cluster name")
		cmdutils.AddRegionFlag(fs, &cmd.ProviderConfig)

		cmd.Wait = false
		cmdutils.AddWaitFlag(fs, &cmd.Wait, "deletion of all resources")
		fs.BoolVar(&force, "force", false, "Force deletion to continue when errors occur")
		fs.BoolVar(&disableNodegroupEviction, "disable-nodegroup-eviction", false, "Force drain to use delete, even if eviction is supported. This will bypass checking PodDisruptionBudgets, use with caution.")
		fs.IntVar(&parallel, "parallel", 1, "Number of nodes to drain in parallel. Max 25")

		cmdutils.AddConfigFileFlag(fs, &cmd.ClusterConfigFile)
		cmdutils.AddTimeoutFlag(fs, &cmd.ProviderConfig.WaitTimeout)
	})

	cmdutils.AddCommonFlagsForAWS(cmd.FlagSetGroup, &cmd.ProviderConfig, true)
}

func doDeleteCluster(cmd *cmdutils.Cmd, force bool, disableNodegroupEviction bool, parallel int) error {
	if err := cmdutils.NewMetadataLoader(cmd).Load(); err != nil {
		return err
	}

	cfg := cmd.ClusterConfig
	meta := cmd.ClusterConfig.Metadata
	printer := printers.NewJSONPrinter()
	ctl, err := cmd.NewProviderForExistingCluster()
	if err != nil {
		if !force {
			return err
		}
		// initialise the controller without refreshing the cluster status.
		// This can happen if the initial cluster stack failed to create the cluster,
		// but we still want to remove other created resources and the cluster stack.
		logger.Warning("failed to create provider for cluster; force = true skipping: %v", err)
		if ctl, err = cmd.NewCtl(); err != nil {
			return err
		}
	}

	logger.Info("deleting EKS cluster %q", meta.Name)
	if err := printer.LogObj(logger.Debug, "cfg.json = \\\n%s\n", cfg); err != nil {
		return err
	}

	cluster, err := cluster.New(cfg, ctl)
	if err != nil {
		return err
	}

	return cluster.Delete(time.Second*20, cmd.Wait, force, disableNodegroupEviction, parallel)
}
