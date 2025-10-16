# Go Server for Bosch Rexroth Nexo Tools
## Rexroth OpenProtocol Rev1

![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)
![License](https://img.shields.io/badge/License-MIT-green.svg)

A robust, multi-tool TCP server written in Go to communicate with Bosch Rexroth Nexo cordless nutrunners using the Rexroth Open Protocol. This server can connect to multiple tools simultaneously, listen for tightening results, and expose the data through a clean JSON REST API.

---

## ## Features

-   **Multi-Tool Connectivity**: Connects to and handles multiple Nexo tools at the same time, each in its own dedicated goroutine.
-   **Automatic Reconnection**: If a tool becomes disconnected, the server will automatically try to reconnect with an exponential backoff strategy.
-   **Open Protocol Parsing**: Parses incoming tightening result messages (MID 0061) to extract key data points like torque, angle, and status.
-   **RESTful API**: Exposes a simple and clean JSON API (built with [Fiber](https://gofiber.io/)) to retrieve the status of all tools and the latest tightening data from any specific tool.
-   **Structured & Maintainable**: The project is organized into logical packages (`api`, `config`, `nexo`, `state`) for easy maintenance and scalability.

---

## ## Project Structure

The project follows a standard Go application layout to keep the code organized:
nexo-server/

├── cmd/

│   └── server/

│       └── main.go         # Main application entry point

├── internal/

│   ├── api/                # Web API (Fiber) handlers and server setup

│   ├── config/             # Tool configuration

│   ├── nexo/               # Logic for TCP connection and Open Protocol

│   └── state/              # In-memory, concurrent-safe state management

└── go.mod                  # Go module definition

---

## ## Getting Started

Follow these instructions to get the server up and running on your local machine.

### ### Prerequisites

-   [Go](https://go.dev/doc/install) (version 1.18 or newer) installed on your system.
-   Access to the Nexo tools on the network.

### ### Installation & Running

1.  **Clone the repository:**
    ```sh
    git clone [https://your-repository-url.com/nexo-server.git](https://your-repository-url.com/nexo-server.git)
    cd nexo-server
    ```

2.  **Install dependencies:**
    This command will download the necessary modules (like Fiber) defined in `go.mod`.
    ```sh
    go mod tidy
    ```

3.  **Configure your tools:**
    Open the `internal/config/config.go` file and modify the `toolConfigs` slice to match the IP addresses and details of your Nexo tools.

    ```go
    // File: internal/config/config.go
    func GetTools() []ToolConfig {
        return []ToolConfig{
            {ID: "nexo-pistol", Name: "Nexo Pistol Grip", IP: "192.168.0.23", Port: "4545"},
            {ID: "nexo-angle", Name: "Nexo Angle Nutrunner", IP: "192.168.0.25", Port: "4545"},
            // Add more tools here
        }
    }
    ```

4.  **Run the server:**
    Execute the following command from the root directory of the project.
    ```sh
    go run ./cmd/server
    ```
    The server will start, attempt to connect to all configured tools, and launch the API on port `8080`.

---

## ## API Usage

The API provides endpoints to monitor the status of the tools and retrieve the latest tightening data.

### ### 1. Get Status of All Tools

Returns a list of all configured tools and their current connection status.

-   **Endpoint**: `GET /tools`
-   **Method**: `GET`
-   **Success Response (200 OK)**:
    ```json
    {
        "status": "ok",
        "tools": {
            "nexo-angle": {
                "name": "Nexo Angle Nutrunner",
                "ip": "192.168.0.25",
                "port": "4545",
                "connected": true,
                "last_update": "2025-10-16T15:05:12Z"
            },
            "nexo-pistol": {
                "name": "Nexo Pistol Grip",
                "ip": "192.168.0.23",
                "port": "4545",
                "connected": false,
                "last_update": "0001-01-01T00:00:00Z"
            }
        },
        "timestamp": "2025-10-16T15:05:20Z"
    }
    ```

### ### 2. Get Last Result from a Specific Tool

Returns the last tightening result received from a single, specified tool.

-   **Endpoint**: `GET /tools/:id`
-   **Method**: `GET`
-   **URL Params**:
    -   `id=[string]` (Required) - The `ID` of the tool as defined in `config.go`. Example: `nexo-pistol`.
-   **Success Response (200 OK)**:
    ```json
    {
        "status": "ok",
        "tool_id": "nexo-angle",
        "connected": true,
        "last_result": {
            "ActualAngle": 120.5,
            "ActualTorque": 45.75,
            "ControllerName": "NEXO-CTRL-02",
            "IDCode": "PART-XYZ-123",
            "ProgramNumber": 3,
            "TighteningResult": "OK",
            "Timestamp": "2025-10-16 22:05:12"
        },
        "last_update": "2025-10-16T15:05:12Z"
    }
    ```
-   **Error Response (404 Not Found)**:
    ```json
    {
        "status": "error",
        "message": "Tool with id 'invalid-id' not found or not configured."
    }
    ```

---

## ## License

This project is licensed under the MIT License. See the `LICENSE` file for details.