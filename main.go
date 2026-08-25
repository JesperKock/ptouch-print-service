package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
)

type requestPayload struct {
	Cmd    string `json:"command"`
	Font   string `json:"font"`
	TextDa string `json:"text_da"`
	TextEn string `json:"text_en"`
	Text   string `json:"text"`
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
	http.HandleFunc("/template", templateHandler)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %s\n", err)
	}
}

func respondError(w http.ResponseWriter, message, err string) {
	response := responsePayload{
		Message: message,
		Error:   err,
	}
	w.WriteHeader(http.StatusInternalServerError)
	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		log.Printf("failed to encode response: %v", encErr)
	}
}

func validCommands(inputCommand string, validCommands []string) bool {
	return slices.Contains(validCommands, inputCommand)
}

// templateHandler generates a spice label template with Danish and English text
// overlaid on a base image using ptouch-print and ImageMagick.
func templateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var payload requestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	commands := []struct {
		name string
		args []string
	}{
		{
			"ptouch-print",
			[]string{"--font=IntoneMono Nerd Font", "--fontsize=26", "--text=" + payload.TextDa, "--writepng", "/images/text_da.png"},
		},
		{
			"ptouch-print",
			[]string{"--font=IntoneMono Nerd Font", "--fontsize=26", "--text=" + payload.TextEn, "--writepng", "/images/text_en.png"},
		},
		{
			"magick",
			[]string{"/images/text_da.png", "-alpha", "set", "-channel", "RGBA", "-transparent", "white", "/images/da_alpha.png"},
		},
		{
			"magick",
			[]string{"/images/text_en.png", "-alpha", "set", "-channel", "RGBA", "-transparent", "white", "/images/en_alpha.png"},
		},
		{
			"magick",
			[]string{"/templates/spicelabel.png", "da_trans_${i}.png", "-gravity", "Center", "-geometry", "+0-30", "-composite", "/images/splice_label-" + payload.TextEn + ".png"},
		},
		{
			"magick",
			[]string{"/images/spice_label-" + payload.TextEn + ".png", "en_trans_${i}.png", "-gravity", "Center", "-geometry", "+0+30", "-composite", "/images/splice_label-" + payload.TextEn + ".png"},
		},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd.name, cmd.args...).Run(); err != nil {
			respondError(w, "Command execution failed", err.Error())
			return
		}
	}

	// TODO: file cleanup
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

	if !validCommands(payload.Cmd, validInputCommands) {
		response := responsePayload{
			Message: "Validation failed",
			Error:   "Valid commands is --font, --fontsize, --writepng, --image, --text, --cutmark, --pad",
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
