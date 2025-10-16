package nexo

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	protocolHeaderLength = 20
	nulByte              = "\x00"
)

// CreateOPMessage
func CreateOPMessage(mid, revision, data string) string {
	dataLength := len(data)
	messageLength := protocolHeaderLength + dataLength
	lengthStr := fmt.Sprintf("%04d", messageLength)
	header := fmt.Sprintf("%s%s%s0%s", lengthStr, mid, revision, strings.Repeat("0", 8))
	message := header + data + nulByte
	return message
}

// MID-building functions
func BuildMID0001() string { return CreateOPMessage("0001", "001", "") }
func BuildMID0060() string { return CreateOPMessage("0060", "001", "") }
func BuildMID0062() string { return CreateOPMessage("0062", "001", "") }
func BuildMID9999() string { return CreateOPMessage("9999", "001", "") }

// ParsePayload.
func ParsePayload(payload string) map[string]interface{} {
    payload = strings.TrimSuffix(payload, nulByte)
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