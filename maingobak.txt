package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// --- 1. Konfigurasi dan Struktur Data Baru ---

const (
	APIPort              = "8080" // Port untuk API web (HTTP)
	ProtocolHeaderLength = 20
	NulByte              = "\x00"
)

// ToolConfig mendefinisikan konfigurasi untuk setiap Nexo tool
type ToolConfig struct {
	ID   string // ID unik untuk API, misal "nexo-pistol"
	Name string // Nama deskriptif, misal "Nexo Pistol Grip"
	IP   string
	Port string
}

// toolConfigs adalah daftar semua alat yang akan dihubungkan oleh server.
// Tambahkan alat baru di sini.
var toolConfigs = []ToolConfig{
	{ID: "nexo-pistol", Name: "Nexo Pistol Grip", IP: "192.168.0.23", Port: "4545"},
	{ID: "nexo-angle", Name: "Nexo Angle Nutrunner", IP: "192.168.0.25", Port: "4545"},
	// Tambahkan alat lain di sini jika ada
	// {ID: "nexo-lantai-2", Name: "Nexo di Lantai 2", IP: "192.168.0.30", Port: "4545"},
}

// ToolState menyimpan data terakhir dan status koneksi untuk SATU alat.
type ToolState struct {
	LatestResult map[string]interface{}
	IsConnected  bool
	LastUpdate   time.Time
}

// GlobalStateManager adalah manager state yang concurrent-safe untuk semua alat.
// Kita menggunakan RWMutex untuk performa yang lebih baik (banyak read, sedikit write).
type GlobalStateManager struct {
	sync.RWMutex
	Tools map[string]*ToolState
}

// Inisialisasi state manager global
var stateManager = GlobalStateManager{
	Tools: make(map[string]*ToolState),
}

// Fungsi helper untuk state manager
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
	// Mengembalikan salinan untuk mencegah race condition di luar
	if !ok {
		return nil, false
	}
	stateCopy := &ToolState{
		LatestResult: tool.LatestResult,
		IsConnected:  tool.IsConnected,
		LastUpdate:   tool.LastUpdate,
	}
	return stateCopy, true
}

func (sm *GlobalStateManager) GetAllToolsStatus() map[string]interface{} {
	sm.RLock()
	defer sm.RUnlock()
	
	statusMap := make(map[string]interface{})
	for _, config := range toolConfigs {
		state, ok := sm.Tools[config.ID]
		if !ok {
			continue
		}
		statusMap[config.ID] = map[string]interface{}{
			"id":          config.ID,
			"name":        config.Name,
			"ip":          config.IP,
			"port":        config.Port,
			"connected":   state.IsConnected,
			"last_update": state.LastUpdate.Format(time.RFC3339),
		}
	}
	return statusMap
}


// --- Struktur Pesan Open Protocol (MID) ---
// (Tidak ada perubahan di bagian ini, fungsinya sudah benar)
func CreateOPMessage(mid, revision, data string) string {
	dataLength := len(data)
	messageLength := ProtocolHeaderLength + dataLength
	lengthStr := fmt.Sprintf("%04d", messageLength)
	header := fmt.Sprintf("%s%s%s0%s", lengthStr, mid, revision, strings.Repeat("0", 8))
	message := header + data + NulByte
	return message
}

func BuildMID0001() string { return CreateOPMessage("0001", "001", "") }
func BuildMID0060() string { return CreateOPMessage("0060", "001", "") }
func BuildMID0062() string { return CreateOPMessage("0062", "001", "") }
func BuildMID9999() string { return CreateOPMessage("9999", "001", "") }
func BuildMID0003() string { return CreateOPMessage("0003", "001", "") }


