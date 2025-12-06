/**
 * Tree Mapper Data Conversion Utilities
 * Converts dependency tree data to GoJS Tree Mapper format
 */

class TreeMapperDataConverter {
  constructor() {
    this.nodeIdCounter = 0;
    this.leftGroupId = -1;
    this.rightGroupId = -2;
  }

  /**
   * Convert dependency graph data to tree mapper format
   * @param {Object} graphData - The dependency graph data
   * @param {Object} options - Conversion options
   * @returns {Object} - GoJS compatible data structure
   */
  convertDependencyData(graphData, options = {}) {
    const {
      leftTitle = 'Source Dependencies',
      rightTitle = 'Target Dependencies',
      groupSpacing = 400,
      leftWidth = 250,
      rightWidth = 250,
      maxDepth = 3
    } = options;

    // Initialize groups
    const nodeDataArray = [
      {
        isGroup: true,
        key: this.leftGroupId,
        text: leftTitle,
        xy: '0 0',
        width: leftWidth
      },
      {
        isGroup: true,
        key: this.rightGroupId,
        text: rightTitle,
        xy: `${groupSpacing} 0`,
        width: rightWidth
      }
    ];

    const linkDataArray = [];
    const processedNodes = new Set();

    // Process nodes from graph data
    if (graphData.nodes) {
      this.processNodes(graphData.nodes, nodeDataArray, processedNodes, maxDepth);
    }

    // Process edges/links
    if (graphData.edges) {
      this.processEdges(graphData.edges, linkDataArray, processedNodes);
    }

    // Add some sample mapping links for demonstration
    this.addSampleMappings(nodeDataArray, linkDataArray);

    return { nodeDataArray, linkDataArray };
  }

  /**
   * Process nodes from dependency graph
   */
  processNodes(nodes, nodeDataArray, processedNodes, maxDepth) {
    // Separate nodes into two groups based on some criteria
    const leftNodes = [];
    const rightNodes = [];

    nodes.forEach(node => {
      // Example logic: split by package type or depth
      if (node.type === 'internal' || node.depth <= 2) {
        leftNodes.push(node);
      } else {
        rightNodes.push(node);
      }
    });

    // Process left side nodes
    this.processNodeGroup(leftNodes, nodeDataArray, this.leftGroupId, processedNodes, maxDepth);

    // Process right side nodes
    this.processNodeGroup(rightNodes, nodeDataArray, this.rightGroupId, processedNodes, maxDepth);
  }

  /**
   * Process a group of nodes
   */
  processNodeGroup(nodes, nodeDataArray, groupId, processedNodes, maxDepth) {
    if (nodes.length === 0) return;

    // Create root node for this group
    const rootKey = groupId === this.leftGroupId ? 0 : 1000;
    const rootNode = {
      key: rootKey,
      group: groupId,
      name: groupId === this.leftGroupId ? 'Internal Modules' : 'External Dependencies',
      isRoot: true
    };

    nodeDataArray.push(rootNode);
    processedNodes.add(rootKey);

    // Add child nodes
    nodes.forEach((node, index) => {
      if (processedNodes.has(node.id)) return;

      const nodeKey = rootKey + index + 1;
      const convertedNode = {
        key: nodeKey,
        group: groupId,
        name: this.getNodeDisplayName(node),
        originalId: node.id,
        type: node.type,
        size: node.size || 1,
        parent: rootKey
      };

      nodeDataArray.push(convertedNode);
      processedNodes.add(nodeKey);
    });
  }

  /**
   * Process edges/dependencies
   */
  processEdges(edges, linkDataArray, processedNodes) {
    edges.forEach(edge => {
      // Convert edge IDs to node keys if needed
      const fromKey = this.findNodeKey(edge.from || edge.source);
      const toKey = this.findNodeKey(edge.to || edge.target);

      if (fromKey && toKey && processedNodes.has(fromKey) && processedNodes.has(toKey)) {
        linkDataArray.push({
          from: fromKey,
          to: toKey,
          weight: edge.weight || 1
        });
      }
    });
  }

  /**
   * Add sample mapping links between left and right groups
   */
  addSampleMappings(nodeDataArray, linkDataArray) {
    const leftNodes = nodeDataArray.filter(n => n.group === this.leftGroupId && !n.isRoot);
    const rightNodes = nodeDataArray.filter(n => n.group === this.rightGroupId && !n.isRoot);

    // Create some sample mappings
    const maxMappings = Math.min(5, leftNodes.length, rightNodes.length);
    for (let i = 0; i < maxMappings; i++) {
      if (leftNodes[i] && rightNodes[i]) {
        linkDataArray.push({
          from: leftNodes[i].key,
          to: rightNodes[i].key,
          category: 'Mapping'
        });
      }
    }
  }

