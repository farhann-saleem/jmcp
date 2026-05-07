#!/usr/bin/env bash
set -euo pipefail

REPO="farhann-saleem/jmcp"
API_ROOT="https://api.github.com/repos/${REPO}"
RELEASES_ROOT="https://github.com/${REPO}/releases/download"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

fetch_latest_tag() {
  curl -fsSL "${API_ROOT}/releases/latest" | grep -o '"tag_name": *"[^"]*"' | head -n1 | cut -d'"' -f4
}

pick_archive() {
  local os="$1"
  local arch="$2"
  if [[ "$os" == "windows" ]]; then
    echo "jmcp_${os}_${arch}.zip"
  else
    echo "jmcp_${os}_${arch}.tar.gz"
  fi
}

install_dir() {
  if [[ -w /usr/local/bin ]]; then
    echo "/usr/local/bin"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    echo "/usr/local/bin"
    return
  fi

  mkdir -p "${HOME}/bin"
  echo "${HOME}/bin"
}

extract_binary() {
  local archive="$1"
  local temp_dir="$2"
  case "$archive" in
    *.tar.gz)
      tar -xzf "$archive" -C "$temp_dir"
      ;;
    *.zip)
      need_cmd unzip
      unzip -q "$archive" -d "$temp_dir"
      ;;
    *)
      echo "unsupported archive format: $archive" >&2
      exit 1
      ;;
  esac
}

main() {
  need_cmd curl
  need_cmd tar

  local os arch tag archive_name url tmpdir target_dir binary_path install_path
  os="$(detect_os)"
  arch="$(detect_arch)"
  tag="$(fetch_latest_tag)"

  if [[ -z "$tag" ]]; then
    echo "failed to resolve latest release from GitHub" >&2
    exit 1
  fi

  archive_name="$(pick_archive "$os" "$arch")"
  url="${RELEASES_ROOT}/${tag}/${archive_name}"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  echo "Downloading ${url}"
  curl -fsSL "$url" -o "${tmpdir}/${archive_name}"
  extract_binary "${tmpdir}/${archive_name}" "$tmpdir"

  binary_path="$(find "$tmpdir" -type f \( -name jmcp -o -name jmcp.exe \) | head -n1)"
  if [[ -z "$binary_path" ]]; then
    echo "downloaded archive did not contain jmcp binary" >&2
    exit 1
  fi

  target_dir="$(install_dir)"
  install_path="${target_dir}/jmcp"

  if [[ "$target_dir" == "/usr/local/bin" && ! -w "$target_dir" ]]; then
    echo "Installing to ${install_path} with sudo"
    sudo install -m 0755 "$binary_path" "$install_path"
  else
    mkdir -p "$target_dir"
    install -m 0755 "$binary_path" "$install_path"
  fi

  echo "Installed jmcp to ${install_path}"
  if [[ "$target_dir" == "${HOME}/bin" ]]; then
    echo "Ensure ${HOME}/bin is in your PATH."
  fi
}

main "$@"