// --- Parsing Logik ---
// (Tidak ada perubahan di bagian ini, fungsinya sudah benar)
func ParsePayload(payload string) map[string]interface{} {
    payload = strings.TrimSuffix(payload, NulByte)
    payload = strings.ReplaceAll(payload, " ", "")

    response := map[string]interface{}{
        "Length":    payload[0:4],
        "MID":       payload[4:8],
        "Revision":  payload[8:11],
        "NoAckFlag": payload[11:20],
    }

    mid := response["MID"].(string)
    if mid != "0061" {
        response["Data"] = "Non-result MID, skipping detailed parsing."
        return response
    }

    outputLength := len(payload)
    var dataSlices map[string]string
    var mode string

    if outputLength >= 205 {
        mode = "Automatic (Length >= 205)"
        dataSlices = map[string]string{
            "CellID":           payload[22:26],
            "ChannelID":        payload[28:30],
            "ControllerName":   payload[32:47],
            "IDCode":           payload[49:58],
            "JobNumber":        payload[60:62],
            "ProgramNumber":    payload[64:67],
            "OKLimit":          payload[69:73],
            "OKValue":          payload[75:79],
            "TighteningStatus": payload[81:82],
            "TorqueStatus":     payload[84:85],
            "AngleStatus":      payload[87:88],
            "MinTorque":        payload[90:96],
            "MaxTorque":        payload[98:104],
            "TargetTorque":     payload[106:112],
            "ActualTorque":     payload[114:120],
            "MinAngle":         payload[122:127],
            "MaxAngle":         payload[129:134],
            "TargetAngle":      payload[136:141],
            "ActualAngle":      payload[143:148],
            "Timestamp":        payload[150:169],
            "LastChange":       payload[171:190],
            "CounterStatus":    payload[192:193],
            "TighteningID":     payload[195:205],
        }
    } else if outputLength <= 197 && outputLength >= 196 {
        mode = "Manual (Length <= 197)"
        dataSlices = map[string]string{
            "CellID":           payload[22:26],
            "ChannelID":        payload[28:30],
            "ControllerName":   payload[32:47],
            "IDCode":           "-",
            "JobNumber":        payload[51:53],
            "ProgramNumber":    payload[55:58],
            "OKLimit":          payload[60:64],
            "OKValue":          payload[66:70],
            "TighteningStatus": payload[72:73],
            "TorqueStatus":     payload[75:76],
            "AngleStatus":      payload[78:79],
            "MinTorque":        payload[81:87],
            "MaxTorque":        payload[89:95],
            "TargetTorque":     payload[97:103],
            "ActualTorque":     payload[105:111],
            "MinAngle":         payload[113:118],
            "MaxAngle":         payload[120:125],
            "TargetAngle":      payload[127:132],
            "ActualAngle":      payload[134:139],
            "Timestamp":        payload[141:160],
            "LastChange":       payload[162:181],
            "CounterStatus":    payload[183:184],
            "TighteningID":     payload[186:196],
        }
    } else {
        response["Error"] = fmt.Sprintf("MID 0061: Unexpected string length: %d", outputLength)
        return response
    }

    parsedData := make(map[string]interface{})
    parsedData["Mode"] = mode

    for k, v := range dataSlices {
        v = strings.TrimSpace(v)
        vFloat, err := strconv.ParseFloat(v, 64)
        if err == nil {
            switch k {
            case "ActualTorque", "TargetTorque", "MinTorque", "MaxTorque":
                parsedData[k] = vFloat / 100.0
            case "ActualAngle", "TargetAngle", "MinAngle", "MaxAngle":
                parsedData[k] = vFloat
            case "TighteningStatus":
                parsedData["TighteningResult"] = "NOK"
                if v == "1" {
                    parsedData["TighteningResult"] = "OK"
                }
            default:
                parsedData[k] = vFloat
            }
        } else {
            parsedData[k] = v
        }
    }

    response["Data"] = parsedData
    return response
}

// --- 2. Fungsi HTTP Server dengan Fiber (Telah Dimodifikasi) ---