  /**
   * Get display name for a node
   */
  getNodeDisplayName(node) {
    if (node.name) return node.name;
    if (node.label) return node.label;
    if (node.id) {
      // Clean up file paths for display
      const parts = node.id.split('/');
      return parts[parts.length - 1] || node.id;
    }
    return `Node ${this.nodeIdCounter++}`;
  }

  /**
   * Find node key by original ID
   */
  findNodeKey(originalId) {
    // This would need to be implemented based on your ID mapping strategy
    return originalId;
  }

  /**
   * Convert file tree data to tree mapper format
   */
  convertFileTreeData(fileTree, options = {}) {
    const {
      leftTitle = 'Source Files',
      rightTitle = 'Dependencies',
      splitCriteria = 'type' // 'type', 'depth', 'size'
    } = options;

    const nodeDataArray = [
      { isGroup: true, key: this.leftGroupId, text: leftTitle, xy: '0 0', width: 250 },
      { isGroup: true, key: this.rightGroupId, text: rightTitle, xy: '400 0', width: 250 }
    ];

    const linkDataArray = [];

    // Process file tree recursively
    this.processFileTree(fileTree, nodeDataArray, linkDataArray, splitCriteria);

    return { nodeDataArray, linkDataArray };
  }

  /**
   * Process file tree structure
   */
  processFileTree(tree, nodeDataArray, linkDataArray, splitCriteria) {
    // Implementation for file tree processing
    // This would depend on your specific file tree structure

    if (Array.isArray(tree)) {
      tree.forEach(item => this.processFileTreeItem(item, nodeDataArray, linkDataArray, splitCriteria));
    } else if (tree.children) {
      tree.children.forEach(item => this.processFileTreeItem(item, nodeDataArray, linkDataArray, splitCriteria));
    }
  }

  /**
   * Process individual file tree item
   */
  processFileTreeItem(item, nodeDataArray, linkDataArray, splitCriteria) {
    const groupId = this.determineGroup(item, splitCriteria);
    const nodeKey = this.nodeIdCounter++;

    const node = {
      key: nodeKey,
      group: groupId,
      name: item.name || item.path,
      type: item.type,
      size: item.size,
      isDirectory: item.isDirectory
    };

    nodeDataArray.push(node);

    // Process children if any
    if (item.children) {
      item.children.forEach(child => {
        this.processFileTreeItem(child, nodeDataArray, linkDataArray, splitCriteria);
        // Add parent-child link
        linkDataArray.push({
          from: nodeKey,
          to: this.nodeIdCounter - 1
        });
      });
    }
  }

  /**
   * Determine which group a node belongs to
   */
  determineGroup(item, criteria) {
    switch (criteria) {
      case 'type':
        return item.type === 'internal' ? this.leftGroupId : this.rightGroupId;
      case 'depth':
        return (item.depth || 0) <= 2 ? this.leftGroupId : this.rightGroupId;
      case 'size':
        return (item.size || 0) > 1000 ? this.leftGroupId : this.rightGroupId;
      default:
        return this.leftGroupId;
    }
  }

  /**
   * Create sample data for testing
   */
  createSampleData() {
    return {
      nodeDataArray: [
        { isGroup: true, key: -1, text: 'Source Modules', xy: '0 0', width: 200 },
        { isGroup: true, key: -2, text: 'External Dependencies', xy: '400 0', width: 200 },

        // Left side nodes
        { key: 0, group: -1, name: 'main.go' },
        { key: 1, group: -1, name: 'config.go', parent: 0 },
        { key: 2, group: -1, name: 'utils.go', parent: 0 },
        { key: 3, group: -1, name: 'handlers.go', parent: 1 },

        // Right side nodes
        { key: 1000, group: -2, name: 'external libs' },
        { key: 1001, group: -2, name: 'github.com/gin', parent: 1000 },
        { key: 1002, group: -2, name: 'github.com/viper', parent: 1000 },
        { key: 1003, group: -2, name: 'database/sql', parent: 1001 }
      ],
      linkDataArray: [
        // Tree structure links
        { from: 0, to: 1 },
        { from: 0, to: 2 },
        { from: 1, to: 3 },
        { from: 1000, to: 1001 },
        { from: 1000, to: 1002 },
        { from: 1001, to: 1003 },

        // Mapping links
        { from: 1, to: 1002, category: 'Mapping' },
        { from: 3, to: 1001, category: 'Mapping' },
        { from: 2, to: 1003, category: 'Mapping' }
      ]
    };
  }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
  module.exports = TreeMapperDataConverter;
} else {
  window.TreeMapperDataConverter = TreeMapperDataConverter;
}