package graph

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"dep-tree/internal/graph/node"
)

const RebuildTestsEnv = "REBUILD_TESTS"

type TestGraph struct {
	Spec [][]int
}

var _ NodeParser[[]int] = &TestGraph{}

func (t *TestGraph) Parse(id string) (*node.Node[[]int], error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	var children []int
	if idInt >= len(t.Spec) {
		return nil, fmt.Errorf("%s not present in spec", id)
	} else {
		children = t.Spec[idInt]
	}
	return node.MakeNode(id, id, children), nil
}

func (t *TestGraph) Deps(n *node.Node[[]int]) []string {
	result := make([]string, 0)
	for _, child := range n.Data {
		result = append(result, strconv.Itoa(child))
	}
	return result
}

func (t *TestGraph) Display(n *node.Node[[]int]) string {
	return n.Id
}

const testDir = "graph_test"

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestMakeGraph(t *testing.T) {
	tests := []struct {
		Name string
		Spec [][]int
	}{
		{
			Name: "Simple",
			Spec: [][]int{
				{1, 2, 3},
				{2, 4},
				{3, 4},
				{4},
				{3},
			},
		},
		{
			Name: "Two in the same level",
			Spec: [][]int{
				{1, 2, 3},
				{3},
				{3},
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			a := require.New(t)
			testParser := TestGraph{
				Spec: tt.Spec,
			}

			result, err := RenderGraph[[]int]("0", &testParser)
			a.NoError(err)
			print(result)

			outFile := path.Join(testDir, path.Base(tt.Name+".txt"))
			if fileExists(outFile) && os.Getenv(RebuildTestsEnv) != "true" {
				expected, err := os.ReadFile(outFile)
				a.NoError(err)
				a.Equal(string(expected), result)
			} else {
				_ = os.Mkdir(testDir, os.ModePerm)
				err := os.WriteFile(outFile, []byte(result), os.ModePerm)
				a.NoError(err)
			}
		})
	}
}
