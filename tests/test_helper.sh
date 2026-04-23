#!/usr/bin/env bash

# Shared helpers for Bats tests.

setup_test_env() {
  export PROJECT_ROOT
  PROJECT_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"

  export TEST_TMPDIR
  TEST_TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/bats-mytonprovider.XXXXXX")"

  export STUBS_DIR="${TEST_TMPDIR}/stubs"
  export STUB_LOG="${TEST_TMPDIR}/stubs.log"
  mkdir -p "$STUBS_DIR"
  : > "$STUB_LOG"

  export ORIGINAL_PATH="$PATH"
  export PATH="$STUBS_DIR:$PATH"
}

teardown_test_env() {
  export PATH="$ORIGINAL_PATH"
  rm -rf "$TEST_TMPDIR"
}

create_stub() {
  local cmd="$1"
  local body="${2:-exit 0}"

  {
    printf '%s\n' '#!/usr/bin/env bash'
    printf '%s\n' 'set -e'
    printf '%s\n' 'if [ -n "${STUB_LOG:-}" ]; then'
    printf '%s\n' "  printf '%s|%s\\n' '$cmd' \"\$*\" >> \"\$STUB_LOG\""
    printf '%s\n' 'fi'
    printf '%s\n' "$body"
  } > "$STUBS_DIR/$cmd"

  chmod +x "$STUBS_DIR/$cmd"
}

assert_output_contains() {
  local expected="$1"
  # shellcheck disable=SC2154
  [[ "$output" == *"$expected"* ]]
}

assert_log_contains() {
  local expected="$1"
  grep -F "$expected" "$STUB_LOG" >/dev/null
}

line_of_pattern() {
  local pattern="$1"
  local line
  line=$(grep -n "$pattern" "$STUB_LOG" | head -n 1 | cut -d: -f1)
  echo "${line:-0}"
}
