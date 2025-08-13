# Asnakech School Servers

A microservices-based education platform built with Go and Gin.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/asnakech-servers.git
   cd asnakech-servers
   ```

2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

3. Install dependencies:
   ```bash
   make install
   ```

### Running the Application

Start the development server:
```bash
make dev
```

The server will start on `http://localhost:8080` by default.

## Project Structure

```
.
├── cmd/                  # Main applications for this project
│   └── api/              # API server entry point
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── handlers/         # HTTP request handlers
│   └── middleware/       # Custom middleware
├── .env.example          # Example environment variables
├── .gitignore           # Git ignore file
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
└── Makefile             # Common development tasks
```

## Available Commands

- `make build` - Build the application
- `make run` - Build and run the application
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make deps` - Install/update dependencies
- `make dev` - Run in development mode with hot-reload

## API Endpoints

- `GET /health` - Health check endpoint

## Environment Variables

- `PORT` - Port to run the server on (default: 8080)
- `ENV` - Environment (development, production, etc.)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
