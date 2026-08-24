package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
)

type requestPayload struct {
	Cmd  string `json:"cmd"`
	Font string `json:"font"`
	Text string `json:"text"`
}

type responsePayload struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

var (
	validInputRegex    = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	validInputCommands = []string{"--font", "--fontsize", "--writepng", "--image", "--text", "--cutmark", "--pad"}
)

func main() {
	http.HandleFunc("/print", printHandler)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %s\n", err)
	}
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	cmd := exec.Command("ptouch-print", "--info")
	if err := cmd.Run(); err != nil {
		response := responsePayload{
			Message: "Command execution failed",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
}

func printHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var payload requestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if !validInputRegex.MatchString(payload.Text) {
		response := responsePayload{
			Message: "Validation failed",
			Error:   "Input must only contain alphanumeric characters or dashes",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if validCommands(payload.Cmd, validInputCommands) {
		response := responsePayload{
			Message: "Validation failed",
			Error: `Valid commands:
						--font <file>		use font <file> or <name>
						--fontsize <size>	Manually set fontsize
						--writepng <file>	instead of printing, write output to png file
						--image <file>		print the given image
						--text <text>		Print 1-4 lines of text.
						--cutmark			Print a mark where the tape should be cut
						--pad <n>			Add n pixels padding (blank tape)
						--info				show info about detected tape`,
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	cmd := exec.Command("ptouch-print", payload.Cmd, payload.Text)
	if err := cmd.Run(); err != nil {
		response := responsePayload{
			Message: "Command execution failed",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responsePayload{
		Message: "Command executed successfully",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
