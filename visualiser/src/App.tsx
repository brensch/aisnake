import React, { useState, useEffect, useCallback } from "react"
import ReactFlow, {
  Node,
  Edge,
  Controls,
  applyNodeChanges,
  applyEdgeChanges,
  NodeChange,
  EdgeChange,
} from "reactflow"
import "reactflow/dist/style.css"

interface TreeNode {
  id: string
  body: string
  visits: number
  avg_score: number
  ucb: number
  isMostVisited: boolean
  children?: TreeNode[]
}

interface TreeFile {
  id: string
  name: string
  data: TreeNode
}

const boxWidth = 300
const boxHeight = 600

const App: React.FC = () => {
  const [trees, setTrees] = useState<TreeFile[]>([])
  const [nodes, setNodes] = useState<Node[]>([])
  const [edges, setEdges] = useState<Edge[]>([])
  const [selectedTree, setSelectedTree] = useState<TreeNode | null>(null)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())

  // Function to fetch the tree data and sort it by date
  const fetchTrees = () => {
    fetch("/api/trees")
      .then((res) => res.json())
      .then((data) => {
        const sortedData = data.sort((a: TreeFile, b: TreeFile) => {
          // Extract the full date and time prefix (YYYYMMDD_HHMMSS.xxx)
          const dateA = a.name.split("_")[0] + a.name.split("_")[1]
          const dateB = b.name.split("_")[0] + b.name.split("_")[1]
          return dateB.localeCompare(dateA) // Sort in descending order (newest first)
        })
        setTrees(sortedData)
      })
      .catch((err) => console.error("Error loading trees", err))
  }

  // Fetch tree data on component mount
  useEffect(() => {
    fetchTrees()
  }, [])

  // Generate the root node for the selected tree
  const handleTreeClick = (tree: TreeNode) => {
    setSelectedTree(tree)
    setNodes([
      {
        id: tree.id,
        data: {
          label: <pre style={{ fontFamily: "Courier New" }}>{tree.body}</pre>,
        },
        position: { x: window.innerWidth / 2 - 150, y: 50 },
      },
    ])
    setEdges([])
    setExpandedNodes(new Set())
  }

  // Toggle node children (expand or collapse)
  const handleNodeClick = (nodeId: string, nodeData: TreeNode) => {
    const isExpanded = expandedNodes.has(nodeId)
    const newExpandedNodes = new Set(expandedNodes)

    if (isExpanded) {
      collapseNode(nodeId)
      newExpandedNodes.delete(nodeId)
    } else {
      expandNode(nodeData)
      newExpandedNodes.add(nodeId)
    }

    setExpandedNodes(newExpandedNodes)
  }

  const expandNode = (parentNode: TreeNode) => {
    if (!parentNode.children) return

    const parentId = parentNode.id
    const parentPosition = nodes.find((n) => n.id === parentId)?.position || {
      x: 0,
      y: 0,
    }
    const newNodes: Node[] = []
    const newEdges: Edge[] = []

    parentNode.children.forEach((child, index) => {
      const element = document.getElementById(parentId)
      const parentHeight = element ? element.offsetHeight : boxHeight

      const newNode: Node = {
        id: child.id,
        data: {
          label: <div style={{ fontFamily: "Courier New" }}>{child.body}</div>,
        },
        position: {
          x:
            parentPosition.x +
            (index - parentNode.children!.length / 2) * boxWidth,
          y: parentPosition.y + parentHeight + 50,
        },
      }
      const newEdge: Edge = {
        id: `e${parentId}-${child.id}`,
        source: parentId,
        target: child.id,
      }

      newNodes.push(newNode)
      newEdges.push(newEdge)
    })

    setNodes((nds) => [...nds, ...newNodes])
    setEdges((eds) => [...eds, ...newEdges])
  }

  const collapseNode = (parentId: string) => {
    const childIds = edges
      .filter((edge) => edge.source === parentId)
      .map((edge) => edge.target)
    setNodes((nds) => nds.filter((node) => !childIds.includes(node.id)))
    setEdges((eds) => eds.filter((edge) => !childIds.includes(edge.target)))
  }

  // Update nodes and edges when changes occur
  const onNodesChange = useCallback(
    (changes: NodeChange[]) =>
      setNodes((nds) => applyNodeChanges(changes, nds)),
    [],
  )
  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) =>
      setEdges((eds) => applyEdgeChanges(changes, eds)),
    [],
  )

  return (
    <div style={{ display: "flex", height: "100vh", width: "100vw" }}>
      {/* Sidebar to display the list of trees */}
      <div
        style={{
          width: "20%",
          overflowY: "scroll",
          padding: "1rem",
          borderRight: "1px solid #ccc",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <h3>Available Trees</h3>
          <button onClick={fetchTrees} style={{ marginLeft: "1rem" }}>
            Refresh
          </button>
        </div>
        <ul>
          {trees.map((tree) => (
            <li
              key={tree.id}
              onClick={() => handleTreeClick(tree.data)}
              style={{ cursor: "pointer", margin: "0.5rem 0" }}
            >
              {tree.name}
            </li>
          ))}
        </ul>
      </div>

      {/* React Flow graph for tree visualization */}
      <div style={{ width: "80%", height: "100%", backgroundColor: "#f0f0f0" }}>
        {selectedTree ? (
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={(event, node) => {
              const treeNode = findTreeNode(selectedTree, node.id)
              if (treeNode) {
                handleNodeClick(node.id, treeNode)
              }
            }}
            fitView
            fitViewOptions={{ padding: 0.2 }}
          >
            <Controls />
          </ReactFlow>
        ) : (
          <p style={{ padding: "1rem" }}>Select a tree to view</p>
        )}
      </div>
    </div>
  )
}

// Helper function to find a node in the tree by id
const findTreeNode = (tree: TreeNode, id: string): TreeNode | null => {
  if (tree.id === id) return tree
  if (!tree.children) return null
  for (const child of tree.children) {
    const found = findTreeNode(child, id)
    if (found) return found
  }
  return null
}

export default App
