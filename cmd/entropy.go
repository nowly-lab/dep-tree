package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gabotechs/dep-tree/internal/config"
	"github.com/gabotechs/dep-tree/internal/entropy"
	"github.com/gabotechs/dep-tree/internal/graph"
	"github.com/gabotechs/dep-tree/internal/language"
)

func EntropyCmd(cfgF func() (*config.Config, error)) *cobra.Command {
	var noBrowserOpen bool
	var enableGui bool
	var renderPath string
	var outputJson bool
	var outputYaml bool
	var summary bool
	var write bool
	var removeRelate bool
	var noUses bool
	var deleteUnused bool
	var excludeFromDelete []string
	var commentAllDelete bool

	cmd := &cobra.Command{
		Use:     "entropy",
		Short:   "(default) Renders a 3d force-directed graph in the browser",
		GroupID: renderGroupId,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := filesFromArgs(args)
			if err != nil {
				return err
			}
			cfg, err := cfgF()
			if err != nil {
				return err
			}
			lang, err := inferLang(files, cfg)
			if err != nil {
				return err
			}
			parser := language.NewParser(lang)
			applyConfigToParser(parser, cfg)

			err = entropy.Render(files, parser, entropy.RenderConfig{
				NoOpen:            noBrowserOpen,
				EnableGui:         enableGui,
				LoadCallbacks:     graph.NewStdErrCallbacks[*language.FileInfo](relPathDisplay),
				RenderPath:        renderPath,
				OutputJson:        outputJson,
				OutputYaml:        outputYaml,
				Summary:           summary,
				Write:             write,
				RemoveRelate:      removeRelate,
				NoUses:            noUses,
				DeleteUnused:      deleteUnused,
				ExcludeFromDelete: excludeFromDelete,
				CommentAllDelete:  commentAllDelete,
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&noBrowserOpen, "no-browser-open", false, "Disable the automatic browser open while rendering entropy")
	cmd.Flags().BoolVar(&enableGui, "enable-gui", false, "Enables a GUI for changing rendering settings")
	cmd.Flags().StringVar(&renderPath, "render-path", "", "Sets the output path of the rendered html file")
	cmd.Flags().BoolVar(&outputJson, "json", false, "Output the dependency graph as JSON to stdout instead of embedding it in HTML")
	cmd.Flags().BoolVar(&outputYaml, "yaml", false, "Output the dependency graph as YAML to stdout instead of embedding it in HTML")
	cmd.Flags().BoolVar(&summary, "summary", false, "When used with --json or --yaml, includes a summary of dependencies for each file")
	cmd.Flags().BoolVar(&write, "write", false, "When used with --json or --yaml and --summary, writes dependency information as comments to the target files")
	cmd.Flags().BoolVar(&removeRelate, "remove-relate", false, "Removes relate_file comments from the target files")
	cmd.Flags().BoolVar(&noUses, "no-uses", false, "When used with --write, doesn't output file_uses section in dependency comments")
	cmd.Flags().BoolVar(&deleteUnused, "delete-unused", false, "Deletes files that have 0 file_is_used_by entries (files that are not used by any other files)")
	cmd.Flags().StringArrayVar(&excludeFromDelete, "exclude-from-delete", nil, "Files matching these patterns will be included in dependency analysis but excluded from deletion by --delete-unused. You can provide multiple patterns.")
	cmd.Flags().BoolVar(&commentAllDelete, "comment-all-delete", false, "Removes all relate_file comments from all files in the project. Useful for cleaning up before commits when team approval is pending.")

	return cmd
}
