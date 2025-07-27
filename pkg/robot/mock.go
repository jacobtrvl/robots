package robot

import (
	"fmt"
	"strings"
	"time"

	"log/slog"

	"k8s.io/apimachinery/pkg/util/uuid"
)

type MockWarehouse struct {
	robots []Robot
}

func NewMockWarehouse() *MockWarehouse {
	r := NewMockRobot()
	go r.Run()
	return &MockWarehouse{
		robots: []Robot{r},
	}
}

func (w *MockWarehouse) Robots() []Robot {
	return w.robots
}

type MockRobot struct {
	state      RobotState
	taskList   map[string]*task
	taskOrder  []string
	cancelChan chan string
}

type task struct {
	commands  []string
	errChan   chan error
	stateChan chan RobotState
}

func NewMockRobot() *MockRobot {
	return &MockRobot{
		state: RobotState{
			X:        0,
			Y:        0,
			HasCrate: false,
		},
		taskList:   make(map[string]*task),
		taskOrder:  make([]string, 0),
		cancelChan: make(chan string),
	}
}

// EnqueueTask adds a task with commands to the robot's command list
func (m *MockRobot) EnqueueTask(commands string) (string, chan RobotState, chan error) {
	commandSplit := strings.Split(commands, " ")
	taskId := string(uuid.NewUUID())
	t := task{
		commands:  commandSplit,
		errChan:   make(chan error),
		stateChan: make(chan RobotState),
	}
	m.taskList[taskId] = &t
	m.taskOrder = append(m.taskOrder, taskId)
	return taskId, t.stateChan, t.errChan
}

// CancelTask cancels a task
func (m *MockRobot) CancelTask(taskId string) error {
	if _, exists := m.taskList[taskId]; !exists {
		return fmt.Errorf("task %s not found", taskId)
	}
	m.cancelChan <- taskId
	return nil
}

// CurrentState returns the current state of the robot
func (m *MockRobot) CurrentState() RobotState {
	return m.state
}

// Run processes commands and updates the robot state
func (m *MockRobot) Run() {
	for {
		// We are avoiding sync issues/locks by running only one command/cancel at a time
		select {
		case taskId := <-m.cancelChan:
			m.cancelTask(taskId)
		default:
			m.runNextCommand()
		}
	}
}

// UpdateStateChannel is non blocking state update
// Channel might not have a receiver, hence it is non-blocking
// This could result in missed updates;
// But taking this approach to avoid command execution getting blocked
func (m *MockRobot) completeTask(taskId string, err error) {
	task, exists := m.taskList[taskId]
	if !exists {
		slog.Warn("Task not found for completion", "taskId", taskId)
		return
	}
	if err != nil {
		select {
		case task.errChan <- err:
		default:
			slog.Warn("Skipped error update, no listener available")
		}
	}
	select {
	case task.stateChan <- m.state:
	default:
		slog.Warn("Skipped state update, no listener available")
	}
	delete(m.taskList, taskId)
	m.taskOrder = m.taskOrder[1:]
	close(task.errChan)
	close(task.stateChan)
}

// cancelTask removes the task from the command list and task order
func (m *MockRobot) cancelTask(taskId string) {
	if _, exists := m.taskList[taskId]; !exists {
		slog.Warn("Task not found for cancellation; Task might be already completed or cancelled", "taskId", taskId)
		return
	}
	delete(m.taskList, taskId)
	for i, id := range m.taskOrder {
		if id == taskId {
			m.taskOrder = append(m.taskOrder[:i], m.taskOrder[i+1:]...)
			break
		}
	}
	slog.Info("Task cancelled", "taskId", taskId)
}

// runNextCommand executes the next command in the task order
func (m *MockRobot) runNextCommand() {

	if len(m.taskOrder) == 0 {
		return
	}

	taskId := m.taskOrder[0]
	task := m.taskList[taskId]

	// Cleanup finished task
	if len(task.commands) == 0 {
		m.completeTask(taskId, nil)
		return
	}

	c := task.commands[0]
	task.commands = task.commands[1:] // Remove the command from queue
	err := m.move(c)
	if err != nil {
		slog.Error("Error executing command", "command", c, "error", err)
		return
	}

	time.Sleep(time.Second) // Simulate processing time
}

// move the robot in the specified direction
// Bounds are hardcoded as per requirements
func (m *MockRobot) move(c string) error {
	switch c {
	case "N":
		if m.state.Y == 10 {
			return fmt.Errorf("invalid command: %s, out of bounds", c)
		}
		m.state.Y++
	case "S":
		if m.state.Y == 0 {
			return fmt.Errorf("invalid command: %s, out of bounds", c)
		}
		m.state.Y--
	case "E":
		if m.state.X == 10 {
			return fmt.Errorf("invalid command: %s, out of bounds", c)
		}
		m.state.X++
	case "W":
		if m.state.X == 0 {
			return fmt.Errorf("invalid command: %s, out of bounds", c)
		}
		m.state.X--
	default:
		return fmt.Errorf("invalid command: %s", c)
	}
	return nil
}
