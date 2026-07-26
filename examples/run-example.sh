#!/usr/bin/env bash

# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# Interactive runner for Packer Artifactory examples.
#
# Works when executed under bash or zsh:
#   ./examples/run-example.sh
#   ./examples/run-example.sh --dev
#   bash examples/run-example.sh --dev
#
# At each prompt: number to select, b=back, h=help, q=quit.
#
# Options:
#   -d, --dev   Start in development mode
#   -h, --help  Show usage


set -euo pipefail

# zsh arrays are 1-based by default; match bash indexing for this script.
if [ -n "${ZSH_VERSION:-}" ]; then
  setopt KSH_ARRAYS
fi

script_path() {
  if [ -n "${BASH_VERSION:-}" ]; then
    printf '%s\n' "${BASH_SOURCE[0]}"
  elif [ -n "${ZSH_VERSION:-}" ]; then
    # shellcheck disable=SC2296
    printf '%s\n' "${(%):-%x}"
  else
    printf '%s\n' "$0"
  fi
}

SCRIPT_DIR="$(cd "$(dirname "$(script_path)")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
EXAMPLES_DIR="${SCRIPT_DIR}"
RESOLVE_ISO="${ROOT_DIR}/scripts/resolve-debian-iso.sh"

# Set by --dev / -d (or ARTIFACTORY_DEV_MODE=1).
DEV_MODE="${ARTIFACTORY_DEV_MODE:-0}"

COLOR_GREEN=$'\033[0;32m'
COLOR_YELLOW=$'\033[1;33m'
COLOR_BLUE=$'\033[0;34m'
COLOR_RED=$'\033[0;31m'
COLOR_RESET=$'\033[0m'

MENU_BACK='__back__'
MENU_QUIT='__quit__'
MENU_HELP='__help__'

is_tty() {
  # Menus run in $(...); stdout is a pipe — treat stderr or /dev/tty as the UI.
  [ -t 2 ] || [ -c /dev/tty ]
}

# Menus run inside $(...); zsh redirects that stdin from /dev/null, and some
# terminals deliver Enter as CR. Always read the controlling TTY and strip CR.
ensure_tty() {
  if [ -c /dev/tty ]; then
    # Restore cooked mode + CR→NL so Enter submits the line.
    stty sane < /dev/tty 2>/dev/null || true
  fi
}

read_tty() {
  REPLY=""
  # Prefer the controlling TTY so menus work inside $(...) under zsh.
  # Fall back to stdin when /dev/tty is unavailable (e.g. some CI pipes).
  if [ -c /dev/tty ] && IFS= read -r REPLY < /dev/tty 2>/dev/null; then
    :
  else
    # shellcheck disable=SC2162
    IFS= read -r REPLY || true
  fi
  REPLY="${REPLY//$'\r'/}"
}

clear_screen() {
  if [ -c /dev/tty ]; then
    clear >/dev/tty 2>/dev/null || printf '\033[H\033[2J' >/dev/tty
  elif [ -t 2 ]; then
    clear >&2 2>/dev/null || printf '\033[H\033[2J' >&2
  fi
}

