package cmd

import (
	"os"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile    string
	etcdClient *clientv3.Client
	storage    cke.Storage
	inf        = &cliInfrastructure{}
)

func loadConfig(p string) (*etcdutil.Config, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := cke.NewEtcdConfig()
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ckecli",
	Short: "command-line interface to control CKE",
	Long: `ckecli is a command-line interface to control CKE.

It does not communicate CKE server; instead it communicates
with etcd.  CKE server watches etcd to receive any updates.`,
	Version: cke.Version,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := well.LogConfig{}.Apply()
		if err != nil {
			return err
		}

		cfg, err := loadConfig(cfgFile)
		if err != nil {
			return err
		}

		etcd, err := etcdutil.NewClient(cfg)
		if err != nil {
			return err
		}
		etcdClient = etcd

		storage = cke.Storage{Client: etcd}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if etcdClient != nil {
			etcdClient.Close()
		}
		inf.Close()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.ErrorExit(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/cke/config.yml", "config file")
}
