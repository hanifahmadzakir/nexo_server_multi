package main

import (
	"fmt"
	"sync"

	"github.com/hanifahmadzakir/nexo_server_multi/internal/api"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/config"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/nexo"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/state"
)

const APIPort = "8080"

func main() {
	fmt.Println("--- Rexroth Nexo Open Protocol Server Multi Client(tools) ---")
	fmt.Println("--- By Ahmadzakir Hanif (DCEA/SVE4-AS) ---")

	// 1. Load Nexo Config(IP Address)
	toolConfigs := config.GetTools()

	// 2. Initialize State Manager
	stateManager := state.NewManager()
	for _, tool := range toolConfigs {
		stateManager.InitTool(tool.ID)
	}

	// 3. Create and Run server using Goroutine
	apiServer := api.NewServer(stateManager, toolConfigs)
	go apiServer.Start(APIPort)

	// 4. Run Goroutine to keep connection alive
	var wg sync.WaitGroup
	for _, tool := range toolConfigs {
		wg.Add(1)
		go func(t config.ToolConfig, sm *state.GlobalStateManager) {
			defer wg.Done()
			nexo.ConnectAndHandleTool(t, stateManager)
		}(tool, stateManager)
	}

	// wait until all connector succeed(server will keep alive)
	wg.Wait()
}