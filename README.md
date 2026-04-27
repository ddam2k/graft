# Graft

**Graft** is a lightweight Docker registry tool specialized for air-gapped and offline deployment environments. It supports image building, modification, and efficient differential transfer without the need for a Linux environment or a Docker daemon.

## Key Features

- **Optimized for Offline & Multi-platform:** Enable image building and manipulation in offline Windows environments (where WSL2 is difficult to set up) or macOS without any additional virtualization configurations.
- **Lightweight Daemon-less Build:** Runs independently in environments without Docker Engine. It supports builds from **scratch** images, allowing for the creation of ultra-lightweight containers.
- **Differential Extraction (Diff-Pull/Push):** No need to transfer entire images. Extract only the changed layers to minimize file size and maximize transfer efficiency.
- **Intuitive Image Modification:** Perform tasks such as setting environment variables, adding files, and changing entry points immediately without a Linux environment.

---

## Main Functions

### 1. Build (Offline & Multi-platform Build)
Build images using a `Dockerfile` without the Linux kernel or WSL2. This is ideal for simple image modifications and deployment preparation in offline environments.

* **Supported Dockerfile Instructions:**
  `FROM`, `COPY`, `ENV`, `WORKDIR`, `ENTRYPOINT`, `EXPOSE`, `CMD`
* **Scratch Build Support:** Capability to create ultra-lightweight images containing only the executable binary without a base image.

```bash
# Example of a Dockerfile-based build
graft build -f Dockerfile -t myregistry/myimage:latest --push -u user -p pass
```

### 2. Diff-Pull & Diff-Push (Efficient Transfer)
Drastically reduces the amount of data transferred when moving images to offline environments.

* **Diff-Pull:** Extracts only the differences (new layers) between two image tags and saves them as a `.tar` file.
* **Diff-Push:** Merges the extracted differential layer file into the target registry.

```bash
# 1. Extract only the changed layers (External Network)
graft diff-pull --base v1.0 --target v1.1 myregistry/myimage -f diff.tar

# 2. Move only the extracted file offline (via USB, etc.)

# 3. Push only the changes (Internal Network)
graft diff-push diff.tar myregistry/myimage:v1.1
```

### 3. Pull & Push
Supports standard Docker image Pull/Push and exporting/importing via `.tar` files.

---

## Use Cases

### 1. Air-gapped Windows/macOS Development
In secure environments where installing WSL2 or Docker Desktop is restricted, you can immediately containerize binaries built with Go, Rust, etc., and deploy them to an internal registry.

### 2. Efficient Updates for Large Images
When an update is required for a server already in operation, use `diff-pull` to quickly transfer only the changed layers instead of re-transferring an entire image several gigabytes in size.

### 3. Lightweight CI/CD Integration
In CI environments such as GitLab Runner, Jenkins, and Tekton, you can build simple Dockerfiles and push to repositories without a Docker daemon or root privileges. While limited in functionality compared to full Docker, it provides a lightweight alternative for basic container build workflows.

---

---

## Installation

### Prerequisites

- **Go 1.26.1** or higher

### Install from Source

#### Using `go install` (Recommended)

```bash
go install github.com/ddam2k/graft@latest
```

The `graft` binary will be installed to your `$GOPATH/bin` directory (usually `~/go/bin``). Make sure this directory is in your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Build from Source

```bash
# Clone the repository
git clone https://github.com/ddam2k/graft.git
cd graft

# Build the binary
go build -o graft .

# (Optional) Install to your PATH
mv graft /usr/local/bin/
```

### Verify Installation

```bash
graft --version
```

---

## Requirements

- **Go 1.26.1** or higher (for building from source)
- **No Docker Daemon Required:** Runs standalone without Docker Engine or virtualization layers.

## Dependencies

- [google/go-containerregistry](https://github.com/google/go-containerregistry): Core library for registry operations.
- [spf13/cobra](https://github.com/spf13/cobra): CLI interface implementation.