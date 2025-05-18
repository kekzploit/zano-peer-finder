# Zano Peer Finder

A powerful tool for discovering and monitoring nodes in the Zano network. This application provides real-time node tracking, geographical visualization, and network analysis capabilities.

## Features

- Real-time node discovery and monitoring
- Interactive world map visualization
- Detailed node information including:
  - Online/offline status
  - Geographic location
  - Network information (ISP, AS, Organization)
  - Last seen timestamp
- Node scanning capabilities
- Search and filter functionality
- Dark mode interface
- Mobile-responsive design

## Prerequisites

- Go 1.16 or higher
- SQLite3
- Nmap (for node scanning functionality)
- Make (optional, for using Makefile)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/zano-peer-finder.git
cd zano-peer-finder
```

2. Install dependencies:
```bash
make deps
```

## Building

### Using Make

The easiest way to build the project is using the provided Makefile:

```bash
make build
```

This will create a binary in the `bin` directory.

### Manual Build

If you prefer to build manually:

```bash
go build -o bin/peer-finder cmd/peer-finder/main.go
```

## Running

### Using Make

```bash
make run
```

### Manual Run

```bash
./bin/peer-finder
```

The application will start and be available at `http://localhost:8080`

## Usage

1. Open your web browser and navigate to `http://localhost:8080`
2. The dashboard will show:
   - Total number of nodes
   - Online/offline node counts
   - New nodes discovered in the last 24 hours
   - Detailed node table with search and filter options

### Map View

1. Click on the "Map View" link in the navigation
2. The map will display all discovered nodes with:
   - Green markers for online nodes
   - Red markers for offline nodes
   - Click on markers to view node details
   - Use filters to show/hide different node types

### Node Scanning

1. Click the scan button (🔍) next to any node in the table
2. The scan will check:
   - Open ports
   - Running services
   - Operating system information
   - Network details

## Development

### Project Structure

```
zano-peer-finder/
├── cmd/
│   └── peer-finder/
│       └── main.go
├── internal/
│   ├── database/
│   └── ipinfo/
├── static/
│   ├── app.js
│   ├── index.html
│   ├── map.html
│   └── style.css
├── Makefile
└── README.md
```

### Available Make Commands

- `make deps` - Install dependencies
- `make build` - Build the application
- `make run` - Run the application
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make lint` - Run linters

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

The GNU General Public License is a free, copyleft license that ensures the software remains free and open source. This means that:
- You are free to use, modify, and distribute the software
- You must make any modifications available under the same license
- You must include the original copyright notice and license
- You must state significant changes made to the software

## Acknowledgments

- Zano Network Team
- OpenStreetMap for map tiles
- Leaflet.js for map functionality