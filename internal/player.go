package internal

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
)

func MPVSendCommand(ipcSocketPath string, command []interface{}) (interface{}, error) {
    var conn net.Conn
    var err error

    if runtime.GOOS == "windows" {
        // Use named pipe for Windows
        // conn, err = winio.DialPipe(ipcSocketPath, nil)
    } else {
        conn, err = net.Dial("unix", ipcSocketPath)
    }
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    commandStr, err := json.Marshal(map[string]interface{}{
        "command": command,
    })
    if err != nil {
        return nil, err
    }

    // Send the command
    _, err = conn.Write(append(commandStr, '\n'))
    if err != nil {
        return nil, err
    }

    // Receive the response
    buf := make([]byte, 4096)
    n, err := conn.Read(buf)
    if err != nil {
        return nil, err
    }

    var response map[string]interface{}
    if err := json.Unmarshal(buf[:n], &response); err != nil {
        return nil, err
    }

    if data, exists := response["data"]; exists {
        return data, nil
    }

    return nil, nil
}

func SeekMPV(ipcSocketPath string, time int) (interface{}, error) {
	command := []interface{}{"seek", time, "absolute"}
	return MPVSendCommand(ipcSocketPath, command)
}

func GetMPVPausedStatus(ipcSocketPath string) (bool, error) {
	status, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "pause"})
	if err != nil || status == nil {
		return false, err
	}

	paused, ok := status.(bool)
	if ok {
		return paused, nil
	}
	return false, nil
}

func GetMPVPlaybackSpeed(ipcSocketPath string) (float64, error) {
	speed, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "speed"})
	if err != nil || speed == nil {
		return 0, err
	}

	currentSpeed, ok := speed.(float64)
	if ok {
		return currentSpeed, nil
	}

	return 0, nil
}

func GetPercentageWatched(ipcSocketPath string) (float64, error) {
	currentTime, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "time-pos"})
	if err != nil || currentTime == nil {
		return 0, err
	}

	duration, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "duration"})
	if err != nil || duration == nil {
		return 0, err
	}

	currTime, ok1 := currentTime.(float64)
	dur, ok2 := duration.(float64)

	if ok1 && ok2 && dur > 0 {
		percentageWatched := (currTime / dur) * 100
		return percentageWatched, nil
	}

	return 0, nil
}

func PercentageWatched(playbackTime int, duration int) float64 {
	if duration > 0 {
		percentage := (float64(playbackTime) / float64(duration)) * 100
		return percentage
	}
	return float64(0)
}

// GetMPVPosition gets the current playback position
func GetMPVPosition(socketPath string) (float64, error) {
	return getMPVProperty(socketPath, "time-pos")
}

// GetMPVDuration gets the total duration of the current file
func GetMPVDuration(socketPath string) (float64, error) {
	return getMPVProperty(socketPath, "duration")
}

// StopMPV stops the current playback
func StopMPV(socketPath string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := struct {
		Command   []string `json:"command"`
		RequestID int      `json:"request_id"`
	}{
		Command:   []string{"quit"},
		RequestID: 1,
	}

	return json.NewEncoder(conn).Encode(cmd)
}

// getMPVProperty is a helper function to get MPV properties
func getMPVProperty(socketPath string, property string) (float64, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	cmd := struct {
		Command   []string `json:"command"`
		RequestID int      `json:"request_id"`
	}{
		Command:   []string{"get_property", property},
		RequestID: 1,
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return 0, err
	}

	var response struct {
		Data      float64 `json:"data"`
		Error     string  `json:"error"`
		RequestID int     `json:"request_id"`
	}

	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return 0, err
	}

	if response.Error != "" {
		return 0, fmt.Errorf("mpv error: %s", response.Error)
	}

	return response.Data, nil
}
