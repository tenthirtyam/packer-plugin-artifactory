#!/usr/bin/env bash

# Copyright (c) Ryan Johnson
# SPDX-License-Identifier: MPL-2.0

# Setup the Artifactory environment variables used by the integration test
# environment.
#
# Usage:
#   source scripts/setup-packer-env.sh
#   make test-integration-environment

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

is_sourced() {
  if [ -n "${ZSH_EVAL_CONTEXT:-}" ]; then
    case "${ZSH_EVAL_CONTEXT}" in
      *:file:* | *file*) return 0 ;;
    esac
    return 1
  fi

  if [ -n "${BASH_VERSION:-}" ]; then
    # shellcheck disable=SC2295
    [[ "${BASH_SOURCE[0]}" != "${0}" ]]
    return
  fi

  return 1
}

_SETUP_ROOT_DIR="$(cd "$(dirname "$(script_path)")/.." && pwd)"

# shellcheck source=scripts/artifactory-docker.sh
source "${_SETUP_ROOT_DIR}/scripts/artifactory-docker.sh"

COLOR_GREEN=$'\033[0;32m'
COLOR_YELLOW=$'\033[1;33m'
COLOR_BLUE=$'\033[0;34m'
COLOR_RED=$'\033[0;31m'
COLOR_DIM=$'\033[2m'
COLOR_RESET=$'\033[0m'

clear_screen() {
  if [ -c /dev/tty ]; then
    clear >/dev/tty 2>/dev/null || printf '\033[H\033[2J' >/dev/tty
  elif [ -t 1 ]; then
    clear 2>/dev/null || printf '\033[H\033[2J'
  fi
}

