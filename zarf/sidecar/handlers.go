package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

type handlers struct {
	composeFile string
}

// =========================================================================
// GET /containers

type ContainerInfo struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Status string `json:"status"`
	Health string `json:"health"`
	Image  string `json:"image"`
	Ports  string `json:"ports"`
}

func (h *handlers) containers(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command(
		"docker", "compose", "-f", h.composeFile,
		"ps", "--format", "json", "-a",
	).Output()
	if err != nil {
		writeError(w, 500, "docker compose ps failed: "+err.Error())
		return
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			Name   string `json:"Name"`
			State  string `json:"State"`
			Status string `json:"Status"`
			Health string `json:"Health"`
			Image  string `json:"Image"`
			Ports  string `json:"Ports"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		containers = append(containers, ContainerInfo{
			Name:   raw.Name,
			State:  raw.State,
			Status: raw.Status,
			Health: raw.Health,
			Image:  raw.Image,
			Ports:  raw.Ports,
		})
	}
	if containers == nil {
		containers = []ContainerInfo{}
	}
	writeJSON(w, containers)
}

// =========================================================================
// GET /timers

type TimerInfo struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	LastRun string `json:"lastRun"`
	NextRun string `json:"nextRun"`
	Result  string `json:"result"`
}

func (h *handlers) timers(w http.ResponseWriter, r *http.Request) {
	units := []string{"planner-deploy", "planner-backup"}
	var timers []TimerInfo

	for _, unit := range units {
		t := TimerInfo{Name: unit}

		activeOut, _ := exec.Command("systemctl", "is-active", unit+".timer").Output()
		t.Active = strings.TrimSpace(string(activeOut)) == "active"

		lastOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=LastTriggerUSec", "--value").Output()
		t.LastRun = strings.TrimSpace(string(lastOut))

		nextOut, _ := exec.Command("systemctl", "show", unit+".timer", "--property=NextElapseUSecRealtime", "--value").Output()
		t.NextRun = strings.TrimSpace(string(nextOut))

		resultOut, _ := exec.Command("systemctl", "show", unit+".service", "--property=Result", "--value").Output()
		t.Result = strings.TrimSpace(string(resultOut))

		timers = append(timers, t)
	}

	writeJSON(w, timers)
}

// =========================================================================
// GET /logs/{service}?lines=100

func (h *handlers) logs(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")

	allowed := map[string]bool{
		"backend": true, "frontend": true, "db": true,
		"planner-deploy": true, "planner-backup": true,
	}
	if !allowed[service] {
		writeError(w, 400, "unknown service: "+service)
		return
	}

	lines := 100
	if n := r.URL.Query().Get("lines"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 500 {
			lines = parsed
		}
	}

	var out []byte
	var err error

	switch service {
	case "planner-deploy", "planner-backup":
		out, err = exec.Command(
			"journalctl", "-u", service+".service",
			"--no-pager", "-n", strconv.Itoa(lines), "--output=short-iso",
		).Output()
	default:
		out, err = exec.Command(
			"docker", "compose", "-f", h.composeFile,
			"logs", "--tail", strconv.Itoa(lines), "--no-color", service,
		).Output()
	}
	if err != nil {
		writeError(w, 500, "failed to get logs: "+err.Error())
		return
	}

	writeJSON(w, map[string]string{"logs": string(out)})
}

// =========================================================================
// GET /claude

type ClaudeInstance struct {
	PID     string `json:"pid"`
	Command string `json:"command"`
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Elapsed string `json:"elapsed"`
}

func (h *handlers) claude(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("ps", "aux").Output()

	var instances []ClaudeInstance
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "claude") || strings.Contains(line, "grep") || strings.Contains(line, "sidecar") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		instances = append(instances, ClaudeInstance{
			PID:     fields[1],
			CPU:     fields[2],
			Memory:  fields[3],
			Elapsed: fields[9],
			Command: strings.Join(fields[10:], " "),
		})
	}
	if instances == nil {
		instances = []ClaudeInstance{}
	}
	writeJSON(w, instances)
}
