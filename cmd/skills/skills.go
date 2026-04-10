package skills

import (
	"github.com/spf13/cobra"
)

// Cmd is the root command for the skills package.
var Cmd = &cobra.Command{
	Use:   "skills",
	Short: "Run skill focused code",
}