print_setup_header() {
  local line_width=72
  local title="Packer Plugin for Artifactory"
  local subtitle="Artifactory Setup"
  local pad_title pad_subtitle

  pad_title=$(( (line_width - ${#title}) / 2 ))
  pad_subtitle=$(( (line_width - ${#subtitle}) / 2 ))

  printf '\n'
  printf "${COLOR_BLUE}%*s%s${COLOR_RESET}\n" "${pad_title}" '' "${title}"
  printf "${COLOR_GREEN}%*s%s${COLOR_RESET}\n" "${pad_subtitle}" '' "${subtitle}"
  printf '\n'
}

prompt_read() {
  local __prompt="$1"
  local __var_name="$2"
  local __sensitive="${3:-false}"
  local __value=""

  printf '%b' "${__prompt}"
  if [ "${__sensitive}" = "true" ]; then
    read -r -s __value
    printf '\n'
  else
    read -r __value
  fi

  printf -v "${__var_name}" '%s' "${__value}"
}

read_with_default() {
  local prompt="$1"
  local default="$2"
  local var_name="$3"
  local sensitive="${4:-false}"
  local input_value=""
  local input_prompt=""

  if [ -n "${default}" ]; then
    input_prompt="${COLOR_YELLOW}${prompt}${COLOR_RESET} (${default}): "
  else
    input_prompt="${COLOR_YELLOW}${prompt}${COLOR_RESET}: "
  fi

  prompt_read "${input_prompt}" input_value "${sensitive}"

  if [ -z "${input_value}" ] && [ -n "${default}" ]; then
    input_value="${default}"
  fi

  printf -v "${var_name}" '%s' "${input_value}"
}

prompt_with_default() {
  local prompt="$1"
  local default="$2"
  local var_name="$3"
  local sensitive="$4"
  local resolved_value=""

  read_with_default "${prompt}" "${default}" resolved_value "${sensitive}"
  export "${var_name}"="${resolved_value}"
}

prompt_required() {
  local prompt="$1"
  local var_name="$2"
  local sensitive="${3:-false}"
  local resolved_value=""

  while true; do
    read_with_default "${prompt}" "" resolved_value "${sensitive}"
    if [ -n "${resolved_value}" ]; then
      export "${var_name}"="${resolved_value}"
      return 0
    fi
    printf '%bRequired.%b\n' "${COLOR_RED}" "${COLOR_RESET}"
  done
}

prompt_connection_credentials() {
  local default_user="${ARTIFACTORY_USERNAME:-${DEFAULT_ARTIFACTORY_USERNAME}}"
  local default_pass="${ARTIFACTORY_PASSWORD:-${DEFAULT_ARTIFACTORY_PASSWORD}}"

  printf '\n'
  read_with_default \
    "Username" \
    "${default_user}" \
    "setup_username" \
    "false"
  printf '\n'
  read_with_default \
    "Password" \
    "${default_pass}" \
    "setup_password" \
    "true"
}

reject_local_access_token() {
  if [ "${ARTIFACTORY_URL}" = "${DEFAULT_ARTIFACTORY_URL}" ]; then
    printf '\n'
    printf '%bAccess tokens are not supported on local Artifactory OSS.%b\n' \
      "${COLOR_YELLOW}" "${COLOR_RESET}" >&2
    printf 'Choose Basic Authentication or API Key Authentication.\n'
    return 1
  fi
  return 0
}

configure_api_key_auth() {
  local api_key=""

  prompt_connection_credentials

  if ! verify_artifactory_credentials \
    "${ARTIFACTORY_URL}" "${setup_username}" "${setup_password}"; then
    return 1
  fi

  if api_key="$(fetch_artifactory_api_key \
    "${ARTIFACTORY_URL}" "${setup_username}" "${setup_password}")"; then
    export ARTIFACTORY_API_KEY="${api_key}"
  else
    printf '\n'
    prompt_required "API Key" "ARTIFACTORY_API_KEY" "true"
  fi

  unset setup_username setup_password
  unset ARTIFACTORY_TOKEN ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD
}

configure_access_token_auth() {
  local access_token=""

  if ! reject_local_access_token; then
    return 1
  fi

  prompt_connection_credentials

  if ! verify_artifactory_credentials \
    "${ARTIFACTORY_URL}" "${setup_username}" "${setup_password}"; then
    unset setup_username setup_password
    return 1
  fi

  if is_artifactory_oss \
    "${ARTIFACTORY_URL}" "${setup_username}" "${setup_password}"; then
    printf '\n'
    printf '%bAccess tokens are not supported on Artifactory OSS.%b\n' \
      "${COLOR_YELLOW}" "${COLOR_RESET}" >&2
    printf 'Choose Basic Authentication or API Key Authentication.\n'
    unset setup_username setup_password
    return 1
  fi

  if access_token="$(fetch_artifactory_access_token \
    "${ARTIFACTORY_URL}" "${setup_username}" "${setup_password}")"; then
    export ARTIFACTORY_TOKEN="${access_token}"
  else
    printf '\n'
    prompt_required "Access Token" "ARTIFACTORY_TOKEN" "true"
  fi

  unset setup_username setup_password
  unset ARTIFACTORY_API_KEY ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD
}

configure_username_password_auth() {
  printf '\n'
  prompt_with_default \
    "Username" \
    "${ARTIFACTORY_USERNAME:-${DEFAULT_ARTIFACTORY_USERNAME}}" \
    "ARTIFACTORY_USERNAME" \
    "false"
  printf '\n'
  prompt_with_default \
    "Password" \
    "${ARTIFACTORY_PASSWORD:-${DEFAULT_ARTIFACTORY_PASSWORD}}" \
    "ARTIFACTORY_PASSWORD" \
    "true"
  unset ARTIFACTORY_API_KEY ARTIFACTORY_TOKEN
}

# Prefer this-session choice, then the example the runner selected, then env credentials.
# Order matches examples/run-example.sh (without "with Properties"):
#   1 Basic Authentication, 2 API Key Authentication, 3 Access Token Authentication
detect_previous_auth_method() {
  if [ -n "${AUTH_METHOD:-}" ]; then
    case "${AUTH_METHOD}" in
      1|2|3) printf '%s\n' "${AUTH_METHOD}"; return 0 ;;
    esac
  fi

  case "${EXAMPLE:-}" in
    basic|basic-with-properties)
      printf '%s\n' "1"
      return 0
      ;;
    api-key)
      printf '%s\n' "2"
      return 0
      ;;
    access-token)
      printf '%s\n' "3"
      return 0
      ;;
  esac

  if [ -n "${ARTIFACTORY_USERNAME:-}" ] || [ -n "${ARTIFACTORY_PASSWORD:-}" ]; then
    printf '%s\n' "1"
    return 0
  fi
  if [ -n "${ARTIFACTORY_API_KEY:-}" ]; then
    printf '%s\n' "2"
    return 0
  fi
  if [ -n "${ARTIFACTORY_TOKEN:-}" ]; then
    printf '%s\n' "3"
    return 0
  fi

  printf '%s\n' ""
}

# Map the runner's selected example to an auth method, if any.
auth_method_from_example() {
  case "${EXAMPLE:-}" in
    basic|basic-with-properties) printf '%s\n' "1" ;;
    api-key) printf '%s\n' "2" ;;
    access-token) printf '%s\n' "3" ;;
    *) printf '%s\n' "" ;;
  esac
}

