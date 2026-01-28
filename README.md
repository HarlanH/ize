# Ize - Search UI & Algorithm Testbed

A minimal testbed for exploring search result UI designs and algorithm experiments, built on top of Algolia.

## Overview

This is a demo application focused on e-commerce-style search with images. It consists of:
- **Backend**: Go HTTP server with an `ize` module for processing search results
- **Frontend**: Vue 3 + TypeScript + Vite single-page application

## Name & Trademark

The "IZE" trademark for software (originally owned by Persoft, Inc. and later Retrieval Dynamics) is currently abandoned according to USPTO records. The name is technically available for use in the software space, though the original IZE was a legendary 1980s search tool whose brand heritage remains part of computing history.

## Architecture

- Backend exposes HTTP APIs (`/api/search` and `/api/ripper`) that query Algolia and process results through the `ize` module
- Frontend provides a three-region layout: search bar at top, refinement panel on left with tabs for "Faceted Search" and "RIPPER", results grid on right
- The `ize` module hosts algorithm experiments including:
  - **RIPPER**: A greedy faceting algorithm that selects top 5 facet values maximizing information gain

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
  "query": "search terms",
  "facetFilters": [["category:Electronics"], ["brand:Apple"]]
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
  ],
  "facets": {
    "category": {
      "Electronics": 42,
      "Clothing": 18
    },
    "brand": {
      "Apple": 15,
      "Samsung": 12
    }
  }
}
```

### POST /api/ripper

RIPPER faceting endpoint that uses a greedy algorithm to select the top 5 facet values maximizing information gain. Requests 100 hits from Algolia for better coverage.

**Request:**
```json
{
  "query": "search terms",
  "facetFilters": [["category:Electronics"]]
}
```

**Response:**
```json
{
  "groups": [
    {
      "facetName": "brand",
      "facetValue": "Samsung",
      "items": [...],
      "count": 15
    },
    {
      "facetName": "category",
      "facetValue": "Phones",
      "items": [...],
      "count": 12
    }
  ],
  "otherGroup": [...]
}
```

**Algorithm Details:**
- Uses entropy-based information gain to select facets
- Greedily selects up to 5 facet values
- Items are removed from consideration once assigned to a group
- Minimum group size: 5% of total items (minimum 2)
- "Other" group contains items not matching any selected facet values

### GET /health

Health check endpoint.

## Features

### Faceted Search
- Traditional faceted navigation with checkboxes
- Multiple values within a facet use OR logic
- Multiple facets combine with AND logic
- Facet counts update based on current filters

### RIPPER Algorithm
- Greedy faceting algorithm inspired by RIPPER (1995)
- Selects top 5 facet values that maximize information gain
- Clicking a facet value applies the filter and re-runs RIPPER on filtered results
- "Other" group shows items not matching any selected facets
- Clicking "Other" applies negated filters to show only those items

## Development

- The `ize` module in `backend/internal/ize` hosts algorithm experiments
  - `ripper.go`: RIPPER faceting algorithm implementation
  - `ize.go`: Default pass-through processor
- The left panel provides tabbed interface for different faceting approaches
- Results are displayed in a grid on the right side
- Debug logging is available for RIPPER algorithm (see `backend/DEBUGGING.md`)

For detailed development guidelines, code standards, and AI agent workflows, see [AGENTS.md](AGENTS.md).
