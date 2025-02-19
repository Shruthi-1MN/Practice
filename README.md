To use this application:

1 Build and start the services:
```
docker-compose up --build
```
2 Get a JWT token:

```
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

3 Use the API with the token:

```
curl -X POST http://localhost:8080/multiply \
  -H "Content-Type: application/json" \
  -H "Authorization: <JWT_TOKEN>" \
  -d '{"a":5,"b":6}'
```

Features included:

1 REST API endpoints for math operations

2 JWT authentication

3 MongoDB integration for operation logging

4 Prometheus metrics

5 Structured logging

6 Error handling and recovery

7 Docker support

8 Concurrent request handling

To test the application, create test files with _test.go suffix and run:

```

go test -v ./...
```
This implementation provides a solid foundation for a scalable, monitored, and secure microservice architecture
