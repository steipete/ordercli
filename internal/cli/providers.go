package cli

import "github.com/spf13/cobra"

func newFoodoraCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "foodora",
		Short: "foodora (via fd-api)",
	}
	cmd.AddCommand(newCountriesCmd(st))
	cmd.AddCommand(newConfigCmd(st))
	cmd.AddCommand(newCookiesCmd(st))
	cmd.AddCommand(newSessionCmd(st))
	cmd.AddCommand(newLoginCmd(st))
	cmd.AddCommand(newLogoutCmd(st))
	cmd.AddCommand(newOrdersCmd(st))
	cmd.AddCommand(newHistoryCmd(st))
	cmd.AddCommand(newOrderCmd(st))
	return cmd
}

func newDeliverooCmd(st *state) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deliveroo",
		Short: "Deliveroo",
	}
	cmd.AddCommand(newDeliverooConfigCmd(st))
	cmd.AddCommand(newDeliverooHistoryCmd(st))
	cmd.AddCommand(newDeliverooOrdersCmd(st))
	return cmd
}
