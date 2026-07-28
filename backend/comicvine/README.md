# Comic Vine API Go Client

A comprehensive, easy-to-use Go client for the [Comic Vine API](https://comicvine.gamespot.com/api/).

## Features

- ✅ Complete API coverage (all resource types)
- ✅ Simple, intuitive interface
- ✅ Strong typing with full model definitions
- ✅ Query builder for complex filters
- ✅ Context support for cancellation
- ✅ Automatic retry with backoff
- ✅ Rate limiting support
- ✅ Comprehensive error handling
- ✅ Well-tested
- ✅ Reusable as a standalone package

## Installation

```bash
go get github.com/yourusername/comicvine
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/yourusername/comicvine"
)

func main() {
    // Create a new client
    client := comicvine.NewClient("YOUR_API_KEY")
    
    // Search for characters
    resp, err := client.Characters().Search(context.Background(), "Spider-Man", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, character := range resp.Results {
        fmt.Printf("Name: %s, Publisher: %s\n", 
            character.Name, 
            character.Publisher.Name)
    }
}
```

## Usage Examples

### Basic Operations

```go
// Get a specific character
character, err := client.Characters().GetByID(ctx, 1443)

// Get all issues with pagination
issues, err := client.Issues().List(ctx, &comicvine.ListOptions{
    Limit:  50,
    Offset: 0,
})

// Search volumes
resp, err := client.Volumes().Search(ctx, "Batman", &comicvine.SearchOptions{
    Limit: 10,
    Resources: []string{"publisher", "first_issue"},
})
```

### Using Query Builder

```go
// Build complex queries
query := comicvine.NewQuery().
    Filter("name", "Spider").
    Filter("publisher", "31"). // Marvel
    Sort("date_added", "desc").
    Limit(20).
    Fields("name", "image", "deck")

characters, err := client.Characters().Get(ctx, query)
```

### Working with Resources

The client provides access to all Comic Vine resources:

- Characters
- Concepts
- Episodes
- Issues
- Locations
- Movies
- Objects
- Origins
- Powers
- Publishers
- Series
- Story Arcs
- Teams
- Volumes
- Promos
- Videos

Each resource supports:
- `GetByID()` - Fetch a single resource by ID
- `List()` - List resources with pagination
- `Search()` - Search resources by name
- `Get()` - Advanced querying with filters

### Custom HTTP Client

```go
// Use a custom HTTP client
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}
client := comicvine.NewClientWithHTTP("YOUR_API_KEY", httpClient)
```

## Configuration

```go
client := comicvine.NewClient("api-key",
    comicvine.WithBaseURL("https://comicvine.gamespot.com/api"),
    comicvine.WithRateLimit(2), // requests per second
    comicvine.WithRetry(3),     // max retry attempts
    comicvine.WithUserAgent("MyApp/1.0"),
)
```

## Error Handling

```go
character, err := client.Characters().GetByID(ctx, id)
if err != nil {
    var apiErr *comicvine.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API Error: %s (Status: %d)\n", 
            apiErr.Message, apiErr.StatusCode)
    }
}
```

## License

MIT
