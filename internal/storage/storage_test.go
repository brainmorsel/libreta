package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	buf := new(bytes.Buffer)
	opts := &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	log := slog.New(slog.NewTextHandler(buf, opts))

	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		t.Helper()
		t.Log("log output\n", buf.String())
	})
	return log
}

func TestStorage(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	//dataDir = "/tmp/debug-test"
	logger := testLogger(t)

	s, err := NewStorage(logger, dataDir)
	require.NoError(t, err)
	require.NoError(t, s.Open(ctx, false))
	defer func() { require.NoError(t, s.Close()) }()

	t.Run("generate_node_id", func(t *testing.T) {
		id1, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)
		id2, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)
	})

	t.Run("node_content", func(t *testing.T) {
		content := []byte(`TEST CONTENT`)
		hash1, err := s.NodeContentSave(ctx, bytes.NewReader(content))
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)

		t.Run("duplicate", func(t *testing.T) {
			hash2, err := s.NodeContentSave(ctx, bytes.NewReader(content))
			require.NoError(t, err)
			assert.Equal(t, hash1, hash2)
		})

		t.Run("load", func(t *testing.T) {
			r, err := s.NodeContentLoad(ctx, hash1)
			require.NoError(t, err)
			nContent, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, content, nContent)
		})

		t.Run("load_not_found", func(t *testing.T) {
			_, err := s.NodeContentLoad(ctx, "unknown-hash")
			assert.ErrorIs(t, err, ErrNoRecord)
		})
	})

	t.Run("node", func(t *testing.T) {
		contentHash1, err := s.NodeContentSave(ctx, bytes.NewReader([]byte(`TEST CONTENT 1`)))
		require.NoError(t, err)
		contentHash2, err := s.NodeContentSave(ctx, bytes.NewReader([]byte(`TEST CONTENT 2`)))
		require.NoError(t, err)
		nodeID, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)

		err = s.NodeSave(ctx, Node{
			ID:              nodeID,
			Name:            "test node",
			ContentHash:     contentHash1,
			ContentMimetype: "text/plain",
			Attributes: []NodeAttribute{
				{Key: "attr1", Value: "value1"},
				{Key: "attr2", Value: "value2"},
			},
		})
		require.NoError(t, err)
		nodes, err := s.NodesLoad(ctx, []string{nodeID})
		require.NoError(t, err)
		require.NotEmpty(t, nodes)
		node := nodes[nodeID]
		assert.Equal(t, nodeID, node.ID)
		assert.Equal(t, "test node", node.Name)
		assert.Equal(t, contentHash1, node.ContentHash)
		assert.Equal(t, "text/plain", node.ContentMimetype)
		assert.Equal(t, len([]byte(`TEST CONTENT 1`)), node.ContentLength)
		assert.False(t, node.IsDeleted())
		assert.Equal(t, node.CreatedAt, node.UpdatedAt)
		assert.Equal(t, 2, len(node.Attributes))

		t.Run("update", func(t *testing.T) {
			err = s.NodeSave(ctx, Node{
				ID:              nodeID,
				Name:            "edited test node",
				ContentHash:     contentHash2,
				ContentMimetype: "text/plain",
				Attributes: []NodeAttribute{
					{Key: "attr2", Value: "value2_updated"},
					{Key: "attr3", Value: "value3"},
				},
			})
			require.NoError(t, err)
			nodes, err := s.NodesLoad(ctx, []string{nodeID})
			require.NoError(t, err)
			require.NotEmpty(t, nodes)
			node := nodes[nodeID]
			assert.Equal(t, nodeID, node.ID)
			assert.Equal(t, "edited test node", node.Name)
			assert.Equal(t, contentHash2, node.ContentHash)
			assert.NotEqual(t, node.CreatedAt, node.UpdatedAt)
			assert.Equal(t, 2, len(node.Attributes))
		})

		t.Run("remove_all_attrs", func(t *testing.T) {
			err = s.NodeSave(ctx, Node{
				ID:              nodeID,
				Name:            "edited test node",
				ContentHash:     contentHash2,
				ContentMimetype: "text/plain",
				Attributes:      nil,
			})
			require.NoError(t, err)
			nodes, err := s.NodesLoad(ctx, []string{nodeID})
			require.NoError(t, err)
			require.NotEmpty(t, nodes)
			node := nodes[nodeID]
			assert.Equal(t, 0, len(node.Attributes))
		})
	})

	t.Run("edge", func(t *testing.T) {
		contentHash, err := s.NodeContentSave(ctx, bytes.NewReader([]byte(``)))
		require.NoError(t, err)
		nodeID1, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)
		nodeID2, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)
		err = s.NodeSave(ctx, Node{ID: nodeID1, Name: "node1", ContentHash: contentHash, ContentMimetype: "text/plain"})
		require.NoError(t, err)
		err = s.NodeSave(ctx, Node{ID: nodeID2, Name: "node2", ContentHash: contentHash, ContentMimetype: "text/plain"})
		require.NoError(t, err)

		err = s.EdgesAdd(ctx, []Edge{
			{SrcID: nodeID2, DstID: nodeID1, Relation: EdgeRelChild},
			{SrcID: nodeID1, DstID: nodeID2, Relation: EdgeRelLink},
		})
		require.NoError(t, err)

		edges, err := s.EdgesForNodes(ctx, []string{nodeID1, nodeID2})
		require.NoError(t, err)
		assert.Equal(t, 2, len(edges))

		t.Run("remove", func(t *testing.T) {
			err := s.EdgesRemove(ctx, []Edge{
				{SrcID: nodeID1, DstID: nodeID2, Relation: EdgeRelLink},
				{SrcID: nodeID1, DstID: "non-existed-node", Relation: EdgeRelLink},
			})
			require.NoError(t, err)
			edges, err := s.EdgesForNodes(ctx, []string{nodeID1, nodeID2})
			require.NoError(t, err)
			assert.Equal(t, 1, len(edges))
		})
	})

	t.Run("query", func(t *testing.T) {
		emptyContentHash, err := s.NodeContentSave(ctx, bytes.NewReader([]byte(``)))
		require.NoError(t, err)
		nodeID1, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)
		nodeID2, err := s.GenerateNodeID(ctx)
		require.NoError(t, err)

		t.Run("by_attribute", func(t *testing.T) {
			err = s.NodeSave(ctx, Node{
				ID:              nodeID1,
				Name:            "root1",
				ContentHash:     emptyContentHash,
				ContentMimetype: "text/plain",
				Attributes: []NodeAttribute{
					{Key: NodeAttrKind, Value: NodeAttrKindRoot},
				},
			})
			require.NoError(t, err)
			err = s.NodeSave(ctx, Node{
				ID:              nodeID2,
				Name:            "root2",
				ContentHash:     emptyContentHash,
				ContentMimetype: "text/plain",
				Attributes: []NodeAttribute{
					{Key: NodeAttrKind, Value: NodeAttrKindRoot},
				},
			})
			require.NoError(t, err)

			nodeIDs, err := s.QueryByAttribute(ctx, NodeAttrKind, NodeAttrKindRoot)
			require.NoError(t, err)
			assert.Equal(t, 2, len(nodeIDs))
		})

		t.Run("full_text_search", func(t *testing.T) {
			hash, err := s.NodeContentSave(ctx, bytes.NewReader([]byte(`xxxxxx FTS FOUND1 xxxxxx`)))
			require.NoError(t, err)

			err = s.NodeSave(ctx, Node{
				ID:              nodeID1,
				Name:            "xxxx FTS FOUND2 xxxx",
				ContentHash:     hash,
				ContentMimetype: "text/plain",
			})
			require.NoError(t, err)
			err = s.NodeSave(ctx, Node{
				ID:              nodeID2,
				Name:            "xxxx FTS FOUND3 xxxx",
				ContentHash:     emptyContentHash,
				ContentMimetype: "text/plain",
			})
			require.NoError(t, err)

			nodeIDs, err := s.QueryFullTextSearch(ctx, "FTS FOUND1", 10)
			require.NoError(t, err)
			assert.Equal(t, []string{nodeID1}, nodeIDs)

			nodeIDs, err = s.QueryFullTextSearch(ctx, "FTS FOUND2", 10)
			require.NoError(t, err)
			assert.Equal(t, []string{nodeID1}, nodeIDs)

			nodeIDs, err = s.QueryFullTextSearch(ctx, "FTS FOUND3", 10)
			require.NoError(t, err)
			assert.Equal(t, []string{nodeID2}, nodeIDs)

			nodeIDs, err = s.QueryFullTextSearch(ctx, "FTS FOUND", 10)
			require.NoError(t, err)
			assert.Equal(t, []string{nodeID1, nodeID2}, nodeIDs)
		})
	})
}

func TestXXX(t *testing.T) {
	assert.True(t, isTextMimetype("text/plain"))
}