print_header() {
  local line_width=72
  local title="Packer Plugin for Artifactory"
  local subtitle="Examples"
  local mode="Development Mode"
  local pad_title pad_subtitle pad_mode

  pad_title=$(( (line_width - ${#title}) / 2 ))
  pad_subtitle=$(( (line_width - ${#subtitle}) / 2 ))
  pad_mode=$(( (line_width - ${#mode}) / 2 ))

  printf '\n' >&2
  printf "${COLOR_BLUE}%*s%s${COLOR_RESET}\n" "${pad_title}" '' "${title}" >&2
  printf "${COLOR_GREEN}%*s%s${COLOR_RESET}\n" "${pad_subtitle}" '' "${subtitle}" >&2
  if [ "${DEV_MODE}" = "1" ]; then
    printf "${COLOR_RED}%*s%s${COLOR_RESET}\n" "${pad_mode}" '' "${mode}" >&2
  fi
  printf '\n' >&2
}

print_nav_hint() {
  local allow_back="$1"

  if [ "${allow_back}" = "1" ]; then
    printf '(%bh%b) Help   (%bb%b) Back   (%bq%b) Quit\n\n' \
      "${COLOR_BLUE}" "${COLOR_RESET}" \
      "${COLOR_GREEN}" "${COLOR_RESET}" \
      "${COLOR_RED}" "${COLOR_RESET}" >&2
  else
    printf '(%bh%b) Help   (%bq%b) Quit\n\n' \
      "${COLOR_BLUE}" "${COLOR_RESET}" \
      "${COLOR_RED}" "${COLOR_RESET}" >&2
  fi
}

print_invalid() {
  local count="$1"

  printf '\n' >&2
  printf '%bInvalid Selection:%b Enter a number between 1 and %s.\n' \
    "${COLOR_YELLOW}" "${COLOR_RESET}" "${count}" >&2
  printf '\n' >&2
}

show_interactive_help() {
  clear_screen
  print_header
  usage >&2
  printf '\n' >&2
  printf '(%bh%b) Help   (%bb%b) Back   (%bq%b) Quit\n\n' \
    "${COLOR_BLUE}" "${COLOR_RESET}" \
    "${COLOR_GREEN}" "${COLOR_RESET}" \
    "${COLOR_RED}" "${COLOR_RESET}" >&2
  printf 'Press %bEnter%b to continue.\n' "${COLOR_GREEN}" "${COLOR_RESET}" >&2
  read_tty || true
}

# Returns 0 when the caller should re-show the same menu (help).
handle_menu_meta() {
  case "$1" in
    "${MENU_HELP}")
      show_interactive_help
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

# prompt_menu TITLE ALLOW_BACK [SKIP_CLEAR=0] LABEL1 VALUE1 ...
# Prints prompts on stderr; prints the selected VALUE, MENU_BACK, or MENU_QUIT on stdout.
# When SKIP_CLEAR is 1, keep the current screen (e.g. summary already shown).
# No defaults — Enter with an empty selection is invalid.
prompt_menu() {
  local title="$1"
  local allow_back="$2"
  local skip_clear="${3:-0}"
  shift 2
  if [ "${skip_clear}" = "0" ] || [ "${skip_clear}" = "1" ]; then
    shift 1
  else
    skip_clear=0
  fi
  local labels=()
  local values=()
  local i=1
  local count=0
  local input=""

  while [ "$#" -gt 0 ]; do
    labels+=("$1")
    values+=("$2")
    shift 2
    count=$((count + 1))
  done

  if [ "${skip_clear}" != "1" ]; then
    clear_screen
    print_header
  fi

  case "${title}" in
    *\?)
      printf '%s\n\n' "${title}" >&2
      ;;
  esac

  i=1
  while [ "${i}" -le "${count}" ]; do
    printf '%s: %s\n' "${i}" "${labels[$((i - 1))]}" >&2
    i=$((i + 1))
  done
  printf '\n' >&2
  print_nav_hint "${allow_back}"

  while true; do
    case "${title}" in
      *\?)
        ;;
      *)
        printf 'Select %s: ' "${title}" >&2
        ;;
    esac
    read_tty || true
    input="${REPLY:-}"

    if [ -z "${input}" ]; then
      print_invalid "${count}"
      continue
    fi

    case "${input}" in
      [qQ])
        printf '%s\n' "${MENU_QUIT}"
        return 0
        ;;
      [bB])
        if [ "${allow_back}" = "1" ]; then
          printf '%s\n' "${MENU_BACK}"
          return 0
        fi
        printf '\n' >&2
        printf '%bAlready at the first step.%b\n\n' \
          "${COLOR_YELLOW}" "${COLOR_RESET}" >&2
        continue
        ;;
      [hH])
        printf '%s\n' "${MENU_HELP}"
        return 0
        ;;
    esac

    case "${input}" in
      *[!0-9]*)
        print_invalid "${count}"
        continue
        ;;
    esac

    if [ "${input}" -ge 1 ] && [ "${input}" -le "${count}" ]; then
      printf '%s\n' "${values[$((input - 1))]}"
      return 0
    fi

    print_invalid "${count}"
  done
}

# Basic examples use Packer HCL defaults for URL/user/pass/repo.
# API key / access token examples still need those secrets in the environment.
artifactory_credentials_set() {
  case "${EXAMPLE:-}" in
    basic|basic-with-properties)
      return 0
      ;;
    api-key)
      [ -n "${ARTIFACTORY_API_KEY:-}" ]
      return
      ;;
    access-token)
      [ -n "${ARTIFACTORY_TOKEN:-}" ]
      return
      ;;
  esac

  if [ -n "${ARTIFACTORY_API_KEY:-}" ] || [ -n "${ARTIFACTORY_TOKEN:-}" ]; then
    return 0
  fi
  if [ -n "${ARTIFACTORY_USERNAME:-}" ] && [ -n "${ARTIFACTORY_PASSWORD:-}" ]; then
    return 0
  fi
  return 1
}

credentials_missing_message() {
  case "${EXAMPLE:-}" in
    api-key)
      printf '%s\n' "Artifactory API Key Is Not Set."
      ;;
    access-token)
      printf '%s\n' "Artifactory Access Token Is Not Set."
      ;;
    *)
      printf '%s\n' "Artifactory Credentials Are Not Set."
      ;;
  esac
}