select_auth_method() {
  local auth_method=""
  local previous=""

  previous="$(detect_previous_auth_method)"

  while true; do
    clear_screen
    print_setup_header

    if [ "${previous}" = "1" ]; then
      printf '1: Basic Authentication  %b(previous)%b\n' "${COLOR_DIM}" "${COLOR_RESET}"
    else
      printf '1: Basic Authentication\n'
    fi
    if [ "${previous}" = "2" ]; then
      printf '2: API Key Authentication  %b(previous)%b\n' "${COLOR_DIM}" "${COLOR_RESET}"
    else
      printf '2: API Key Authentication\n'
    fi
    if [ "${previous}" = "3" ]; then
      printf '3: Access Token Authentication  %b(previous)%b\n' "${COLOR_DIM}" "${COLOR_RESET}"
    else
      printf '3: Access Token Authentication\n'
    fi
    printf '\n'

    if [ -n "${ARTIFACTORY_URL:-}" ] && [ "${ARTIFACTORY_URL}" != "${DEFAULT_ARTIFACTORY_URL}" ]; then
      printf 'Existing URL: %s\n\n' "${ARTIFACTORY_URL}"
    fi

    printf '(%bb%b) Back   (%bq%b) Quit\n' \
      "${COLOR_GREEN}" "${COLOR_RESET}" \
      "${COLOR_RED}" "${COLOR_RESET}"
    printf '\n'
    if [ -n "${previous}" ]; then
      printf 'Select Authentication Method [%s]: ' "${previous}"
    else
      printf 'Select Authentication Method: '
    fi
    # shellcheck disable=SC2162
    read -r auth_method || true

    if [ -z "${auth_method}" ] && [ -n "${previous}" ]; then
      auth_method="${previous}"
    fi

    case "${auth_method}" in
      b|B)
        return 1
        ;;
      q|Q)
        return 2
        ;;
      1|2|3)
        AUTH_METHOD="${auth_method}"
        return 0
        ;;
      *)
        printf '\n'
        printf '%bInvalid Selection:%b Enter a number between 1 and 3.\n' \
          "${COLOR_YELLOW}" "${COLOR_RESET}"
        printf '\n'
        printf 'Press %bEnter%b to continue.\n' "${COLOR_GREEN}" "${COLOR_RESET}"
        # shellcheck disable=SC2162
        read -r _ || true
        ;;
    esac
  done
}

