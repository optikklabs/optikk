package cmd

import (
	"fmt"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/spf13/cobra"
)

func newTeamCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Create teams and add members",
	}
	cmd.AddCommand(newTeamCreateCmd(app), newTeamMemberCmd(app))
	return cmd
}

// adminClient builds an API client from the cached admin session.
func adminClient() (*apiclient.Client, error) {
	apiBase, token, err := apiclient.LoadToken()
	if err != nil {
		return nil, err
	}
	c := apiclient.New(apiBase)
	c.SetToken(token)
	return c, nil
}

func newTeamCreateCmd(_ *App) *cobra.Command {
	var org, slug string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a team (returns team_id + api_key)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := adminClient()
			if err != nil {
				return err
			}
			name := args[0]
			if org == "" {
				org = name
			}
			team, err := client.CreateTeam(cmd.Context(), apiclient.CreateTeamRequest{
				TeamName: name, OrgName: org, Slug: slug,
			})
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "team_id: %d\nslug:    %s\napi_key: %s\n", team.ID, team.Slug, team.APIKey)
			fmt.Fprintf(w, "next: optikk tenant onboard %s --key %s\n", team.Slug, team.APIKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&org, "org", "", "organization name (default: team name)")
	cmd.Flags().StringVar(&slug, "slug", "", "team slug (server-generated if empty)")
	return cmd
}

func newTeamMemberCmd(_ *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage team members",
	}

	var (
		teamID   int64
		password string
		name     string
		role     string
	)
	add := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a member to a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := adminClient()
			if err != nil {
				return err
			}
			email := args[0]
			if name == "" {
				name = email
			}
			user, err := client.CreateUser(cmd.Context(), apiclient.CreateUserRequest{
				Email: email, Name: name, Password: password, Role: role, TeamID: teamID,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "user_id: %d  email: %s  team: %d\n", user.ID, user.Email, teamID)
			return nil
		},
	}
	add.Flags().Int64Var(&teamID, "team", 0, "team id to add the member to (required)")
	add.Flags().StringVar(&password, "password", "", "member password (required)")
	add.Flags().StringVar(&name, "name", "", "member display name (default: email)")
	add.Flags().StringVar(&role, "role", "", "member role")
	add.MarkFlagRequired("team")
	add.MarkFlagRequired("password")
	cmd.AddCommand(add)
	return cmd
}
