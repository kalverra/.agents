package cmd

import (
	"fmt"
	"io"
	"os"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/markdown"
	"github.com/kalverra/agents/internal/ui"
)

var countTokensCmd = &cobra.Command{
	Use:   "count-tokens [FILE]",
	Short: "Count LLM tokens in a file, string, or stdin",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		str, _ := cmd.Flags().GetString("string")
		encoding, _ := cmd.Flags().GetString("encoding")
		model, _ := cmd.Flags().GetString("model")
		verbose, _ := cmd.Flags().GetBool("verbose")
		stripHookable, _ := cmd.Flags().GetBool("strip-hookable-markers")

		var text string
		switch {
		case str != "":
			text = str
		case len(args) > 0:
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			text = string(data)
		default:
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			text = string(data)
		}

		if stripHookable {
			text = markdown.StripHookableDelimiterLines(text)
		}

		var enc *tiktoken.Tiktoken
		var encName string

		if model != "" {
			e, err := tiktoken.EncodingForModel(model)
			if err != nil {
				return fmt.Errorf("unknown model for tiktoken: %s", model)
			}
			enc = e
			encName = fmt.Sprintf("model:%s", model)
		} else {
			e, err := tiktoken.GetEncoding(encoding)
			if err != nil {
				return fmt.Errorf("unknown encoding: %s", encoding)
			}
			enc = e
			encName = encoding
		}

		tokens := enc.Encode(text, nil, nil)
		n := len(tokens)

		if verbose {
			ui.WarnPrintf("encoding: %s\n", encName)
			ui.WarnPrintf("chars:  %d\n", len(text))
			ui.WarnPrintf("tokens: %d\n", n)
		} else {
			if ui.AIOutput {
				ui.Printf("%d\n", n)
			} else {
				fmt.Println(n) // Keep simple for piping if not in AI mode
			}
		}

		return nil
	},
}

func init() {
	countTokensCmd.Flags().StringP("string", "s", "", "Count tokens in this string instead of a file")
	countTokensCmd.Flags().String("encoding", "cl100k_base", "tiktoken encoding name")
	countTokensCmd.Flags().String("model", "", "Map via tiktoken model (e.g. gpt-4o)")
	countTokensCmd.Flags().BoolP("verbose", "v", false, "Print encoding, char count, and token count")
	countTokensCmd.Flags().Bool("strip-hookable-markers", false, "Remove hookable tag lines before counting")
	rootCmd.AddCommand(countTokensCmd)
}