credentials_confirm_prompt() {
  case "${EXAMPLE:-}" in
    api-key)
      printf '%s\n' "Set Artifactory API Key? (y/n) "
      ;;
    access-token)
      printf '%s\n' "Set Artifactory API Token? (y/n) "
      ;;
    *)
      printf '%s\n' "Configure Credentials? (y/n) "
      ;;
  esac
}

ensure_artifactory_env() {
  local setup_rc=0
  local missing_msg=""
  local confirm_prompt=""

  if artifactory_credentials_set; then
    return 0
  fi

  missing_msg="$(credentials_missing_message)"
  confirm_prompt="$(credentials_confirm_prompt)"

  while true; do
    clear_screen
    print_header
    printf '%b%s%b\n\n' "${COLOR_YELLOW}" "${missing_msg}" "${COLOR_RESET}" >&2
    print_nav_hint 1

    while true; do
      printf '%s' "${confirm_prompt}" >&2
      read_tty || true
      case "${REPLY:-}" in
        [yY])
          set +e
          # shellcheck disable=SC1091
          source "${ROOT_DIR}/scripts/setup-packer-env.sh"
          setup_rc=$?
          set -e
          case "${setup_rc}" in
            0)
              if artifactory_credentials_set; then
                return 0
              fi
              # Incomplete setup — re-show this screen.
              break
              ;;
            1)
              # Back from auth method — re-show this screen.
              break
              ;;
            *)
              return 2
              ;;
          esac
          ;;
        [nN] | [bB])
          return 1
          ;;
        [qQ])
          return 2
          ;;
        [hH])
          show_interactive_help
          clear_screen
          print_header
          printf '%b%s%b\n\n' "${COLOR_YELLOW}" "${missing_msg}" "${COLOR_RESET}" >&2
          print_nav_hint 1
          ;;
        *)
          printf '\n' >&2
          printf '%bInvalid Selection:%b Enter %by%b or %bn%b.\n\n' \
            "${COLOR_YELLOW}" "${COLOR_RESET}" \
            "${COLOR_GREEN}" "${COLOR_RESET}" \
            "${COLOR_RED}" "${COLOR_RESET}" >&2
          ;;
      esac
    done
  done
}

example_label() {
  case "$1" in
    basic) printf '%s\n' "Basic Authentication" ;;
    basic-with-properties) printf '%s\n' "Basic Authentication with Properties" ;;
    api-key) printf '%s\n' "API Key Authentication" ;;
    access-token) printf '%s\n' "Access Token Authentication" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

platform_label() {
  case "$1" in
    desktop) printf '%s\n' "VMware Desktop Hypervisors" ;;
    vsphere) printf '%s\n' "VMware vSphere" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

