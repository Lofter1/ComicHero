# Metron Proxy

Used to proxy metron calls and bypass CORS restrictions

## Setup

### Environment Variables

METRON_PROXY_URL = The full URL including protocol and port
POCKETBASE_URL = Full URL to Pocketbase, needed for Auth requests
METRON_PASSWORD = Metron Password for API login
METRON_USERNAME = Metron Username for API login

### Optional Environment Variables
REDIS_ADDR = Location of the Redis server for caching. If not set, uses in memory cache
