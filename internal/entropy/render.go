package entropy

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/gabotechs/dep-tree/internal/graph"
	"github.com/gabotechs/dep-tree/internal/language"
	"github.com/gabotechs/dep-tree/internal/utils"
	"gopkg.in/yaml.v3"
)

// FileDependencies represents the dependencies of a file
type FileDependencies struct {
	FileName     string   `json:"fileName"`
	DependsOn    []string `json:"dependsOn"`
	DependedOnBy []string `json:"dependedOnBy"`
}

// GraphWithSummary extends Graph with a summary of dependencies for each file
type GraphWithSummary struct {
	Graph
	FileDependencies map[string]FileDependencies `json:"fileDependencies"`
}

//go:embed generated/index.html
var index []byte

const ToReplace = `"__INLINE_DATA",{}`
const ReplacePrefix = `"__INLINE_DATA",`

type RenderConfig struct {
	NoOpen            bool
	EnableGui         bool
	RenderPath        string
	LoadCallbacks     graph.LoadCallbacks[*language.FileInfo]
	OutputJson        bool
	OutputYaml        bool // If true, outputs the dependency graph as YAML
	Summary           bool
	Write             bool
	RemoveRelate      bool     // If true, removes relate_file comments from files
	NoUses            bool     // If true, doesn't output file_uses section in dependency comments
	DeleteUnused      bool     // If true, deletes files that have 0 file_is_used_by entries
	ExcludeFromDelete []string // Files matching these patterns will be excluded from deletion but included in dependency analysis
	CommentAllDelete  bool     // If true, removes all relate_file comments from all files
}

