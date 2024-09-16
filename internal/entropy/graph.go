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

type Node struct {
	Id           int64    `json:"id"`
	IsEntrypoint bool     `json:"isEntrypoint"`
	FileName     string   `json:"fileName"`
	PathBuf      []string `json:"pathBuf"`
	Group        string   `json:"group,omitempty"`
	DirName      string   `json:"dirName"`
	Loc          int      `json:"loc"`
	Size         int      `json:"size"`
	Position     Position `json:"position"`
	IsDirectory  bool     `json:"isDirectory"`
	ClassName    string   `json:"className,omitempty"`
	Type         string   `json:"type,omitempty"`
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Link struct {
	From     int64 `json:"source"`
	To       int64 `json:"target"`
	IsCyclic bool  `json:"isCyclic"`
}

type Graph struct {
	Nodes     []Node `json:"nodes"`
	Links     []Link `json:"edges"`
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

	dirNodes := make(map[string]int64)

	for _, node := range allNodes {
		dirName := filepath.Dir(node.Data.RelPath)
		isDirectory := !strings.Contains(filepath.Base(node.Data.RelPath), ".")
		if _, exists := dirNodes[dirName]; !exists {
			dirNode := Node{
				Id:          int64(len(out.Nodes) + 1),
				FileName:    dirName,
				DirName:     filepath.Dir(dirName) + string(os.PathSeparator),
				Position:    Position{X: 0, Y: 0},
				PathBuf:     strings.Split(filepath.Join(node.Data.AbsPath, ".."), string(os.PathSeparator)),
				Group:       node.Data.Package,
				IsDirectory: true,
				ClassName:   ".nodeDir",
				Type:        "directory",
			}
			out.Nodes = append(out.Nodes, dirNode)
			dirNodes[dirName] = dirNode.Id
		}

		n := Node{
			Id:           node.ID(),
			IsEntrypoint: node.Data.AbsPath == singleEntrypointAbsPath,
			FileName:     filepath.Base(node.Data.RelPath),
			PathBuf:      strings.Split(node.Data.AbsPath, string(os.PathSeparator)),
			Group:        node.Data.Package,
			DirName:      dirName + string(os.PathSeparator),
			Loc:          node.Data.Loc,
			Size:         maxNodeSize * node.Data.Loc / maxLoc,
			IsDirectory:  isDirectory,
			Position:     Position{X: 0, Y: 0},
			ClassName:    ".node",
			Type:         "file",
		}
		out.Nodes = append(out.Nodes, n)

		out.Links = append(out.Links, Link{
			From: dirNodes[dirName],
			To:   n.Id,
		})

		for _, to := range g.FromId(node.Id) {
			out.Links = append(out.Links, Link{
				From: n.Id,
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
