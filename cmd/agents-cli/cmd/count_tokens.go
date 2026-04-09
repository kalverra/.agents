package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/pkoukk/tiktoken-go"
	"github.com/spf13/cobra"
)

var countTokensCmd = &cobra.Command{
	Use:   "count-tokens [FILE]",
	Short: "Count LLM tokens in a file or string",
	RunE: func(cmd *cobra.Command, args []string) error {
		str, _ := cmd.Flags().GetString("string")
		encoding, _ := cmd.Flags().GetString("encoding")
		model, _ := cmd.Flags().GetString("model")
		verbose, _ := cmd.Flags().GetBool("verbose")
		stripHookable, _ := cmd.Flags().GetBool("strip-hookable-markers")

		var text string
		if str != "" {
			text = str
		} else if len(args) > 0 {
			b, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			text = string(b)
		} else {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			text = string(b)
		}

		if stripHookable {
			text = stripHookableDelimiterLines(text)
		}

		var enc *tiktoken.Tiktoken
		var err error
		var encName string

		if model != "" {
			enc, err = tiktoken.EncodingForModel(model)
			if err != nil {
				return fmt.Errorf("unknown model for tiktoken: %s", model)
			}
			encName = fmt.Sprintf("model:%s", model)
		} else {
			enc, err = tiktoken.GetEncoding(encoding)
			if err != nil {
				return fmt.Errorf("unknown encoding: %s", encoding)
			}
			encName = encoding
		}

		tokens := enc.Encode(text, nil, nil)
		n := len(tokens)

		if verbose {
			fmt.Fprintf(os.Stderr, "encoding: %s\n", encName)
			fmt.Fprintf(os.Stderr, "chars:  %d\n", len(text))
			fmt.Fprintf(os.Stderr, "tokens: %d\n", n)
		} else {
			fmt.Println(n)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(countTokensCmd)
	countTokensCmd.Flags().StringP("string", "s", "", "Count tokens in this string instead of a file")
	countTokensCmd.Flags().String("encoding", "cl100k_base", "tiktoken encoding name")
	countTokensCmd.Flags().String("model", "", "Map via tiktoken.encoding_for_model")
	countTokensCmd.Flags().BoolP("verbose", "v", false, "Print encoding, character count, and token count")
	countTokensCmd.Flags().Bool("strip-hookable-markers", false, "Remove standalone <!-- hookable / /hookable --> lines before counting")
	countTokensCmd.MarkFlagsMutuallyExclusive("encoding", "model")
}
