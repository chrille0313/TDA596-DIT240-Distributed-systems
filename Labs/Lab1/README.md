## Lab 1 – HTTP Server and Proxy

We implement a small HTTP/1.1 server and an HTTP proxy on top of a tiny custom HTTP library.  
Below is a brief overview of the project structure and how the code works.

### Repository Layout

- **`http/`**: Minimal HTTP abstractions used by both binaries
  - `server.go`: Connection handling, parsing requests, dispatching to handlers, building responses
  - `response.go`, `response_builder.go`, `status_code.go`, `header.go`, `method.go`, `error.go`: Supporting types for HTTP responses, headers, methods, status codes, and error handling.
- **`http_server/`**: File-based HTTP server (main program)
  - `http_server.go`: Starts the server and serves files from the `public/` directory using GET/POST.
  - `files.go`: Helper functions for reading/writing files and mapping URLs to filesystem paths.
  - `public/`: Static content served by the server (HTML, CSS, images, etc.).
  - `Dockerfile`: Container setup for running the HTTP server.
- **`proxy/`**: HTTP proxy (main program)
  - `proxy.go`: Listens for client connections, forwards requests to target servers, and relays responses.
  - `Dockerfile`: Container setup for running the proxy.

### How our solution works (high‑level)

- **HTTP library (`http/`)**

  - We expose a small `Server` type with a `Listen(address, maxConnections)` method and `Get`/`Post` handler registration.
  - Internally, `Listen`:
    - Creates a buffered channel `connections` with size `maxConnections` to **limit concurrent connections**.
    - Accepts TCP connections in a loop and starts a goroutine `handleConnection` for each, after putting it into the channel.
    - When a connection is done, `handleConnection` reads from the channel to free a slot.
  - `handleConnection`:
    - Uses `net/http.ReadRequest` to parse exactly one HTTP request from the TCP stream.
    - Dispatches to the registered handler based on the HTTP method (via a `map[HTTPMethod]RequestHandler`).
    - If parsing fails or no handler exists, we wrap it in an `HTTPError` and build a simple error response via `handleError`.
  - `ResponseBuilder` is a helper that handlers use to set status, headers, and body; `Build()` returns a `Response`, and `Response.Bytes()` formats it into raw HTTP bytes that we write to the connection.

- **HTTP file server (`http_server/`)**

  - `main` wires up two handlers on the shared `http.Server`:
    - **GET handler**:
      - Validates the URL path and resolves a content type with `contentTypeForPath`.
      - Reads the corresponding file from `./public` using `readFile`.
      - On success: sets the response body and a `Content-Type` header.
      - On error: maps file/OS errors to HTTP status codes via `statusFromFileError` (e.g. `os.ErrNotExist` → `404`, permission/invalid/unsupported content type → `400`).
    - **POST handler**:
      - Reads the request body and writes it to a file under `./public` via `writeFile`.
      - On success: returns **201 Created** without a body.
      - On error: again uses `statusFromFileError` for consistent error codes.
  - Both handlers rely on the shared `http.Server` infrastructure for connection handling and error responses.

- **HTTP proxy (`proxy/`)**
  - We register a single **GET handler** on a `http.Server` instance.
  - For each incoming request, `handleProxyRequest`:
    - Figures out the **target host** from `request.URL.Host`, falling back to `request.Host`, and defaulting the port to `:80` if none is present.
    - Opens a raw TCP connection to the target using `net.Dial("tcp", targetHost)`.
    - Strips the `Proxy-Connection` header and clears `request.RequestURI` so that the outgoing request is a valid origin-form request for `net/http`.
    - Writes the original request to the backend server over the TCP connection.
    - Reads the backend response using `net/http.ReadResponse`, fully consumes the body into memory with `io.ReadAll`, and copies:
      - the status code (`rb.Status(...)`)
      - the response body (`rb.Bytes(...)`)
      - and all headers (`rb.Header(...)`) into our own response.
  - If anything fails (no host, dial error, read error, etc.), we return an `HTTPError` with either **400 Bad Request** (no host) or **500 Internal Server Error**, which is converted to a simple error response by `handleError`.

### Building the Programs

From `Labs/Lab1`:

```bash
go build ./http_server
go build ./proxy
```

This will produce the binaries:

- `http_server/http_server`
- `proxy/proxy`

### Running the HTTP Server

From `Labs/Lab1`:

```bash
go run ./http_server
```

By default the server:

- Binds to host **`0.0.0.0`**
- Listens on port **`80`**
- Serves files from the local `public/` directory

You can override host and port:

```bash
go run ./http_server --host 127.0.0.1 --port 8080
# or using the short form for port
go run ./http_server -p 8080
```

Example request (assuming port 8080):

```bash
curl http://localhost:8080/index.html
```

#### Supported Operations

- **GET**: Returns a file from `public/` when the path and content type are supported.
- **POST**: Writes the request body to a file under `public/` and returns status **201 Created** on success.

Error handling examples:

- Non‑existing file → **404 Not Found**
- Invalid path/unsupported content type → **400 Bad Request**
- Other I/O errors → **500 Internal Server Error**

### Running the Proxy

From `Labs/Lab1`:

```bash
go run ./proxy
```

Default behaviour:

- Binds to host **`0.0.0.0`**
- Listens on port **`80`**
- Forwards incoming HTTP requests to the host specified in the request (`Host` header or URL).

Override host and port:

```bash
go run ./proxy --host 127.0.0.1 --port 8081
# or
go run ./proxy -p 8081
```

The first non‑flag argument can also be used as the port:

```bash
go run ./proxy 8081
```

To test the proxy, point a client at the proxy instead of directly at the server. For example, if your HTTP server is running on `localhost:8080`:

```bash
curl -x http://localhost:8081 http://localhost:8080/index.html
```

### Command‑Line Flags (Both Programs)

Both `http_server` and `proxy` accept:

- `--host` (string): Address to bind to (default: `0.0.0.0`)
- `--port` (string): Port to listen on (default: `80`)
- `-p` (string): Alias for `--port`
- `--maxConnections` (uint): Maximum number of concurrent connections (default: `10`)
