package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rumpl/structs/internal/skill"
)

func newSkillCommand(userHomeDir func() (string, error)) *cobra.Command {
	command := &cobra.Command{
		Use:           "skill",
		Short:         "Manage the bundled agent skill",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	command.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install the agent skill in ~/.agents/skills/structs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := userHomeDir()
			if err != nil {
				return fmt.Errorf("locate home directory: %w", err)
			}

			path, err := skill.Install(home)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
				return fmt.Errorf("write skill location: %w", err)
			}

			return nil
		},
	})

	return command
}
