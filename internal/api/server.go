package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/config"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/state"
)

// Server struct for managing dependencies.
type Server struct {
	stateManager *state.GlobalStateManager
	toolConfigs  []config.ToolConfig
	app          *fiber.App
}

// NewServer create new instance of server API.
func NewServer(sm *state.GlobalStateManager, tools []config.ToolConfig) *Server {
	app := fiber.New(fiber.Config{ServerHeader: "Nexo-OP-Fiber"})
	server := &Server{
		stateManager: sm,
		toolConfigs:  tools,
		app:          app,
	}
	server.setupRoutes()
	return server
}

// setupRoutes register all API routes.
func (s *Server) setupRoutes() {
	s.app.Get("/tools", s.handleGetAllTools)
	s.app.Get("/tools/:id", s.handleGetToolResult)
}

// Start server HTTP.
func (s *Server) Start(port string) {
	fmt.Printf("API Server: Starting HTTP Server on port %s...\n", port)
	if err := s.app.Listen(":" + port); err != nil {
		fmt.Printf("API Server Error: %v\n", err)
	}
}

// handleGetAllTools handle API requests for the status of all tools.
func (s *Server) handleGetAllTools(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"tools":     s.stateManager.GetAllToolsStatus(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleGetToolResult handle API requests for the latest result of a specific tool.
func (s *Server) handleGetToolResult(c *fiber.Ctx) error {
	toolID := c.Params("id")
	state, found := s.stateManager.GetToolState(toolID)
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "error",
			"message": fmt.Sprintf("Tool with id '%s' not found or not configured.", toolID),
		})
	}

	response := fiber.Map{
		"status":      "ok",
		"tool_id":     toolID,
		"connected":   state.IsConnected,
		"last_result": state.LatestResult,
		"last_update": state.LastUpdate.Format(time.RFC3339),
	}
	
	if len(state.LatestResult) == 0 {
		response["status"] = "no data, waiting for first tightening result"
		return c.Status(fiber.StatusNoContent).JSON(response)
	}

	return c.JSON(response)
}