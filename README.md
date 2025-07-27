# Robot Warehouse

RESTful APIs and a mock warehouse/robot implementation.

## Requirements
- Go installed (tested with version 1.24.4)

## Running the Application
```bash
make run
```

## Testing

### Enqueue without validation at API layer
```bash
curl --location 'http://localhost:8080/enqueue' \
--header 'Content-Type: application/json' \
--data '{
  "commands": "N E S W N E S W N E S W N E S W"
}'
```

### Enqueue with additional validation before sending to robot
```bash
curl --location 'http://localhost:8080/enqueue?validate=true' \
--header 'Content-Type: application/json' \
--data '{
  "commands": "N E S W N E S W N E S W N E S W"
}'
```

**Note:** I implemented validation initially at the API layer and later moved it to MockRobot. Just keeping the logic for demonstration.

### Get the robot state
```bash
curl --location 'http://localhost:8080/state'
```

### Cancel a task
```bash
curl --location --request POST 'http://localhost:8080/cancel/<taskid>'
```


## Commands
- `N` - Move North
- `S` - Move South  
- `E` - Move East
- `W` - Move West

Commands are space-separated (e.g., "N E S W").


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