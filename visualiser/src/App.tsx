import React, { useState, useEffect, useCallback } from "react"
import {
  BrowserRouter as Router,
  Route,
  Routes,
  Link,
  useParams,
  useNavigate,
} from "react-router-dom"
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
import dagre from "dagre"

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
}

const boxWidth = 300
const boxHeight = 600

const dagreGraph = new dagre.graphlib.Graph()
dagreGraph.setDefaultEdgeLabel(() => ({}))

const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
  // Set graph to top-to-bottom with extra spacing
  dagreGraph.setGraph({ rankdir: "TB", ranksep: 100, nodesep: 100 })

  // Define the dimensions for each node
  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: boxWidth, height: boxHeight })
  })

  // Define edges between nodes
  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target)
  })

  // Layout the graph using dagre
  dagre.layout(dagreGraph)

  // Update node positions based on the layout
  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id)
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - boxWidth / 2,
        y: nodeWithPosition.y - boxHeight / 2,
      },
      style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
    }
  })

  return { nodes: layoutedNodes, edges }
}

const TreeViewer: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [nodes, setNodes] = useState<Node[]>([])
  const [edges, setEdges] = useState<Edge[]>([])
  const [selectedTree, setSelectedTree] = useState<TreeNode | null>(null)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null) // Track selected node

  useEffect(() => {
    if (id) {
      setLoading(true)
      fetch(`/api/trees/${id}.json`)
        .then((res) => res.json())
        .then((data) => {
          setSelectedTree(data)
          setLoading(false)
          const rootNode: Node = {
            id: data.id,
            data: {
              label: (
                <pre style={{ fontFamily: "Courier New" }}>{data.body}</pre>
              ),
            },
            position: { x: 0, y: 0 },
            style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
          }
          setNodes([rootNode])
          setEdges([])
          setExpandedNodes(new Set())
        })
        .catch((err) => {
          console.error("Error loading tree data", err)
          setLoading(false)
        })
    }
  }, [id])

  const handleNodeClick = (nodeId: string, nodeData: TreeNode) => {
    const isExpanded = expandedNodes.has(nodeId)
    const newExpandedNodes = new Set(expandedNodes)

    if (isExpanded) {
      collapseNodeAndDescendants(nodeId)
      newExpandedNodes.delete(nodeId)
    } else {
      const { newNodes, newEdges } = expandNode(nodeData)
      newExpandedNodes.add(nodeId)
      handleLayout(newNodes, newEdges) // Apply layout after expanding nodes
    }

    setSelectedNodeId(nodeId) // Set the clicked node as selected
    setExpandedNodes(newExpandedNodes)
  }

  const expandNode = (parentNode: TreeNode) => {
    if (!parentNode.children) return { newNodes: [], newEdges: [] }

    const parentId = parentNode.id
    const newNodes: Node[] = []
    const newEdges: Edge[] = []

    parentNode.children.forEach((child) => {
      const newNode: Node = {
        id: child.id,
        data: {
          label: <pre style={{ fontFamily: "Courier New" }}>{child.body}</pre>,
        },
        position: { x: 0, y: 0 }, // Default position, will be re-laid out
        style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
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

    return { newNodes, newEdges } // Return newly added nodes and edges
  }

  const handleLayout = (newNodes: Node[], newEdges: Edge[]) => {
    // Combine the existing nodes and edges with new ones
    const updatedNodes = [...nodes, ...newNodes]
    const updatedEdges = [...edges, ...newEdges]

    // Perform layout using the combined graph
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      updatedNodes,
      updatedEdges,
    )

    setNodes(layoutedNodes)
    setEdges(layoutedEdges)
  }

  const collapseNodeAndDescendants = (parentId: string) => {
    const descendants = getAllDescendants(parentId)
    setNodes((nds) => nds.filter((node) => !descendants.includes(node.id)))
    setEdges((eds) => eds.filter((edge) => !descendants.includes(edge.target)))
  }

  const getAllDescendants = (parentId: string): string[] => {
    const directChildren = edges
      .filter((edge) => edge.source === parentId)
      .map((edge) => edge.target)

    let allDescendants = [...directChildren]

    directChildren.forEach((childId) => {
      allDescendants = [...allDescendants, ...getAllDescendants(childId)]
    })

    return allDescendants
  }

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

  if (loading) {
    return <p>Loading...</p>
  }

  if (!selectedTree) {
    return <p>No tree selected</p>
  }

  return (
    <div style={{ width: "80%", height: "100%", backgroundColor: "#f0f0f0" }}>
      <div
        style={{
          position: "absolute",
          top: 10,
          right: 10,
          zIndex: 1000, // Ensure button appears above the canvas
        }}
      >
        <button onClick={() => handleLayout(nodes, edges)}>Re-layout</button>
      </div>
      <ReactFlow
        nodes={nodes.map((node) => ({
          ...node,
          style: {
            ...node.style,
            backgroundColor: node.id === selectedNodeId ? "#FFD700" : "#FFF", // Highlight selected node
          },
        }))}
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
    </div>
  )
}

const App: React.FC = () => {
  return (
    <Router>
      <div style={{ display: "flex", height: "100vh", width: "100vw" }}>
        <Sidebar />
        <Routes>
          <Route
            path="/"
            element={<p style={{ padding: "1rem" }}>Select a tree to view</p>}
          />
          <Route path="/trees/:id" element={<TreeViewer />} />
        </Routes>
      </div>
    </Router>
  )
}

// Sidebar to display the list of trees and handle navigation
const Sidebar: React.FC = () => {
  const [trees, setTrees] = useState<TreeFile[]>([])
  const navigate = useNavigate()

  const fetchTrees = () => {
    fetch("/api/trees")
      .then((res) => res.json())
      .then((data) => {
        const sortedData = data.sort((a: TreeFile, b: TreeFile) => {
          const dateA = a.name.split("_")[0] + a.name.split("_")[1]
          const dateB = b.name.split("_")[0] + b.name.split("_")[1]
          return dateB.localeCompare(dateA)
        })
        setTrees(sortedData)
      })
      .catch((err) => console.error("Error loading trees", err))
  }

  useEffect(() => {
    fetchTrees()
  }, [])

  return (
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
            onClick={() => navigate(`/trees/${tree.id}`)}
            style={{ cursor: "pointer", margin: "0.5rem 0" }}
          >
            {tree.name}
          </li>
        ))}
      </ul>
    </div>
  )
}

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