configure_auth() {
  clear_screen
  print_setup_header

  case "${AUTH_METHOD}" in
    1) printf '%bBasic Authentication%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}" ;;
    2) printf '%bAPI Key Authentication%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}" ;;
    3) printf '%bAccess Token Authentication%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}" ;;
  esac

  if [ -z "${ARTIFACTORY_URL:-}" ] || [ "${ARTIFACTORY_URL}" = "${DEFAULT_ARTIFACTORY_URL}" ]; then
    printf 'Accept all defaults to use the local Docker defaults.\n\n'
  fi

  prompt_with_default \
    "URL" \
    "${ARTIFACTORY_URL:-${DEFAULT_ARTIFACTORY_URL}}" \
    "ARTIFACTORY_URL" \
    "false"

  case "${AUTH_METHOD}" in
    1) configure_username_password_auth ;;
    2) configure_api_key_auth ;;
    3) configure_access_token_auth ;;
  esac
}

configure_repository() {
  printf '\n'
  prompt_with_default \
    "Repository" \
    "${ARTIFACTORY_REPOSITORY:-${DEFAULT_ARTIFACTORY_REPOSITORY}}" \
    "ARTIFACTORY_REPOSITORY" \
    "false"
}

print_setup_summary() {
  clear_screen
  print_setup_header

  printf '%bReady.%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}"
  printf 'ARTIFACTORY_URL=%s\n' "${ARTIFACTORY_URL}"

  if [ -n "${ARTIFACTORY_API_KEY:-}" ]; then
    printf 'ARTIFACTORY_API_KEY=[set]\n'
  elif [ -n "${ARTIFACTORY_TOKEN:-}" ]; then
    printf 'ARTIFACTORY_TOKEN=[set]\n'
  else
    printf 'ARTIFACTORY_USERNAME=%s\n' "${ARTIFACTORY_USERNAME}"
    printf 'ARTIFACTORY_PASSWORD=[set]\n'
  fi

  printf 'ARTIFACTORY_REPOSITORY=%s\n' "${ARTIFACTORY_REPOSITORY}"
  printf '\n'
  printf 'Press %bEnter%b to continue.\n' "${COLOR_GREEN}" "${COLOR_RESET}"
  # shellcheck disable=SC2162
  read -r _ || true
}

# Returns: 0 configured, 1 back, 2 quit.
setup_packer_env() {
  local rc=0
  local from_example=""

  # Runner already chose the example — skip the auth method menu.
  from_example="$(auth_method_from_example)"
  if [ -n "${from_example}" ]; then
    AUTH_METHOD="${from_example}"
  else
    select_auth_method
    rc=$?
    if [ "${rc}" -ne 0 ]; then
      return "${rc}"
    fi
  fi

  # From the example runner, API key / token examples only need that secret.
  # URL and repository already have Packer HCL defaults.
  case "${EXAMPLE:-}" in
    api-key|access-token)
      clear_screen
      print_setup_header
      case "${AUTH_METHOD}" in
        2)
          printf '%bAPI Key Authentication%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}"
          prompt_required "API Key" "ARTIFACTORY_API_KEY" "true"
          unset ARTIFACTORY_TOKEN ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD
          ;;
        3)
          printf '%bAccess Token Authentication%b\n\n' "${COLOR_GREEN}" "${COLOR_RESET}"
          prompt_required "Acess Token" "ARTIFACTORY_TOKEN" "true"
          unset ARTIFACTORY_API_KEY ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD
          ;;
      esac
      return 0
      ;;
  esac

  while true; do
    if configure_auth; then
      break
    fi

    printf '\n'
    printf 'Press %bEnter%b to choose another authentication method.\n' \
      "${COLOR_GREEN}" "${COLOR_RESET}"
    # shellcheck disable=SC2162
    read -r _ || true

    select_auth_method
    rc=$?
    if [ "${rc}" -ne 0 ]; then
      return "${rc}"
    fi
  done

  configure_repository

  # Standalone setup shows a summary; the example runner continues to Start Build.
  if [ -z "${EXAMPLE:-}" ]; then
    print_setup_summary
  fi
  return 0
}

if ! is_sourced; then
  echo "error: source scripts/setup-packer-env.sh" >&2
  exit 1
fi

setup_packer_env
