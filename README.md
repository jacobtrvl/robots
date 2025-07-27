# Robot Warehouse

RESTful APIs and a mock warehouse/robot implementation.

## Requirements
- Go installed (tested with version 1.24.4)

## Regarding status API
- As per the requirement API that need to be supported is
   "Create a RESTful API to report the command series's execution status."
  This could mean we report the status of taskId.
  But Robot interface predefined in the question doesn't provide any methods to get the status of the task. Hence I am providing state & task status with whatever possible states we can derived of the given interface.
  I donot want to introduce additional method to interface, but use existing ones & implement the logic


## Running the Application
```bash
make run
```

## Testing
- Port is hardcoded to 8080. Currently not configurable. Change in main.go file, if required.
### Enqueue API
```bash
curl --location 'http://localhost:8080/enqueue' \
--header 'Content-Type: application/json' \
--data '{
  "commands": "N E S W N E S W N E S W N E S W"
}'
```
- Note: commands are not validated before accepting.

### Get task status
```bash
curl --location 'http://localhost:8080/status/<task_id>'
```
### Get the robot state
```bash
curl --location 'http://localhost:8080/state'
```

### Cancel a task
```bash
curl --location --request POST 'http://localhost:8080/cancel/<taskid>'
```


## Notifying Ground Station - Design

The Robot interface provides channels `stateCh` & `errCh` for command execution state updates. 
In the current implementation, the MockRobot sends the state once a command series (task) is executed via `stateCh`.
In case of errors, they are passed to `errCh`.

The API layer receives these events and currently just logs them. Instead, we could implement proper alerting:

### Alerting Options
Alerting depends on the current infrastructure available at the Control Station:

- **Alerting logic in same application, in a different goroutine**: Instead of  logging, we can implement necessary logic for alerting ground station.
- **gRPC Streaming**: Stream events to another monitoring & alerting application through RPCs like gRPC
- **Monitoring Integration**: If monitoring systems like Grafana or DataDog are available, configure alerts when events occur. This mechanism is great when third-party alerting mechanisms like SMS, slack messages etc is required.
- **Message Streaming**: Send events to Kafka streams where another monitoring application handles alerting
- **Webhook Notifications**: Send HTTP webhooks to external systems for real-time notifications

In general, any event-driven notification mechanism can be used. 
- For simplicity, another microservice for monitoring & alerting logic, with direct communication with Robots/Warehouse will be my proposal
- Communication with warehouse through channels may not be ideal in production setup. We should use right RPC mechanism with proper retry mechanisms and error handling. 

Proposed architecture in summary: 
     
     
     Robot/Warehouse agent <--- gRPC---> API gateway ---grpc--> Monitoring & Alerting service
