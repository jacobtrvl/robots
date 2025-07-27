package api

import (
	"fmt"

	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jacobtrvl/robots/pkg/robot"
)

type RobotAPI struct {
	w robot.Warehouse
	// Currently holds the status of all tasks.
	// This should be persisted & cleared periodically.
	status map[string]string
}

func NewRobotApi(w robot.Warehouse) *RobotAPI {
	return &RobotAPI{
		w:      w,
		status: make(map[string]string),
	}
}

func (a *RobotAPI) EnqueueHandler(w http.ResponseWriter, r *http.Request) {
	robot, err := a.getRobot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		Commands string `json:"commands"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	taskID, stateCh, errCh, err := enqueuTask(robot, req.Commands)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.streamState(taskID, stateCh, errCh)
	a.status[taskID] = "Queued"
	resp := map[string]string{"task_id": taskID}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
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
	a.status[taskID] = "Cancelled"
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
	r.HandleFunc("/status/{taskID}", a.taskStatus).Methods("GET")
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

func enqueuTask(robot robot.Robot, commands string) (string, chan robot.RobotState, chan error, error) {
	if commands == "" {
		return "", nil, nil, fmt.Errorf("commands are required")
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

func (a *RobotAPI) taskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskID"]
	status := "Task not found"
	if s, ok := a.status[taskID]; ok {
		status = s
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// streamState just logs the robot state and errors from channels.
// Chance of goroutine leak if channels are properly handled in Robot function.
func (a *RobotAPI) streamState(taskId string, stateChan chan robot.RobotState, errChan chan error) {
	go func() {
		select {
		case err, ok := <-errChan:
			if ok && err != nil {
				a.status[taskId] = "Error: " + err.Error()
				slog.Error("Task execution failed", "error", err.Error(), "task_id", taskId)
			}
		case state, ok := <-stateChan:
			if ok {
				a.status[taskId] = "Completed"
				slog.Info("Completed command. Current state:", "X:", state.X, "Y:", state.Y, "HasCrate:", state.HasCrate, "task_id", taskId)
			}
		}
	}()
}
