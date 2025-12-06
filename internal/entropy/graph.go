package entropy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabotechs/dep-tree/internal/graph"
	"github.com/gabotechs/dep-tree/internal/language"
	"github.com/gabotechs/dep-tree/internal/utils"
)

const (
	maxNodeSize = 10
)

// isLikelyEntrypoint determines if a file is likely an entrypoint based on its name and path
func isLikelyEntrypoint(filename, absPath string) bool {
	// Next.js API routes (route.ts, route.js)
	if filename == "route.ts" || filename == "route.js" {
		return true
	}

	// Next.js pages (page.tsx, page.ts, page.jsx, page.js)
	if filename == "page.tsx" || filename == "page.ts" || filename == "page.jsx" || filename == "page.js" {
		return true
	}

	// Next.js layout files
	if filename == "layout.tsx" || filename == "layout.ts" || filename == "layout.jsx" || filename == "layout.js" {
		return true
	}

	// Next.js middleware
	if filename == "middleware.ts" || filename == "middleware.js" {
		return true
	}

	// Common entry point patterns - but only at root level or specific directories
	if filename == "index.ts" || filename == "index.js" || filename == "index.tsx" || filename == "index.jsx" {
		// Only consider index files as entrypoints if they are:
		// 1. At the root level (src/index.ts)
		// 2. In main directories like src/, app/, pages/
		// 3. NOT in subdirectories like hooks/, components/, utils/, etc.

		// Get the directory path relative to the project root
		dir := filepath.Dir(absPath)

		// Check if it's in a subdirectory that shouldn't be considered an entrypoint
		lowerDir := strings.ToLower(dir)
		excludedDirs := []string{"hooks", "components", "utils", "helpers", "lib", "types", "models", "schemas", "constants"}

		for _, excluded := range excludedDirs {
			if strings.Contains(lowerDir, "/"+excluded+"/") || strings.HasSuffix(lowerDir, "/"+excluded) {
				return false
			}
		}

		// Only consider as entrypoint if it's in a main directory or at root level
		return strings.Contains(lowerDir, "/src/") || strings.Contains(lowerDir, "/app/") || strings.Contains(lowerDir, "/pages/") ||
			strings.HasSuffix(lowerDir, "/src") || strings.HasSuffix(lowerDir, "/app") || strings.HasSuffix(lowerDir, "/pages")
	}

	// Main application files
	if filename == "main.ts" || filename == "main.js" || filename == "app.ts" || filename == "app.js" {
		return true
	}

	return false
}

type Node struct {
	Id           int64    `json:"id"`
	IsEntrypoint bool     `json:"isEntrypoint"`
	FileName     string   `json:"fileName"`
	PathBuf      []string `json:"pathBuf"`
	Group        string   `json:"group,omitempty"`
	DirName      string   `json:"dirName"`
	Loc          int      `json:"loc"`
	Size         int      `json:"size"`
}

type Link struct {
	From     int64 `json:"from"`
	To       int64 `json:"to"`
	IsCyclic bool  `json:"isCyclic"`
}

type Graph struct {
	Nodes     []Node `json:"nodes"`
	Links     []Link `json:"links"`
	EnableGui bool   `json:"enableGui"`
}

func makeGraph(files []string, parser graph.NodeParser[*language.FileInfo], loadCallbacks graph.LoadCallbacks[*language.FileInfo]) (Graph, error) {
	g := graph.NewGraph[*language.FileInfo]()
	err := g.Load(files, parser, loadCallbacks)
	if err != nil {
		return Graph{}, err
	}
	var singleEntrypointAbsPath string
	var entrypoints []*graph.Node[*language.FileInfo]
	if len(files) == 1 {
		entrypoint := g.Get(files[0])
		if entrypoint == nil {
			return Graph{}, fmt.Errorf("could not find entrypoint %s", files[0])
		}
		entrypoints = append(entrypoints, entrypoint)
		singleEntrypointAbsPath = entrypoint.Data.AbsPath
	} else {
		entrypoints = g.GetNodesWithoutParents()
	}

	cycles := g.RemoveCycles(entrypoints)
	out := Graph{
		Nodes: make([]Node, 0),
		Links: make([]Link, 0),
	}

	allNodes := g.AllNodes()
	maxLoc := max(utils.Max(allNodes, func(n *graph.Node[*language.FileInfo]) int {
		return n.Data.Loc
	}), 1)

	for _, node := range allNodes {
		cwd, err := os.Getwd()
		if err != nil {
			return Graph{}, err
		}

		filename := filepath.Base(node.Data.RelPath)

		// If filename is "." (empty RelPath), use AbsPath instead
		if filename == "." {
			filename = filepath.Base(node.Data.AbsPath)
		}

		dirNameWithFile := strings.Replace(node.Data.AbsPath, cwd, "", 1)
		// Use filepath operations instead of string replacement to handle special characters correctly
		dirName := filepath.Dir(dirNameWithFile)
		if dirName != "." {
			dirName = dirName + string(os.PathSeparator)
		} else {
			dirName = ""
		}

		// Determine if this node is an entrypoint
		isEntrypoint := node.Data.AbsPath == singleEntrypointAbsPath

		// For multiple files (glob patterns), also consider certain file patterns as entrypoints
		if !isEntrypoint && len(files) > 1 {
			isEntrypoint = isLikelyEntrypoint(filename, node.Data.AbsPath)
		}

		n := Node{
			Id:           node.ID(),
			IsEntrypoint: isEntrypoint,
			FileName:     filename,
			PathBuf:      strings.Split(node.Data.AbsPath, string(os.PathSeparator)),
			Group:        node.Data.Package,
			DirName:      dirName,
			Loc:          node.Data.Loc,
			Size:         maxNodeSize * node.Data.Loc / maxLoc,
		}
		out.Nodes = append(out.Nodes, n)

		for _, to := range g.FromId(node.Id) {
			out.Links = append(out.Links, Link{
				From: node.ID(),
				To:   to.ID(),
			})
		}
	}

	for _, cycle := range cycles {
		out.Links = append(out.Links, Link{
			From:     graph.MakeNode(cycle.Cause[0], 0).ID(),
			To:       graph.MakeNode(cycle.Cause[1], 0).ID(),
			IsCyclic: true,
		})
	}

	return out, nil
}
