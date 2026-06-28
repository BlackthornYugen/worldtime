# WorldTime Clock & Timezone Comparison Service

A fast, self-contained world clock and timezone comparison service (inspired by World Time Buddy) built in Go. It features a modern glassmorphic web/mobile dashboard, automatic timezone redirects, and a clean ASCII timeline comparison for terminal (`curl`) requests.

---

## Technical Features
- **Fuzzy City Lookup**: Resolves path-based city names (e.g. `/Waterloo/London/`) using an embedded 33,000+ city database from GeoNames. Re-prioritizes exact name matches over alternate names to avoid incorrect matching. For CLI requests, the HTTP `301 Moved Permanently` redirect contains the target page's body so `curl` displays the comparison table immediately without requiring `--location`.
- **Dynamic Format Negotiation**: Automatically detects requests from command line tools (`curl`, `wget`, PowerShell `Invoke-WebRequest`, `HTTPie`) or requests with `Accept: text/plain` headers, returning the terminal interface instead of the SPA.
- **Terminal ANSI Color-Coding**: Visually colors hourly cells to categorize parts of the day (Yellow for work hours, Light Blue/Cyan for transition hours, and Dark Blue for night).
  * **Brackets Removal & Red Separator Line**: When colors are enabled, the brackets `[ ]` around the current hour are removed. Instead, the single vertical line `│` directly to the right of the current hour is colored **red**, highlighting the current hour boundary.
  * **Hour-Only Formatting & Grid Shifting**: Timelines show hours only (e.g. `15` instead of `15:00`) centered inside columns. For timezones offset by 30 minutes, internal grid separators `│` shift to the right by 4 spaces. Outer start/end borders are omitted, and **25 hours** are displayed (prepending one hour in front) to ensure full column coverage across the standard 24-hour grid boundary. Label columns prepend the current local time (`HH:MM`) and show relative timezone offsets (e.g., `+9.5`) instead of absolute UTC offsets, removing date strings.
  > [!NOTE]
  > ANSI colors and red separators are only returned if the client's `Accept` header contains `*/*` (default CLI client behavior) or one of the custom ANSI mime-types: `text/x-ansi`, `text/ansi`, or `application/x-ansi`. Standard `text/plain` requests return uncolored plaintext, ensuring compatibility when piping output to files/log parsers.
- **Statically Compiled Container**: Bundles static web assets (`embed.FS`) and the IANA ZoneInfo database (`time/tzdata`) inside a bare `scratch` container image.


---

## Usage

### Via `curl`

You can fetch the time comparison directly in your terminal using `curl`:

```bash
curl https://time.jskw.dev/Waterloo/Campinas/York/Stuttgart/Bengaluru+as+Bangalor\?focus\=York
```

### Via Docker

You can run the pre-built container image. Note that you either need to use the `-t` flag with `docker run` or pass `--color always` to the container to get ANSI colors in the output.

```bash
docker run -t ghcr.io/blackthornyugen/worldtime:latest Waterloo Campinas York Stuttgart Bangalore
```

### Standalone CLI Mode

You can run the binary directly in your terminal to print comparison timelines directly to stdout and exit:

```bash
# Compare Waterloo and Bangalore (Kolkata) using the first city as focus
go run . Waterloo Bangalore

# Compare Waterloo and Bangalore (Kolkata) focusing on Bangalore
go run . -f Bangalore Waterloo Bangalore

# Disable colors in stdout
go run . -c never Waterloo Bangalore
```

### Local Server Development

To run the web server locally:

```bash
# Start the web server (Default behavior when no arguments are provided)
go run .

# Force start the web server
go run . --server
```

Open `http://localhost:8080` in your browser or test the API via `curl`:

```bash
curl -sL http://localhost:8080/Waterloo/London/
```

---

## CI/CD GitHub Action Setup

The repository contains a GitHub Actions workflow in `.github/workflows/publish.yml` that builds and publishes the statically compiled Docker container image on push to the `main` branch.

It publishes to **GitHub Container Registry (GHCR)** at `ghcr.io/${{ github.repository }}/worldtime`. 

Authentication is fully automated using the built-in `${{ secrets.GITHUB_TOKEN }}` and `${{ github.actor }}` permissions. No manual secrets configuration is required.


