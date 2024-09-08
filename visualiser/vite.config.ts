import { defineConfig, Plugin } from "vite"
import react from "@vitejs/plugin-react"
import fs from "fs"
import path from "path"

export default defineConfig({
  plugins: [
    react(),
    {
      name: "tree-data-api", // Custom plugin for serving the tree data
      configureServer(server) {
        // Custom middleware to serve the JSON tree data
        server.middlewares.use("/api/trees", (req, res) => {
          const dataDir = path.join(__dirname, "tree-data")

          // Read the JSON files from the tree-data folder
          const files = fs.readdirSync(dataDir)
          const treeData = files.map((file) => ({
            id: file,
            name: path.basename(file, ".json"),
            data: JSON.parse(
              fs.readFileSync(path.join(dataDir, file), "utf-8"),
            ),
          }))

          // Send the JSON response
          res.setHeader("Content-Type", "application/json")
          res.end(JSON.stringify(treeData))
        })
      },
    } as Plugin,
  ],
  server: {
    port: 3000,
  },
})