// handleGetAllTools menangani permintaan API untuk status semua alat
func handleGetAllTools(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"tools":     stateManager.GetAllToolsStatus(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleGetToolResult menangani permintaan API untuk hasil terakhir dari alat spesifik
func handleGetToolResult(c *fiber.Ctx) error {
	toolID := c.Params("id")
	
	state, found := stateManager.GetToolState(toolID)
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

func startHTTPServer(port string) {
	app := fiber.New(fiber.Config{ServerHeader: "Nexo-OP-Fiber"})

	// Rute API baru
	app.Get("/tools", handleGetAllTools)
	app.Get("/tools/:id", handleGetToolResult)
	// Rute lama bisa dipertahankan untuk backward compatibility jika perlu
	// app.Get("/latest_result", handleGetToolResult) // Contoh jika mau

	fmt.Printf("API Server: Starting HTTP Server on port %s for API access...\n", port)
	if err := app.Listen(":" + port); err != nil {
		fmt.Printf("API Server Error: %v\n", err)
	}
}

// --- 3. Fungsi Koneksi dan Komunikasi TCP (Telah Dimodifikasi) ---

// HandleConnection sekarang menerima toolID untuk mengelola state yang benar
func HandleConnection(conn net.Conn, toolID string) {
	defer conn.Close()
	defer func() {
		stateManager.SetConnectionStatus(toolID, false)
		fmt.Printf("TCP Client [%s]: Koneksi ditutup.\n", toolID)
	}()

	fmt.Printf("TCP Client [%s]: Successfully connected to %s. Starting communication setup...\n", toolID, conn.RemoteAddr())
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 1. Kirim MID 0001 dan terima MID 0002
	conn.Write([]byte(BuildMID0001()))
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString(NulByte[0])
	if err != nil || !strings.Contains(response, "0002") {
		fmt.Printf("TCP Client [%s]: Failed to read/MID 0002 not received: %v\n", toolID, err)
		return
	}
	fmt.Printf("TCP Client [%s]: <- MID 0002 received. Communication Established.\n", toolID)

	// 2. Kirim MID 0060 dan terima MID 0005
	conn.Write([]byte(BuildMID0060()))
	response, err = reader.ReadString(NulByte[0])
	if err != nil || !strings.Contains(response, "0005") {
		fmt.Printf("TCP Client [%s]: Failed to read/MID 0005 for MID 0060 not received: %v\n", toolID, err)
		return
	}
	fmt.Printf("TCP Client [%s]: <- MID 0005 received. Listening for results.\n", toolID)

	conn.SetReadDeadline(time.Time{})
	stateManager.SetConnectionStatus(toolID, true)

	// Goroutine Keep Alive per koneksi
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if state, ok := stateManager.GetToolState(toolID); !ok || !state.IsConnected {
				return // Hentikan jika koneksi sudah terputus
			}
			if _, err := conn.Write([]byte(BuildMID9999())); err != nil {
				fmt.Printf("TCP Client [%s]: Error sending Keep Alive: %v. Stopping.\n", toolID, err)
				return
			}
		}
	}()

	// Loop utama untuk membaca data
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		payload, err := reader.ReadString(NulByte[0])
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
			
			// Simpan hasil ke state manager yang benar
			stateManager.UpdateResult(toolID, resultData)

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

// connectAndHandleTool adalah loop untuk satu alat, menangani koneksi dan koneksi ulang.
func connectAndHandleTool(tool ToolConfig) {
	maxRetries := 5
	baseDelay := 1 * time.Second

	for { // Loop tak terbatas untuk koneksi ulang
		for i := 0; i < maxRetries; i++ {
			fmt.Printf("TCP Client [%s]: Mencoba terhubung ke %s:%s (Percobaan %d/%d)...\n", tool.ID, tool.IP, tool.Port, i+1, maxRetries)
			conn, err := net.DialTimeout("tcp", tool.IP+":"+tool.Port, 5*time.Second)
			if err == nil {
				// Koneksi berhasil, panggil handler
				HandleConnection(conn, tool.ID)
				// Jika HandleConnection selesai, berarti koneksi terputus.
				// Reset retry counter dan coba lagi segera.
				i = -1 // akan menjadi 0 setelah i++
				time.Sleep(baseDelay)
				continue
			}
			
			// Koneksi gagal, terapkan backoff
			delay := baseDelay * time.Duration(1<<i)
			fmt.Printf("TCP Client [%s]: Koneksi gagal: %v. Mencoba lagi dalam %s.\n", tool.ID, err, delay)
			time.Sleep(delay)
		}
		// Setelah 5 percobaan gagal, tunggu lebih lama sebelum siklus baru
		fmt.Printf("TCP Client [%s]: Gagal terhubung setelah %d percobaan. Menunggu 30 detik...\n", tool.ID, maxRetries)
		time.Sleep(30 * time.Second)
	}
}

// --- 4. main() yang baru ---
func main() {
	fmt.Println("--- Rexroth Nexo Open Protocol Multi-Tool Client & Web API ---")

	// 1. Jalankan HTTP Server Fiber di goroutine terpisah
	go startHTTPServer(APIPort)

	// 2. Inisialisasi state untuk setiap alat
	for _, tool := range toolConfigs {
		stateManager.InitTool(tool.ID)
	}

	// 3. Jalankan goroutine untuk setiap alat yang dikonfigurasi
	var wg sync.WaitGroup
	for _, tool := range toolConfigs {
		wg.Add(1)
		go func(t ToolConfig) {
			defer wg.Done()
			connectAndHandleTool(t)
		}(tool)
	}
	
	// Tunggu semua goroutine selesai (dalam kasus ini, tidak akan pernah,
	// jadi program akan terus berjalan)
	wg.Wait()
}