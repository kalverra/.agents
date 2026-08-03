package skills

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/config"
	"github.com/kalverra/agents/internal/git"
	"github.com/kalverra/agents/internal/output"
	"github.com/kalverra/agents/internal/ticket"
)

var TicketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Todoist and Jira issue helpers for session workflows",
}

var ticketFetchCmd = &cobra.Command{
	Use:   "fetch <task_id_or_url_or_jira_key>",
	Short: "Fetch a Todoist or Jira task by id, task URL, or Jira key (defaults to Todoist)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		useJira, _ := cmd.Flags().GetBool("jira")
		ref := args[0]
		ctx := cmd.Context()

		var tk *ticket.Ticket
		var err error

		if !useJira {
			td, loadErr := loadTodoist(cmd)
			if loadErr != nil {
				return loadErr
			}
			if key, parseErr := ticket.ParseJiraRef(ref); parseErr == nil {
				tk, err = td.SearchTasksByJiraKey(ctx, key)
			} else {
				tk, err = td.Fetch(ctx, ref)
			}
			if err != nil && errors.Is(err, ticket.ErrNotJiraRef) {
				// Fall back to Jira if not found or ref shape demands Jira
				j, jErr := loadJira(cmd)
				if jErr == nil {
					tk, err = j.Fetch(ctx, ref)
				}
			}
		} else {
			j, jErr := loadJira(cmd)
			if jErr != nil {
				return jErr
			}
			tk, err = j.Fetch(ctx, ref)
		}

		if err != nil {
			return err
		}

		// Write <ID>.md markdown file for AI agent context
		fileName := tk.ID + ".md"
		mdContent := ticket.TicketToMarkdownFile(tk)
		_ = os.WriteFile(fileName, []byte(mdContent), 0644)

		payload := tk.ToFetchPayload()
		if output.JSON() {
			outXML := fmt.Sprintf(
				"<ticket_fetch status=\"ok\" file=%q>\n%s</ticket_fetch>",
				fileName,
				ticket.FetchPayloadToAIXML(payload),
			)
			output.Write("ticket-fetch", outXML, nil)
			return nil
		}
		printTicket(tk)
		fmt.Printf("\nSaved ticket context to %s\n", fileName)
		return nil
	},
}

var ticketCommentCmd = &cobra.Command{
	Use:   "comment <task_id_or_url_or_jira_key>",
	Short: "Add a comment to a task (defaults to Todoist)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := cmd.Flags().GetString("body")
		if err != nil {
			return err
		}
		useJira, _ := cmd.Flags().GetBool("jira")
		ref := args[0]
		ctx := cmd.Context()

		var targetID string

		if !useJira {
			td, loadErr := loadTodoist(cmd)
			if loadErr != nil {
				return loadErr
			}
			if key, parseErr := ticket.ParseJiraRef(ref); parseErr == nil {
				matchedTask, sErr := td.SearchTasksByJiraKey(ctx, key)
				if sErr != nil {
					return fmt.Errorf("could not find Todoist task for Jira key %s: %w", key, sErr)
				}
				targetID = matchedTask.ID
			} else {
				resolved, rErr := ticket.ParseTaskRef(ref)
				if rErr != nil {
					return rErr
				}
				targetID = resolved
			}

			if err := td.Comment(ctx, targetID, body); err != nil {
				return err
			}
		} else {
			j, jErr := loadJira(cmd)
			if jErr != nil {
				return jErr
			}
			key, parseErr := ticket.ParseJiraRef(ref)
			if parseErr != nil {
				return parseErr
			}
			targetID = key
			if err := j.Comment(ctx, targetID, body); err != nil {
				return err
			}
		}

		if output.JSON() {
			outXML := fmt.Sprintf("<ticket_comment task_id=%q status=\"posted\"/>", targetID)
			output.Write("ticket-comment", outXML, nil)
			return nil
		}

		output.Write("ticket-comment", map[string]string{"task_id": targetID}, func() {
			fmt.Printf("Comment posted on %s\n", targetID)
		})
		return nil
	},
}

var ticketCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Jira issue (defaults to Jira Cloud with --epic support; use --todoist for Todoist)",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")
		useTodoist, _ := cmd.Flags().GetBool("todoist")
		epicKey, _ := cmd.Flags().GetString("epic")
		projectKey, _ := cmd.Flags().GetString("project")
		issueType, _ := cmd.Flags().GetString("type")

		ctx := cmd.Context()
		cfg, loadErr := config.Load(config.WithFlags(cmd.Root().PersistentFlags()))
		if loadErr != nil {
			return loadErr
		}

		var tk *ticket.Ticket
		var err error

		if useTodoist {
			td, tErr := loadTodoist(cmd)
			if tErr != nil {
				return tErr
			}
			tk, err = td.CreateTask(ctx, ticket.CreateTaskRequest{
				Title:       title,
				Description: desc,
			})
		} else {
			j, jErr := loadJira(cmd)
			if jErr != nil {
				return jErr
			}
			if projectKey == "" {
				projectKey = cfg.JiraDefaultProject
			}
			tk, err = j.CreateIssue(ctx, ticket.CreateIssueRequest{
				Project:     projectKey,
				Summary:     title,
				Description: desc,
				IssueType:   issueType,
				EpicKey:     epicKey,
			})
		}

		if err != nil {
			return err
		}

		if output.JSON() {
			outXML := fmt.Sprintf("<ticket_create id=%q title=%q url=%q/>", tk.ID, tk.Title, tk.URL)
			output.Write("ticket-create", outXML, nil)
			return nil
		}

		output.Write("ticket-create", tk, func() {
			fmt.Printf("Created ticket %s: %s\nURL: %s\n", tk.ID, tk.Title, tk.URL)
		})
		return nil
	},
}

var ticketEpicsCmd = &cobra.Command{
	Use:   "epics",
	Short: "List active Jira Epics assigned to current user or updated recently",
	RunE: func(cmd *cobra.Command, args []string) error {
		j, err := loadJira(cmd)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		epics, err := j.ListEpics(ctx)
		if err != nil {
			return err
		}
		if output.JSON() {
			var b strings.Builder
			b.WriteString("<epics>\n")
			for _, e := range epics {
				fmt.Fprintf(&b, "  <epic id=%q title=%q status=%q/>\n", e.ID, e.Title, e.Status)
			}
			b.WriteString("</epics>")
			output.Write("ticket-epics", b.String(), nil)
			return nil
		}
		output.Write("ticket-epics", epics, func() {
			if len(epics) == 0 {
				fmt.Println("No active Jira Epics found.")
				return
			}
			fmt.Println("Active Jira Epics:")
			for _, e := range epics {
				fmt.Printf("  - %s: %s [%s]\n", e.ID, e.Title, e.Status)
			}
		})
		return nil
	},
}

var ticketStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Inspect current working directory git repo, work status, and branch Jira ticket",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.WithFlags(cmd.Root().PersistentFlags()))
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		repo, repoErr := git.DetectRepo(".")
		isWork := false
		jiraKey := ""
		defaultBranch := ""
		if repoErr == nil && repo != nil {
			isWork = ticket.IsWorkRepo(repo, cfg.WorkRepos)
			jiraKey = ticket.ExtractJiraKey(repo.Branch)
			defaultBranch, _ = git.DetectDefaultBranch(".")
		}

		ctx := cmd.Context()
		var matchedTodoist *ticket.Ticket
		var matchedJira *ticket.Ticket
		var candidateEpics []*ticket.Ticket
		var assignedTickets []*ticket.Ticket
		writtenFile := ""

		if jiraKey != "" {
			if td, tErr := loadTodoist(cmd); tErr == nil {
				matchedTodoist, _ = td.SearchTasksByJiraKey(ctx, jiraKey)
			}
			if j, jErr := loadJira(cmd); jErr == nil {
				matchedJira, _ = j.Fetch(ctx, jiraKey)
			}

			// Automatically write <KEY>.md file if ticket data is fetched
			var targetTicket *ticket.Ticket
			if matchedJira != nil {
				targetTicket = matchedJira
			} else if matchedTodoist != nil {
				targetTicket = matchedTodoist
			}
			if targetTicket != nil {
				writtenFile = jiraKey + ".md"
				mdContent := ticket.TicketToMarkdownFile(targetTicket)
				_ = os.WriteFile(writtenFile, []byte(mdContent), 0644)
			}
		} else if isWork {
			if j, jErr := loadJira(cmd); jErr == nil {
				candidateEpics, _ = j.ListEpics(ctx)
				assignedTickets, _ = j.ListAssignedTickets(ctx)
			}
		}

		var b strings.Builder
		if repoErr != nil {
			b.WriteString("Git status: Not in a git repository\n")
		} else {
			fmt.Fprintf(&b, "Repo:        %s/%s\n", repo.Owner, repo.Name)
			fmt.Fprintf(&b, "Branch:      %s\n", formatBranchOutput(repo.Branch, defaultBranch))
			fmt.Fprintf(&b, "Is Work:     %t\n", isWork)
			if jiraKey != "" {
				fmt.Fprintf(&b, "Jira Key:    %s\n", jiraKey)
			} else {
				b.WriteString("Jira Key:    (none)\n")
			}
			if writtenFile != "" {
				fmt.Fprintf(&b, "Ticket File: ./%s\n", writtenFile)
			}
			if matchedTodoist != nil {
				fmt.Fprintf(&b, "Todoist:     %s (%s)\n", matchedTodoist.Title, matchedTodoist.URL)
			}
			if matchedJira != nil {
				fmt.Fprintf(&b, "Jira Ticket: %s (%s)\n", matchedJira.Title, matchedJira.URL)
			}
			if len(candidateEpics) > 0 {
				b.WriteString("\nActive Epics:\n")
				for _, e := range candidateEpics {
					fmt.Fprintf(&b, "  - %s: %s [%s]\n", e.ID, e.Title, e.Status)
				}
			}
			if len(assignedTickets) > 0 {
				b.WriteString("\nAssigned Tickets:\n")
				for _, t := range assignedTickets {
					fmt.Fprintf(&b, "  - %s: %s [%s]\n", t.ID, t.Title, t.Status)
				}
			}
		}

		statusText := b.String()
		output.Write("ticket-status", statusText, func() {
			fmt.Print(statusText)
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

func formatBranchOutput(currentBranch, defaultBranch string) string {
	if currentBranch == "" {
		return "(none)"
	}
	if defaultBranch != "" && currentBranch == defaultBranch {
		return currentBranch + " (default)"
	}
	return currentBranch
}

func init() {
	ticketFetchCmd.Flags().Bool("jira", false, "Force fetch directly from Jira Cloud")

	ticketCommentCmd.Flags().String("body", "", "Comment body (required)")
	_ = ticketCommentCmd.MarkFlagRequired("body")
	ticketCommentCmd.Flags().Bool("jira", false, "Post comment directly to Jira Cloud instead of Todoist")

	ticketCreateCmd.Flags().String("title", "", "Task title/summary (required)")
	_ = ticketCreateCmd.MarkFlagRequired("title")
	ticketCreateCmd.Flags().String("description", "", "Task description")
	ticketCreateCmd.Flags().String("epic", "", "Parent Jira Epic key (e.g. DX-50)")
	ticketCreateCmd.Flags().String("project", "", "Jira project key (defaults to JIRA_DEFAULT_PROJECT or DX)")
	ticketCreateCmd.Flags().String("type", "Task", "Jira issue type")
	ticketCreateCmd.Flags().Bool("todoist", false, "Create task in Todoist instead of Jira Cloud")

	TicketCmd.AddCommand(ticketFetchCmd, ticketCommentCmd, ticketCreateCmd, ticketStatusCmd, ticketEpicsCmd)
	Cmd.AddCommand(TicketCmd)
}
