# Ize - Search UI & Algorithm Testbed

A minimal testbed for exploring search result UI designs and algorithm experiments, built on top of Algolia.

## Overview

This is a demo application focused on e-commerce-style search with images. It consists of:
- **Backend**: Go HTTP server with an `ize` module for processing search results
- **Frontend**: Vue 3 + TypeScript + Vite single-page application

## Architecture

- Backend exposes a simple HTTP API (`/api/search`) that queries Algolia and processes results through the `ize` module
- Frontend provides a three-region layout: search bar at top, empty refinement panel on left, results grid on right
- The `ize` module currently acts as a pass-through but is designed to host re-ranking and other algorithm experiments

## Setup

### Prerequisites

- Go 1.21+ 
- Node.js 18+ and npm/pnpm/yarn
- An Algolia account with an index containing `name`, `description`, and `image` fields

### Configuration

Create a `config.json` file in the `backend/` directory (you can copy `config.json.example` as a template):

```json
{
  "algolia_app_id": "your-app-id",
  "algolia_api_key": "your-api-key",
  "algolia_index_name": "your-index-name",
  "port": "8080"
}
```

Alternatively, set environment variables:
- `ALGOLIA_APP_ID`
- `ALGOLIA_API_KEY`
- `ALGOLIA_INDEX_NAME`
- `PORT` (defaults to 8080)

### Running the Backend

```bash
cd backend
go mod tidy  # Install dependencies
go run cmd/server/main.go
```

The server will start on `http://localhost:8080` (or your configured port).

### Running the Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend will start on `http://localhost:5173` (or Vite's default port).

### Testing

Run backend tests:
```bash
cd backend
go test ./...
```

## API

### POST /api/search

Search endpoint that queries Algolia and processes results through `ize`.

**Request:**
```json
{
  "query": "search terms"
}
```

**Response:**
```json
{
  "hits": [
    {
      "id": "object-id",
      "name": "Product Name",
      "description": "Product description",
      "image": "https://example.com/image.jpg"
    }
  ]
}
```

### GET /health

Health check endpoint.

## Development

- The `ize` module in `backend/internal/ize` is where algorithm experiments will be added
- The left panel in the frontend is reserved for future refinement controls
- Results are displayed in a grid on the right side

For detailed development guidelines, code standards, and AI agent workflows, see [AGENTS.md](AGENTS.md).
