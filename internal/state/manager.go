package state

import (
	"sync"
	"time"
)

// ToolState save last data and connection for only single tool.
type ToolState struct {
	LatestResult map[string]interface{}
	IsConnected  bool
	LastUpdate   time.Time
}

// GlobalStateManager is a concurrent-safe state manager for all tools.
type GlobalStateManager struct {
	sync.RWMutex
	Tools map[string]*ToolState
}

// NewManager is a constructor for GlobalStateManager.
func NewManager() *GlobalStateManager {
	return &GlobalStateManager{
		Tools: make(map[string]*ToolState),
	}
}

func (sm *GlobalStateManager) InitTool(toolID string) {
	sm.Lock()
	defer sm.Unlock()
	if _, exists := sm.Tools[toolID]; !exists {
		sm.Tools[toolID] = &ToolState{
			LatestResult: make(map[string]interface{}),
			IsConnected:  false,
		}
	}
}

func (sm *GlobalStateManager) SetConnectionStatus(toolID string, isConnected bool) {
	sm.Lock()
	defer sm.Unlock()
	if tool, ok := sm.Tools[toolID]; ok {
		tool.IsConnected = isConnected
	}
}

func (sm *GlobalStateManager) UpdateResult(toolID string, result map[string]interface{}) {
	sm.Lock()
	defer sm.Unlock()
	if tool, ok := sm.Tools[toolID]; ok {
		tool.LatestResult = result
		tool.LastUpdate = time.Now()
	}
}

func (sm *GlobalStateManager) GetToolState(toolID string) (*ToolState, bool) {
	sm.RLock()
	defer sm.RUnlock()
	tool, ok := sm.Tools[toolID]
	if !ok {
		return nil, false
	}
	// return a copy for concurrency safety
	stateCopy := &ToolState{
		LatestResult: tool.LatestResult,
		IsConnected:  tool.IsConnected,
		LastUpdate:   tool.LastUpdate,
	}
	return stateCopy, true
}

func (sm *GlobalStateManager) GetAllToolsStatus() map[string]*ToolState {
	sm.RLock()
	defer sm.RUnlock()
	// return a copy from the map for safety
	allTools := make(map[string]*ToolState)
	for id, state := range sm.Tools {
		allTools[id] = &ToolState{
			LatestResult: state.LatestResult,
			IsConnected:  state.IsConnected,
			LastUpdate:   state.LastUpdate,
		}
	}
	return allTools
}