func Render(files []string, parser graph.NodeParser[*language.FileInfo], cfg RenderConfig) error {
	graph3d, err := makeGraph(files, parser, cfg.LoadCallbacks)
	if err != nil {
		return err
	}
	graph3d.EnableGui = cfg.EnableGui

	// Build file dependencies for summary, write, remove-relate, delete-unused, or comment-all-delete operations
	var fileDependencies map[string]FileDependencies
	if cfg.Summary || cfg.Write || cfg.RemoveRelate || cfg.DeleteUnused || cfg.CommentAllDelete || cfg.OutputJson || cfg.OutputYaml {
		// Create a map of node IDs to file names for quick lookup
		nodeIdToFileName := make(map[int64]string)
		for _, node := range graph3d.Nodes {
			nodeIdToFileName[node.Id] = node.DirName + node.FileName
		}

		// Initialize the map with all nodes
		fileDependencies = make(map[string]FileDependencies)
		for _, node := range graph3d.Nodes {
			fileName := node.DirName + node.FileName
			fileDependencies[fileName] = FileDependencies{
				FileName:     fileName,
				DependsOn:    []string{},
				DependedOnBy: []string{},
			}
		}

		// Process all links to build the dependencies
		for _, link := range graph3d.Links {
			fromFileName := nodeIdToFileName[link.From]
			toFileName := nodeIdToFileName[link.To]

			// Add to DependsOn for the source file
			fromDeps := fileDependencies[fromFileName]
			fromDeps.DependsOn = appendIfNotExists(fromDeps.DependsOn, toFileName)
			fileDependencies[fromFileName] = fromDeps

			// Add to DependedOnBy for the target file
			toDeps := fileDependencies[toFileName]
			toDeps.DependedOnBy = appendIfNotExists(toDeps.DependedOnBy, fromFileName)
			fileDependencies[toFileName] = toDeps
		}
	}

	// Handle file operations (write, remove-relate, delete-unused, comment-all-delete)
	if cfg.Write || cfg.RemoveRelate || cfg.DeleteUnused || cfg.CommentAllDelete {
		// Create a map to store file paths by file name
		filePathsByName := make(map[string]string)
		for _, node := range graph3d.Nodes {
			fileName := node.DirName + node.FileName

			// Find the absolute path for this file by reconstructing the path from pathBuf
			var matched bool
			for _, file := range files {
				// Strategy 1: Match using the actual file path from pathBuf (most accurate)
				if len(node.PathBuf) > 0 {
					actualPath := strings.Join(node.PathBuf, "/")
					if file == actualPath {
						filePathsByName[fileName] = file
						matched = true
						break
					}
				}

				// Strategy 2: Direct suffix match with combined fileName
				if strings.HasSuffix(file, fileName) {
					filePathsByName[fileName] = file
					matched = true
					break
				}
			}

			// Strategy 3: Only use base filename matching as last resort and ensure it's a unique match
			if !matched {
				var candidates []string
				for _, file := range files {
					baseFileName := filepath.Base(file)
					if len(node.PathBuf) > 0 {
						nodeBaseName := filepath.Base(strings.Join(node.PathBuf, "/"))
						if baseFileName == nodeBaseName {
							candidates = append(candidates, file)
						}
					}
				}

				// Only use this strategy if there's exactly one candidate
				if len(candidates) == 1 {
					filePathsByName[fileName] = candidates[0]
				}
			}
		}

		// Process each file based on the options
		for fileName, deps := range fileDependencies {
			if filePath, ok := filePathsByName[fileName]; ok {
				// Check if file should be deleted (has 0 file_is_used_by entries)
				if cfg.DeleteUnused && len(deps.DependedOnBy) == 0 {
					// Check if file is excluded from deletion
					if shouldExcludeFromDelete(filePath, cfg.ExcludeFromDelete) {
						// File is excluded from deletion, continue with other operations
					} else {
						// Find the corresponding node to check if it's an entry point
						var isEntrypoint bool
						for _, node := range graph3d.Nodes {
							nodeFileName := node.DirName + node.FileName
							if nodeFileName == fileName {
								isEntrypoint = node.IsEntrypoint
								break
							}
						}

						if !isEntrypoint {
							err := os.Remove(filePath)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error deleting unused file %s: %v\n", filePath, err)
							} else {
								fmt.Printf("Deleted unused file: %s\n", filePath)
							}
							continue // Skip writing dependencies to deleted files
						}
					}
				}
			}

			// Write or remove dependency information if requested
			if filePath, ok := filePathsByName[fileName]; ok {
				if cfg.Write || cfg.RemoveRelate || cfg.CommentAllDelete {
					err := writeDependenciesToFile(filePath, deps, cfg.RemoveRelate || cfg.CommentAllDelete, cfg.NoUses)
					if err != nil {
						if cfg.RemoveRelate || cfg.CommentAllDelete {
							fmt.Fprintf(os.Stderr, "Error removing relate_file comments from %s: %v\n", filePath, err)
						} else {
							fmt.Fprintf(os.Stderr, "Error writing dependencies to %s: %v\n", filePath, err)
						}
					}
				}
			}
		}
	}

	// Display summary table if requested
	if cfg.Summary {
		displaySummaryTable(fileDependencies)
		return nil
	}

	// Handle output formats
	if cfg.OutputYaml {
		var marshaled []byte
		var err error
		if fileDependencies != nil {
			// YAML output with enhanced data
			enhancedOutput := GraphWithSummary{
				Graph:            graph3d,
				FileDependencies: fileDependencies,
			}
			marshaled, err = yaml.Marshal(enhancedOutput)
		} else {
			// Standard YAML output
			marshaled, err = yaml.Marshal(graph3d)
		}
		if err != nil {
			return err
		}
		fmt.Println(string(marshaled))
		return nil
	}

	if cfg.OutputJson {
		var marshaled []byte
		var err error
		if fileDependencies != nil {
			// JSON output with enhanced data
			enhancedOutput := GraphWithSummary{
				Graph:            graph3d,
				FileDependencies: fileDependencies,
			}
			marshaled, err = json.Marshal(enhancedOutput)
		} else {
			// Standard JSON output
			marshaled, err = json.Marshal(graph3d)
		}
		if err != nil {
			return err
		}
		fmt.Println(string(marshaled))
		return nil
	}

	// For HTML output, just marshal the graph3d
	marshaled, err := json.Marshal(graph3d)
	if err != nil {
		return err
	}

	// Otherwise, proceed with the original HTML embedding logic
	rendered := bytes.ReplaceAll(index, []byte(ToReplace), append([]byte(ReplacePrefix), marshaled...))
	var temp string
	if cfg.RenderPath != "" {
		temp = cfg.RenderPath
	} else {
		temp = filepath.Join(os.TempDir(), "index.html")
	}
	err = os.WriteFile(temp, rendered, 0o600)
	if err != nil {
		return err
	}
	if cfg.NoOpen {
		fmt.Println(temp)
		return nil
	} else {
		return openInBrowser(temp)
	}
}

