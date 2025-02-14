package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/SLASH2NL/lingua"
	"github.com/SLASH2NL/lingua/extract"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "lingua",
	Short:         "A tool to extract and update translations from source code.",
	SilenceErrors: true,
}

// extractCmd scans the source code for translation keys and updates the translation files.
var extractCmd = &cobra.Command{
	Use:   "extract LANGUAGE SRC_DIR TRANSLATIONS_DIR",
	Short: "Scan the source code in SRC_DIR for translation keys and update the translation files in TRANSLATIONS_DIR.",
	Long: `Scan the source code in SRC_DIR for translation keys and update the translation files in TRANSLATIONS_DIR.

# Scan the source code dir ./src and update the translations in ./translations.
# Use --remove to remove all translations in the translation files that have not been found in the source code.
$ lingua extract en ./src ./translations --remove
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		translationDir := args[1]

		remove := cmd.Flag("remove").Value.String() == "true"

		// First read all existing translations.
		existing, err := lingua.ContainerFromFs(
			afero.NewBasePathFs(afero.NewOsFs(), translationDir),
		)
		if err != nil {
			return fmt.Errorf("error reading existing translations: %w", err)
		}

		srcMessages, err := extractMessages(dir)
		if err != nil {
			return fmt.Errorf("error extracting messages: %w", err)
		}

		// Traverse all existing translations and add new keys if they are not present.
		// If remove is set, remove all translations that are not found in the source code.
		existingMessages := existing.Raw()
		for langID, messages := range existingMessages {
			for _, key := range srcMessages {
				if _, ok := messages[key]; ok {
					continue
				}

				// Add the key as empty translation.
				existingMessages[langID][key] = ""
			}

			if remove {
				for key := range messages {
					if slices.Contains(srcMessages, key) {
						continue
					}

					// Remove the key from the translations.
					delete(existingMessages[langID], key)
				}
			}
		}

		// Traverse all existing translations and write them alphabetically sorted to the file.
		for langID, messages := range existingMessages {

			// Sort the keys and write them to a custom yaml structure to preserve the order.
			keys := make([]string, 0, len(messages))
			for k := range messages {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			root := &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
			}

			for _, k := range keys {
				keyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: k,
				}
				valueNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: messages[k],
					Style: yaml.DoubleQuotedStyle,
				}
				root.Content = append(root.Content, keyNode, valueNode)
			}

			file, err := os.OpenFile(filepath.Join(translationDir, langID.String()+".yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer file.Close()

			encoder := yaml.NewEncoder(file)
			encoder.SetIndent(2)
			if err := encoder.Encode(root); err != nil {
				return fmt.Errorf("error writing yaml: %w", err)
			}
		}

		return nil
	},
}

func init() {
	extractCmd.Flags().Bool("remove", false, "Remove all translations in the translation files that have not been found in DIR.")
	rootCmd.AddCommand(extractCmd)
}

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

func extractMessages(srcDir string) ([]string, error) {
	messages, err := extract.KeysFromSource(srcDir)
	if err != nil {
		return nil, fmt.Errorf("error reading translations from source: %w", err)
	}

	return messages, nil
}
