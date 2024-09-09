import dagre from "dagre"
import React, { useCallback, useEffect, useState } from "react"
import {
  Route,
  BrowserRouter as Router,
  Routes,
  useNavigate,
  useParams,
} from "react-router-dom"
import ReactFlow, {
  applyEdgeChanges,
  applyNodeChanges,
  Controls,
  Edge,
  EdgeChange,
  Node,
  NodeChange,
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
  board: Board
}

interface TreeFile {
  id: string
  name: string
}

interface Board {
  height: number
  width: number
  food: Point[]
  hazards: Point[]
  snakes: Snake[]
}

interface Point {
  x: number
  y: number
}

interface Snake {
  id: string
  name: string
  health: number
  body: Point[]
  latency: string
  head: Point
  shout: string
}

const boxWidthDefault = 300
const boxHeightDefault = 800

const dagreGraph = new dagre.graphlib.Graph()
dagreGraph.setDefaultEdgeLabel(() => ({}))

const getLayoutedElements = (
  nodes: Node[],
  edges: Edge[],
  boxHeight: number,
  boxWidth: number,
) => {
  // Set graph to left-to-right with proper spacing
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
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)

  const [boxHeight, setBoxHeight] = useState(boxHeightDefault)
  const [boxWidth, setBoxWidth] = useState(boxWidthDefault)

  // Reset nodes and edges when a new tree is selected
  useEffect(() => {
    if (id) {
      // Clear existing nodes and edges only when switching trees
      setNodes([])
      setEdges([])
      setSelectedTree(null)
      setExpandedNodes(new Set())

      // Fetch the new tree data
      setLoading(true)
      fetch(`/api/trees/${id}.json`)
        .then((res) => res.json())
        .then((data: TreeNode) => {
          setSelectedTree(data)
          setLoading(false)

          // Automatically expand the most visited path
          const { newNodes, newEdges } = autoExpandMostVisitedPath(data)
          handleLayout(newNodes, newEdges)
        })
        .catch((err) => {
          console.error("Error loading tree data", err)
          setLoading(false)
        })
    }
  }, [id])

  // Re-layout the graph when box dimensions change
  useEffect(() => {
    if (nodes.length > 0 && edges.length > 0) {
      handleLayout(nodes, edges)
    }
  }, [boxHeight, boxWidth])

  const autoExpandMostVisitedPath = (tree: TreeNode) => {
    const newNodes: Node[] = []
    const newEdges: Edge[] = []
    const newExpandedNodes = new Set<string>()

    const traverseAndExpand = (currentNode: TreeNode) => {
      newExpandedNodes.add(currentNode.id)

      newNodes.push({
        id: currentNode.id,
        data: {
          label: (
            <div>
              <pre style={{ fontFamily: "Courier New" }}>
                {currentNode.body}
              </pre>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  copyBoardState(currentNode.board)
                }}
              >
                Copy Board State
              </button>
            </div>
          ),
        },
        position: { x: 0, y: 0 }, // Initial position, will be laid out
        style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
      })

      if (currentNode.children && currentNode.children.length > 0) {
        currentNode.children.forEach((child) => {
          newNodes.push({
            id: child.id,
            data: {
              label: (
                <div>
                  <pre style={{ fontFamily: "Courier New" }}>{child.body}</pre>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      copyBoardState(child.board)
                    }}
                  >
                    Copy Board State
                  </button>
                </div>
              ),
            },
            position: { x: 0, y: 0 }, // Initial position, will be laid out
            style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
          })

          newEdges.push({
            id: `e${currentNode.id}-${child.id}`,
            source: currentNode.id,
            target: child.id,
            label: `UCB: ${child.ucb.toFixed(5)}`, // Add UCB as label on the edge
          })
        })

        const mostVisitedChild = currentNode.children.reduce(
          (maxChild, child) =>
            child.visits > maxChild.visits ? child : maxChild,
          currentNode.children[0],
        )

        traverseAndExpand(mostVisitedChild)
      }
    }

    traverseAndExpand(tree)
    setExpandedNodes(newExpandedNodes)
    return { newNodes, newEdges }
  }

  const handleNodeClick = (nodeId: string, nodeData: TreeNode) => {
    const isExpanded = expandedNodes.has(nodeId)
    const newExpandedNodes = new Set(expandedNodes)

    if (isExpanded) {
      collapseNodeAndDescendants(nodeId)
      newExpandedNodes.delete(nodeId)
    } else {
      const { newNodes, newEdges } = expandNode(nodeData)
      newExpandedNodes.add(nodeId)
      handleLayout([...nodes, ...newNodes], [...edges, ...newEdges])
    }

    setSelectedNodeId(nodeId)
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
          label: (
            <div>
              <pre style={{ fontFamily: "Courier New" }}>{child.body}</pre>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  copyBoardState(child.board)
                }}
              >
                Copy Board State
              </button>
            </div>
          ),
        },
        position: { x: 0, y: 0 }, // Default position, will be re-laid out
        style: { width: `${boxWidth}px`, height: `${boxHeight}px` },
      }
      const newEdge: Edge = {
        id: `e${parentId}-${child.id}`,
        source: parentId,
        target: child.id,
        label: `UCB: ${child.ucb.toFixed(5)}`, // Add UCB as label on the edge
      }

      newNodes.push(newNode)
      newEdges.push(newEdge)
    })

    return { newNodes, newEdges }
  }

  const handleLayout = (newNodes: Node[], newEdges: Edge[]) => {
    const updatedNodes = [...newNodes]
    const updatedEdges = [...newEdges]

    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      updatedNodes,
      updatedEdges,
      boxHeight,
      boxWidth,
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
    <div
      key={id}
      style={{ width: "80%", height: "100%", backgroundColor: "#f0f0f0" }}
    >
      <div
        style={{
          position: "absolute",
          top: 10,
          right: 10,
          zIndex: 1000, // Ensure button appears above the canvas
        }}
      >
        <button onClick={() => handleLayout(nodes, edges)}>Re-layout</button>
        <div>
          <label>Box Width:</label>
          <input
            type="number"
            value={boxWidth}
            onChange={(e) => setBoxWidth(Number(e.target.value))}
          />
        </div>
        <div>
          <label>Box Height:</label>
          <input
            type="number"
            value={boxHeight}
            onChange={(e) => setBoxHeight(Number(e.target.value))}
          />
        </div>
      </div>
      <ReactFlow
        key={id}
        nodes={nodes.map((node) => ({
          ...node,
          style: {
            ...node.style,
            backgroundColor: node.id === selectedNodeId ? "#FFD700" : "#FFF",
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
        minZoom={0.1}
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
        borderRight: "1px solid #444", // Darker border for dark mode
        backgroundColor: "#1e1e1e", // Dark background
        color: "#fff", // Light text color for visibility
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "1rem",
        }}
      >
        <h3 style={{ margin: 0, color: "#fff" }}>Available Trees</h3>
        <button
          onClick={fetchTrees}
          style={{
            marginLeft: "1rem",
            padding: "0.5rem 1rem",
            backgroundColor: "#007BFF", // Keep button color, but adjust if needed
            color: "#fff",
            border: "none",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          Refresh
        </button>
      </div>
      <div>
        {trees.map((tree) => (
          <div
            key={tree.id}
            onClick={() => navigate(`/trees/${tree.id}`)}
            style={{
              cursor: "pointer",
              padding: "1rem",
              marginBottom: "0.5rem",
              backgroundColor: "#333", // Dark card background
              borderRadius: "8px",
              boxShadow: "0 2px 4px rgba(0, 0, 0, 0.3)", // Adjusted for dark mode
              color: "#fff", // White text color
              transition: "transform 0.2s, box-shadow 0.2s",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = "scale(1.02)"
              e.currentTarget.style.boxShadow = "0 4px 8px rgba(0, 0, 0, 0.6)" // Stronger shadow in dark mode
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = "scale(1)"
              e.currentTarget.style.boxShadow = "0 2px 4px rgba(0, 0, 0, 0.3)"
            }}
          >
            {tree.name}
          </div>
        ))}
      </div>
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

const copyBoardState = (board: Board) => {
  const boardStateString = JSON.stringify(board)
  navigator.clipboard.writeText(boardStateString).then(
    () => alert("Board state copied to clipboard"),
    (err) => console.error("Failed to copy board state", err),
  )
}