// displaySummaryTable displays a table of files sorted by file_is_used_by count in descending order
func displaySummaryTable(fileDependencies map[string]FileDependencies) {
	// Create a slice to hold file info for sorting
	type fileInfo struct {
		fileName       string
		usedByCount    int
		dependsOnCount int
	}

	var files []fileInfo
	for fileName, deps := range fileDependencies {
		files = append(files, fileInfo{
			fileName:       fileName,
			usedByCount:    len(deps.DependedOnBy),
			dependsOnCount: len(deps.DependsOn),
		})
	}

	// Sort by usedByCount in descending order
	sort.Slice(files, func(i, j int) bool {
		return files[i].usedByCount > files[j].usedByCount
	})

	// Create a tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(w, "\nFile Dependency Summary:")
	fmt.Fprintln(w, "File\tUsed By Count\tDepends On Count")
	fmt.Fprintln(w, "----\t-------------\t----------------")

	for _, file := range files {
		fmt.Fprintf(w, "%s\t%d\t%d\n", file.fileName, file.usedByCount, file.dependsOnCount)
	}

	w.Flush()
	fmt.Println()
}

// shouldExcludeFromDelete checks if a file path matches any of the exclude-from-delete patterns
func shouldExcludeFromDelete(filePath string, excludePatterns []string) bool {
	if len(excludePatterns) == 0 {
		return false
	}

	// Escape square brackets in the file path to treat them as literal characters
	escapedPath := escapeSquareBrackets(filePath)

	for _, pattern := range excludePatterns {
		// Do NOT escape the pattern - it should remain as a glob pattern
		if ok, _ := utils.GlobstarMatch(pattern, escapedPath); ok {
			return true
		}
	}
	return false
}

// escapeSquareBrackets escapes square brackets in a pattern to treat them as literal characters
func escapeSquareBrackets(pattern string) string {
	// Replace [ with \[ and ] with \]
	pattern = strings.ReplaceAll(pattern, "[", "\\[")
	pattern = strings.ReplaceAll(pattern, "]", "\\]")
	return pattern
}

