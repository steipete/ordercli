package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Run(ctx context.Context, args []string) error {
	root := newRoot()
	root.SetArgs(args)
	root.SetContext(ctx)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func newRoot() *cobra.Command {
	var cfgPath string

	cmd := &cobra.Command{
		Use:   "foodoracli",
		Short: "foodora order status CLI",
	}
	cmd.PersistentFlags().StringVar(&cfgPath, "config", "", "config path (default: OS config dir)")

	st := &state{}
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		st.configPath = cfgPath
		return st.load()
	}
	cmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		return st.save()
	}

	cmd.AddCommand(newCountriesCmd(st))
	cmd.AddCommand(newConfigCmd(st))
	cmd.AddCommand(newLoginCmd(st))
	cmd.AddCommand(newLogoutCmd(st))
	cmd.AddCommand(newOrdersCmd(st))
	cmd.AddCommand(newOrderCmd(st))

	return cmd
}