format_label() {
  case "$1" in
    ova) printf '%s\n' "OVA" ;;
    ovf) printf '%s\n' "OVF" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

# Host arch for desktop builds (macOS, Linux, WSL).
detect_host_arch() {
  local machine
  machine="$(uname -m 2>/dev/null || true)"
  case "${machine}" in
    arm64|aarch64) printf '%s\n' "arm64" ;;
    x86_64|amd64) printf '%s\n' "amd64" ;;
    *)
      printf '%berror:%b unsupported host architecture: %s\n' \
        "${COLOR_RED}" "${COLOR_RESET}" "${machine:-unknown}" >&2
      return 1
      ;;
  esac
}

print_summary() {
  clear_screen
  print_header

  printf '%bReady To Build%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}" >&2
  printf 'Example:          %s\n' "$(example_label "${EXAMPLE}")" >&2
  printf 'Platform:         %s\n' "$(platform_label "${PLATFORM}")" >&2
  printf 'Architecture:     %s\n' "${ARCH}" >&2
  printf 'Artifact Format:  %s\n' "$(format_label "${FORMAT}")" >&2
  printf '\n' >&2
  printf '%bThe following command will run:%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}" >&2
  printf '%bpacker build %s%b\n\n' "${COLOR_BLUE}" "${PACKER_ARGS[*]}" "${COLOR_RESET}" >&2
}

print_status() {
  printf '\n%b%s%b\n\n' "${COLOR_GREEN}" "$1" "${COLOR_RESET}"
}

usage() {
  cat <<'EOF'
Usage: run-example.sh [options]

Interactive Packer Artifactory example runner.

Options:
  -d, --dev     Start in development mode (make dev + builder plugins if missing).
  -h, --help    Show this help.

During menus:
  b             Go back (when available)
  h             Show this help
  q             Quit

Without development mode, runs packer init (registry install of all plugins).

Environment:
  ARTIFACTORY_DEV_MODE=1   Start in development mode (same as -d / --dev).
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --dev | -d)
        DEV_MODE=1
        shift
        ;;
      --help | -h)
        usage
        exit 0
        ;;
      *)
        printf '%berror:%b unknown option: %s\n\n' "${COLOR_RED}" "${COLOR_RESET}" "$1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done

  case "${DEV_MODE}" in
    1 | true | TRUE | yes | YES) DEV_MODE=1 ;;
    *) DEV_MODE=0 ;;
  esac
}

packer_plugin_installed() {
  local source="$1"
  packer plugins installed 2>/dev/null | grep -F "${source}" >/dev/null
}

ensure_builder_plugin() {
  local source="$1"

  if packer_plugin_installed "${source}"; then
    printf '  Packer plugin %s is already installed.\n' "${source}"
    return 0
  fi

  print_status "Installing Packer plugin ${source}..."
  packer plugins install "${source}"
}

prepare_plugins() {
  if [ "${DEV_MODE}" = "1" ]; then
    print_status "Building the plugin..."
    make -C "${ROOT_DIR}" dev

    # Packer validates every required_plugins entry, even with -only.
    printf '\n'
    ensure_builder_plugin "github.com/vmware/vmware"
    ensure_builder_plugin "github.com/vmware/vsphere"
    return 0
  fi

  print_status "Initializing plugins (packer init)..."
  (
    cd "${EXAMPLE_DIR}"
    packer init .
  )
}

build_packer_args() {
  local packer_args=()

  EXAMPLE_DIR="${EXAMPLES_DIR}/${EXAMPLE}"
  if [ ! -d "${EXAMPLE_DIR}" ]; then
    printf '\n%berror:%b example directory not found: %s\n\n' \
      "${COLOR_RED}" "${COLOR_RESET}" "${EXAMPLE_DIR}" >&2
    return 1
  fi

  packer_args+=(-force)

  case "${PLATFORM}" in
    desktop)
      ONLY="vmware-iso.example"
      packer_args+=(-only="${ONLY}")
      if [ "${ARCH}" = "amd64" ]; then
        packer_args+=(-var-file=pkrvars/desktop-amd64.pkrvars.hcl)
      fi
      ;;
    vsphere)
      ONLY="vsphere-iso.example"
      packer_args+=(-only="${ONLY}")
      packer_args+=(-var-file=pkrvars/vsphere.pkrvars.hcl)
      ;;
  esac

  packer_args+=(-var-file="pkrvars/export-${FORMAT}.pkrvars.hcl")
  packer_args+=(.)
  PACKER_ARGS=("${packer_args[@]}")
}