// appendIfNotExists adds a string to a slice if it doesn't already exist
func appendIfNotExists(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// getCommentSyntax returns the comment syntax for the given file extension
func getCommentSyntax(filePath string) (string, string, string, error) {
	ext := filepath.Ext(filePath)

	switch ext {
	case ".js", ".ts", ".jsx", ".tsx":
		return "/**\n * <dependency_files>", " * </dependency_files>\n */", " *", nil
	case ".css", ".scss", ".less":
		return "/* <relate_file>", " */", " *", nil
	case ".py":
		return "# <relate_file>", "# </relate_file>", "#", nil
	case ".rb":
		return "# <relate_file>", "# </relate_file>", "#", nil
	case ".go":
		return "// <relate_file>", "// </relate_file>", "//", nil
	case ".java", ".kt", ".scala", ".c", ".cpp", ".cs", ".php", ".swift":
		return "// <relate_file>", "// </relate_file>", "//", nil
	case ".rs":
		return "// <relate_file>", "// </relate_file>", "//", nil
	case ".html", ".xml", ".svg":
		return "<!-- <relate_file> -->", "<!-- </relate_file> -->", "<!--", nil
	case ".sh", ".bash", ".zsh":
		return "# <relate_file>", "# </relate_file>", "#", nil
	case ".lua":
		return "-- <relate_file>", "-- </relate_file>", "--", nil
	case ".hs", ".lhs":
		return "-- <relate_file>", "-- </relate_file>", "--", nil
	case ".sql":
		return "-- <relate_file>", "-- </relate_file>", "--", nil
	case ".r":
		return "# <relate_file>", "# </relate_file>", "#", nil
	case ".md", ".markdown":
		return "<!-- <relate_file> -->", "<!-- </relate_file> -->", "<!--", nil
	default:
		// Default to C-style comments
		return "// <relate_file>", "// </relate_file>", "//", nil
	}
}

// writeDependenciesToFile writes or removes dependency information to/from the given file
func writeDependenciesToFile(filePath string, deps FileDependencies, removeRelate bool, noUses bool) error {
	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Get the comment syntax for this file
	startMarker, endMarker, commentPrefix, err := getCommentSyntax(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Remove any existing dependency blocks (both old and new formats)
	// Pattern for new JSDoc style comments
	newPattern := regexp.QuoteMeta(startMarker) + `[\s\S]*?` + regexp.QuoteMeta(endMarker)
	newRe := regexp.MustCompile(newPattern)

	// Pattern for old style comments
	oldPattern := `// <relate_file>[\s\S]*?// </relate_file>`
	oldRe := regexp.MustCompile(oldPattern)

	// Pattern for previous JSDoc style comments with relate_file tag
	oldJSDocPattern := `/\*\*[\s\S]*?<relate_file>[\s\S]*?\*/`
	oldJSDocRe := regexp.MustCompile(oldJSDocPattern)

	// Pattern for previous JSDoc style comments with 依存関係にあるファイル tag
	oldJapaneseJSDocPattern := `/\*\*[\s\S]*?<依存関係にあるファイル>[\s\S]*?\*/`
	oldJapaneseJSDocRe := regexp.MustCompile(oldJapaneseJSDocPattern)

	// Remove old format comments first
	contentStr = oldRe.ReplaceAllString(contentStr, "")
	contentStr = oldJSDocRe.ReplaceAllString(contentStr, "")
	contentStr = oldJapaneseJSDocRe.ReplaceAllString(contentStr, "")

	var newContent string
	if removeRelate {
		// Remove new format comments as well
		newContent = newRe.ReplaceAllString(contentStr, "")
		// Clean up any extra newlines
		newContent = strings.TrimSpace(newContent)
	} else {
		// Sort the dependencies by file name to ensure consistent ordering
		sortedDependsOn := make([]string, len(deps.DependsOn))
		copy(sortedDependsOn, deps.DependsOn)
		sort.Strings(sortedDependsOn)

		sortedDependedOnBy := make([]string, len(deps.DependedOnBy))
		copy(sortedDependedOnBy, deps.DependedOnBy)
		sort.Strings(sortedDependedOnBy)

		// Create the dependency comment block
		var dependencyBlock strings.Builder
		dependencyBlock.WriteString(startMarker + "\n")

		// Add dependencies information in YAML-like format
		if !noUses {
			dependencyBlock.WriteString(commentPrefix + " file_uses:\n")
			if len(sortedDependsOn) > 0 {
				for _, dep := range sortedDependsOn {
					dependencyBlock.WriteString(commentPrefix + " - ." + dep + "\n")
				}
			}
		}

		dependencyBlock.WriteString(commentPrefix + " this_file_is_imported_by:\n")
		if len(sortedDependedOnBy) > 0 {
			for _, dep := range sortedDependedOnBy {
				dependencyBlock.WriteString(commentPrefix + " - ." + dep + "\n")
			}
		}

		dependencyBlock.WriteString(endMarker)

		// Remove any existing new format comments and add the new block
		contentStr = newRe.ReplaceAllString(contentStr, "")

		// Clean up any extra newlines at the beginning
		contentStr = strings.TrimLeft(contentStr, "\n")

		// Add dependency block at the top of the file
		newContent = dependencyBlock.String() + "\n\n" + contentStr
	}

	// Write the modified content back to the file
	return os.WriteFile(filePath, []byte(newContent), 0644)
}
