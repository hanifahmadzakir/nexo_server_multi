package nexo

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hanifahmadzakir/nexo_server_multi/internal/config"
	"github.com/hanifahmadzakir/nexo_server_multi/internal/state"
)

const (
	nulByteFromProtocol = "\x00"
)

// HandleConnection managing life cycle of TCP connection.
func HandleConnection(conn net.Conn, toolID string, sm *state.GlobalStateManager) {
	defer conn.Close()
	defer func() {
		sm.SetConnectionStatus(toolID, false)
		fmt.Printf("TCP Client [%s]: Connection closed.\n", toolID)
	}()

	fmt.Printf("TCP Client [%s]: Successfully connected to %s. Starting communication setup...\n", toolID, conn.RemoteAddr())
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 1. send MID 0001 and receive MID 0002
	conn.Write([]byte(BuildMID0001()))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString(nulByteFromProtocol[0])
	if err != nil || !strings.Contains(response, "0002") {
		fmt.Printf("TCP Client [%s]: Failed to read/MID 0002 not received: %v\n", toolID, err)
		return
	}
	fmt.Printf("TCP Client [%s]: <- MID 0002 received. Communication Established.\n", toolID)

	// 2. send MID 0060 and receive MID 0005
	conn.Write([]byte(BuildMID0060()))
	response, err = reader.ReadString(nulByteFromProtocol[0])
	if err != nil || !strings.Contains(response, "0005") {
		fmt.Printf("TCP Client [%s]: Failed to read/MID 0005 for MID 0060 not received: %v\n", toolID, err)
		return
	}
	fmt.Printf("TCP Client [%s]: <- MID 0005 received. Listening for results.\n", toolID)

	conn.SetReadDeadline(time.Time{})
	sm.SetConnectionStatus(toolID, true)

	// Goroutine Keep Alive per connection
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if state, ok := sm.GetToolState(toolID); !ok || !state.IsConnected {
				return // stop if the connection is lost
			}
			if _, err := conn.Write([]byte(BuildMID9999())); err != nil {
				fmt.Printf("TCP Client [%s]: Error sending Keep Alive: %v. Stopping.\n", toolID, err)
				return
			}
		}
	}()

	// Main loop to read data
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		payload, err := reader.ReadString(nulByteFromProtocol[0])
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("TCP Client [%s]: Read timeout, connection may be stalled.\n", toolID)
				continue
			}
			fmt.Printf("TCP Client [%s]: Error reading from controller: %v\n", toolID, err)
			return
		}

		parsed := ParsePayload(payload)
		midReceived := parsed["MID"].(string)

		switch midReceived {
		case "0061":
			resultData, ok := parsed["Data"].(map[string]interface{})
			if !ok {
				if errMsg, hasErr := parsed["Error"]; hasErr {
					fmt.Printf("TCP Client [%s]: FAILED PARSING MID 0061: %v\n", toolID, errMsg)
				}
				continue
			}
			
			// save to state manager
			sm.UpdateResult(toolID, resultData)

			fmt.Printf("\n--- NEW RESULT FROM [%s] ---\n", toolID)
			fmt.Printf("Mode: %v | ID Code: %v | Result: %v\n",
				resultData["Mode"], resultData["IDCode"], resultData["TighteningResult"])
			fmt.Printf("Actual Torque: %.2f | Actual Angle: %.2f\n",
				resultData["ActualTorque"], resultData["ActualAngle"])
			fmt.Println("--------------------------------")

			conn.Write([]byte(BuildMID0062()))

		case "9999": // Keep Alive reply

		default:
			fmt.Printf("TCP Client [%s]: <- Received non-result MID %s\n", toolID, midReceived)
		}
	}
}


// ConnectAndHandleTool is loop for single Nexo
func ConnectAndHandleTool(tool config.ToolConfig, sm *state.GlobalStateManager) {
	maxRetries := 5
	baseDelay := 1 * time.Second

	for { // infinite loop for connection retries
		for i := 0; i < maxRetries; i++ {
			fmt.Printf("TCP Client [%s]: Trying to connect to %s:%s (Attempt %d/%d)...\n", tool.ID, tool.IP, tool.Port, i+1, maxRetries)
			conn, err := net.DialTimeout("tcp", tool.IP+":"+tool.Port, 5*time.Second)
			//if connection success, call handler
			if err == nil {
				HandleConnection(conn, tool.ID, sm)
				// If handle connection finish, connection is terminated
				// Reset retry counter and try again
				i = -1 // will change to 0 after ++
				time.Sleep(baseDelay)
				continue
			}

			// If connection fail, apply backoff
			delay := baseDelay * time.Duration(1<<i)
			fmt.Printf("TCP Client [%s]: Connection Failed: %v. Try Again in %s.\n", tool.ID, err, delay)
			time.Sleep(delay)
		}
		// After 5 failed attempts, wait longer before the next cycle
		fmt.Printf("TCP Client [%s]: Failed to connect after %d attempts. Waiting 30 seconds before retry...\n", tool.ID, maxRetries)
		time.Sleep(30 * time.Second)
	}
}