main() {
  local step=1
  local choice=""
  local confirm=""
  local setup_rc=0

  EXAMPLE=""
  PLATFORM=""
  ARCH=""
  FORMAT=""
  ONLY=""
  EXAMPLE_DIR=""
  PACKER_ARGS=()

  ensure_tty
  clear_screen

  while true; do
    case "${step}" in
      1)
        choice="$(prompt_menu "Platform" 0 \
          "VMware vSphere" "vsphere" \
          "VMware Desktop Hypervisors" "desktop")"
        if handle_menu_meta "${choice}"; then
          continue
        fi
        case "${choice}" in
          "${MENU_QUIT}")
            return 0
            ;;
          vsphere)
            PLATFORM="vsphere"
            ARCH="amd64"
            step=2
            ;;
          desktop)
            PLATFORM="desktop"
            ARCH="$(detect_host_arch)" || return 1
            step=2
            ;;
          *)
            PLATFORM="${choice}"
            step=2
            ;;
        esac
        ;;
      2)
        choice="$(prompt_menu "Example" 1 \
          "Basic Authentication" "basic" \
          "Basic Authentication with Properties" "basic-with-properties" \
          "API Key Authentication" "api-key" \
          "Access Token Authentication" "access-token")"
        if handle_menu_meta "${choice}"; then
          continue
        fi
        case "${choice}" in
          "${MENU_QUIT}")
            return 0
            ;;
          "${MENU_BACK}")
            step=1
            ;;
          *)
            EXAMPLE="${choice}"
            step=4
            ;;
        esac
        ;;
      4)
        choice="$(prompt_menu "Export Format" 1 \
          "OVA (Single-file Artifact)" "ova" \
          "OVF (Multi-file Artifact)" "ovf")"
        if handle_menu_meta "${choice}"; then
          continue
        fi
        case "${choice}" in
          "${MENU_QUIT}")
            return 0
            ;;
          "${MENU_BACK}")
            step=2
            ;;
          *)
            FORMAT="${choice}"
            step=5
            ;;
        esac
        ;;
      5)
        set +e
        ensure_artifactory_env
        setup_rc=$?
        set -e
        case "${setup_rc}" in
          0)
            step=6
            ;;
          1)
            step=4
            ;;
          2)
            return 0
            ;;
          *)
            return 1
            ;;
        esac
        ;;
      6)
        build_packer_args
        print_summary
        print_nav_hint 1
        while true; do
          printf 'Start Build? (y/n) ' >&2
          read_tty || true
          case "${REPLY:-}" in
            [yY])
              break 2
              ;;
            [nN] | [bB])
              step=4
              break
              ;;
            [qQ])
              return 0
              ;;
            [hH])
              show_interactive_help
              print_summary
              print_nav_hint 1
              ;;
            *)
              printf '\n' >&2
              printf '%bInvalid Selection:%b Enter %by%b or %bn%b.\n\n' \
                "${COLOR_YELLOW}" "${COLOR_RESET}" \
                "${COLOR_GREEN}" "${COLOR_RESET}" \
                "${COLOR_RED}" "${COLOR_RESET}" >&2
              ;;
          esac
        done
        ;;
    esac
  done

  clear_screen
  print_header

  # shellcheck disable=SC1090
  eval "$("${RESOLVE_ISO}" "${ARCH}")"

  prepare_plugins

  print_status "Building the machine image..."
  (
    cd "${EXAMPLE_DIR}"
    packer build "${PACKER_ARGS[@]}"
  )

  printf '\n%bDone.%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}"
}

parse_args "$@"
main
