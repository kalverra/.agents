package skills

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/config"
	"github.com/kalverra/agents/internal/output"
	"github.com/kalverra/agents/internal/ticket"
)

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Todoist and Jira issue helpers for session workflows",
}

var ticketFetchCmd = &cobra.Command{
	Use:   "fetch <task_id_or_url>",
	Short: "Fetch a Todoist or Jira task by id or task URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := ticketProvider(cmd, args[0])
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		tk, err := p.Fetch(ctx, args[0])
		if err != nil {
			return err
		}
		payload := tk.ToFetchPayload()
		if output.JSON() {
			_, err := fmt.Fprint(cmd.OutOrStdout(), ticket.FetchPayloadToAIXML(payload))
			return err
		}
		printTicket(tk)
		return nil
	},
}

var ticketCommentCmd = &cobra.Command{
	Use:   "comment <task_id_or_url>",
	Short: "Add a comment to a Todoist task or Jira issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := cmd.Flags().GetString("body")
		if err != nil {
			return err
		}
		p, err := ticketProvider(cmd, args[0])
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		if err := p.Comment(ctx, args[0], body); err != nil {
			return err
		}
		resolved, resErr := resolveCommentTarget(args[0])
		if resErr != nil {
			return resErr
		}
		output.Write("ticket-comment", map[string]string{"task_id": resolved}, func() {
			fmt.Printf("Comment posted on %s\n", resolved)
		})
		return nil
	},
}

func loadTodoist(cmd *cobra.Command) (*ticket.Todoist, error) {
	cfg, err := config.Load(config.WithFlags(cmd.Root().PersistentFlags()))
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return ticket.NewTodoist(*zerolog.Ctx(cmd.Context()), ticket.TodoistConfig{
		Token:   cfg.TodoistAPIToken,
		BaseURL: cfg.TodoistRESTBase,
	}), nil
}

func loadJira(cmd *cobra.Command) (*ticket.Jira, error) {
	cfg, err := config.Load(config.WithFlags(cmd.Root().PersistentFlags()))
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return ticket.NewJira(*zerolog.Ctx(cmd.Context()), ticket.JiraConfig{
		Email:    cfg.JiraEmail,
		APIToken: cfg.JiraAPIToken,
		Domain:   cfg.JiraDomain,
	}), nil
}

// ticketProvider picks Todoist vs Jira from the shape of the reference.
func ticketProvider(cmd *cobra.Command, raw string) (ticket.Provider, error) {
	_, err := ticket.ParseJiraRef(raw)
	if err == nil {
		return loadJira(cmd)
	}
	if !errors.Is(err, ticket.ErrNotJiraRef) {
		return nil, err
	}
	return loadTodoist(cmd)
}

func resolveCommentTarget(raw string) (string, error) {
	if k, err := ticket.ParseJiraRef(raw); err == nil {
		return k, nil
	} else if !errors.Is(err, ticket.ErrNotJiraRef) {
		return "", err
	}
	return ticket.ParseTaskRef(raw)
}

func printTicket(tk *ticket.Ticket) {
	fmt.Printf("ID:          %s\n", tk.ID)
	fmt.Printf("Title:       %s\n", tk.Title)
	if tk.Description != "" {
		fmt.Printf("Description: %s\n", tk.Description)
	}
	if tk.Status != "" {
		fmt.Printf("Status:      %s\n", tk.Status)
	}
	if tk.URL != "" {
		fmt.Printf("URL:         %s\n", tk.URL)
	}
	if len(tk.Comments) == 0 {
		fmt.Println("Comments:    (none)")
		return
	}
	fmt.Println("Comments:")
	for _, c := range tk.Comments {
		when := c.PostedAt
		if when == "" {
			when = "?"
		}
		fmt.Printf("  - [%s] %s\n", when, c.Content)
	}
}

func init() {
	ticketCommentCmd.Flags().String("body", "", "Comment body (required)")
	err := ticketCommentCmd.MarkFlagRequired("body")
	if err != nil {
		panic(err)
	}

	ticketCmd.AddCommand(ticketFetchCmd, ticketCommentCmd)
	Cmd.AddCommand(ticketCmd)
}
