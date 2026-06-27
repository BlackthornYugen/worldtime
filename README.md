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

## 1. Local Development
To run the server locally:
```bash
go run .
```
Open `http://localhost:8080` in your browser or run:
```bash
curl -sL http://localhost:8080/Waterloo/London/
```

---

## 2. Container Build
To package the app for production, build the static binary inside a multi-stage Docker container targetting `linux/amd64`:
```bash
docker build --provenance=false -t worldtime:latest .
```
> [!IMPORTANT]
> The `--provenance=false` flag is critical to disable build attestations. Without this, the generated manifest list will fail when imported directly into standard `containerd` environments.

---

## 3. Remote Node Transfer
If your Kubernetes cluster runs on a remote node (e.g., `ubuntu.jskw.dev`) and you import images locally rather than pushing to a registry:

1. **Save the image** as a tarball:
   ```bash
   docker save worldtime:latest | pv > latest.tar
   ```

2. **Transfer** it to the remote cluster node:
   ```bash
   rsync -vaz latest.tar ubuntu.jskw.dev:/tmp/latest.tar
   ```

3. **Import** the image directly into containerd's Kubernetes namespace (`k8s.io`):
   ```bash
   ssh ubuntu.jskw.dev "sudo ctr --namespace k8s.io images import /tmp/latest.tar"
   ```

---

## 4. Kubernetes Deployment

We provide a pre-configured [k8s.yaml](./k8s.yaml) manifest containing both the Deployment and Service.

### Option A: Using the Manifest File (Recommended)
This approach sets up resource limits and configures `imagePullPolicy: IfNotPresent` to prevent Kubernetes from trying to pull the image from Docker Hub.
```bash
kubectl apply -f k8s.yaml
```

### Option B: Using Ad-hoc `kubectl` Commands
If you prefer deploying purely via the command-line interface:

1. **Create the Deployment**:
   ```bash
   kubectl create deployment worldtime --image=worldtime:latest --port=8080
   ```

2. **Patch the image pull policy** (required because the `:latest` tag defaults to `Always`, which fails for locally imported images):
   ```bash
   kubectl patch deployment worldtime -p '{"spec":{"template":{"spec":{"containers":[{"name":"worldtime","imagePullPolicy":"IfNotPresent"}]}}}}'
   ```

3. **Expose the Deployment** as a service:
   ```bash
   kubectl expose deployment worldtime --port=80 --target-port=8080
   ```

---

## Standalone CLI Mode Usage

You can run the binary directly in your terminal to print comparison timelines directly to stdout and exit:

```bash
# Start the web server (Default behavior when no arguments are provided)
go run .

# Force start the web server
go run . --server

# Compare Waterloo and Bangalore (Kolkata) using the first city as focus
go run . Waterloo Bangalore

# Compare Waterloo and Bangalore (Kolkata) focusing on Bangalore
go run . -f Bangalore Waterloo Bangalore

# Disable colors in stdout
go run . -c never Waterloo Bangalore
```

---

## CI/CD GitHub Action Setup

The repository contains a GitHub Actions workflow in `.github/workflows/publish.yml` that builds and publishes the statically compiled Docker container image on push to the `main` branch.

It publishes to **GitHub Container Registry (GHCR)** at `ghcr.io/${{ github.repository }}/worldtime`. 

Authentication is fully automated using the built-in `${{ secrets.GITHUB_TOKEN }}` and `${{ github.actor }}` permissions. No manual secrets configuration is required.


