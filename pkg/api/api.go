package api

import (
	"context"
	"fmt"
	"strings"

	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jacobtrvl/robots/pkg/robot"
)

type RobotAPI struct {
	w robot.Warehouse
}

func NewRobotApi(w robot.Warehouse) *RobotAPI {
	return &RobotAPI{
		w: w,
	}
}

func (a *RobotAPI) EnqueueHandler(w http.ResponseWriter, r *http.Request) {
	robot, err := a.getRobot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	validate := r.URL.Query().Get("validate")
	var req struct {
		Commands string `json:"commands"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	taskID, stateCh, errCh, err := enqueuTask(robot, req.Commands, validate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	streamState(r.Context(), stateCh, errCh)
	resp := map[string]string{"task_id": taskID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *RobotAPI) CancelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskID"]
	robot, err := a.getRobot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := cancelCommand(robot, taskID); err != nil {
		http.Error(w, fmt.Sprintf("failed to cancel task: %v", err), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *RobotAPI) StateHandler(w http.ResponseWriter, r *http.Request) {
	robot, err := a.getRobot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state := currentState(robot)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (a *RobotAPI) NewRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/enqueue", a.EnqueueHandler).Methods("POST")
	r.HandleFunc("/cancel/{taskID}", a.CancelHandler).Methods("POST")
	r.HandleFunc("/state", a.StateHandler).Methods("GET")

	// robotId is igonored!
	r.HandleFunc("/enqueue/{robotId}", a.EnqueueHandler).Methods("POST")
	r.HandleFunc("/cancel/{robotId}/{taskID}", a.CancelHandler).Methods("POST")
	r.HandleFunc("/state/{robotId}", a.StateHandler).Methods("GET")
	return r
}

func (a *RobotAPI) getRobot() (robot.Robot, error) {
	// Dummy method
	// For this exercise, we are hardcoding the robot instance
	// Robot instance might be based on warehouse and robot id. Depends on actual business logic.
	if a.w == nil || len(a.w.Robots()) == 0 {
		return nil, fmt.Errorf("warehouse not initialized or no robots available")
	}
	return a.w.Robots()[0], nil
}

func enqueuTask(robot robot.Robot, commands string, validate string) (string, chan robot.RobotState, chan error, error) {
	if commands == "" {
		return "", nil, nil, fmt.Errorf("commands are required")
	}
	if validate == "true" {
		err := isValidCommand(commands, robot)
		if err != nil {
			return "", nil, nil, fmt.Errorf("task enqueue failed: %w", err)
		}
	}
	taskId, stateCh, errCh := robot.EnqueueTask(commands)
	return taskId, stateCh, errCh, nil
}

func cancelCommand(robot robot.Robot, taskID string) error {
	return robot.CancelTask(taskID)
}

func currentState(robot robot.Robot) robot.RobotState {
	return robot.CurrentState()
}

// streamState just logs the robot state and errors from channels.
// Chance of goroutine leak if channels are not closed properly.
func streamState(_ context.Context, stateChan chan robot.RobotState, errChan chan error) {
	go func() {
		for state := range stateChan {
			slog.Info("Completed command. Current state:", "X:", state.X, "Y:", state.Y, "HasCrate:", state.HasCrate)
		}
	}()
	go func() {
		for err := range errChan {
			if err != nil {
				slog.Error("Error in robot command execution", "error", err.Error())
			}
		}
	}()
}

func isValidCommand(command string, robot robot.Robot) error {
	state := robot.CurrentState()
	x, y := int(state.X), int(state.Y)
	var err error
	commandSplit := strings.Split(command, " ")
	for _, c := range commandSplit {
		x, y, err = simulateMove(x, y, c)
		if err != nil {
			return fmt.Errorf("command validation failed %s: %w", command, err)
		}
		if !withinBounds(x, y) {
			return fmt.Errorf("command validation failed %s: out of bounds (%d, %d)", command, x, y)
		}
	}
	return nil
}

func simulateMove(x int, y int, direction string) (int, int, error) {
	var err error
	switch direction {
	case "N":
		y++
	case "S":
		y--
	case "E":
		x++
	case "W":
		x--
	default:
		err = fmt.Errorf("invalid direction: %s", direction)
	}
	return x, y, err
}

// withinBounds checks coordinates validity.
// Checking as int to avoid overflow issues
// This may not work if we are using whole uint range.
func withinBounds(x, y int) bool {
	return x >= 0 && x <= 10 && y >= 0 && y <= 10
}
