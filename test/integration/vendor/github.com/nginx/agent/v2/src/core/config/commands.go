/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var ROOT_COMMAND = &cobra.Command{
	Use:   "nginx-agent [flags]",
	Short: "nginx-agent",
}

var COMPLETION_COMMAND = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate completion script.",
	Long: `To load completions:

Bash:

$ source <(nginx-agent completion bash)

# To load completions for each session, execute once:
Linux:
  $ nginx-agent completion bash > /etc/bash_completion.d/nginx-agent
MacOS:
  $ nginx-agent completion bash > /usr/local/etc/bash_completion.d/nginx-agent

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ nginx-agent completion zsh > "${fpath[1]}/_nginx-agent"

# You will need to start a new shell for this setup to take effect.

Fish:

$ nginx-agent completion fish | source

# To load completions for each session, execute once:
$ nginx-agent completion fish > ~/.config/fish/completions/nginx-agent.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		}

		if err != nil {
			log.Warnf("Error sending command: %v", err)
		}
	},
}
