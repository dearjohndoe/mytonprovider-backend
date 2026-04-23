#!/usr/bin/env bats

source "$BATS_TEST_DIRNAME/test_helper.sh"

setup() {
  setup_test_env

  export PROVIDER_DIR="$TEST_TMPDIR/provider"
  mkdir -p "$PROVIDER_DIR"
  : > "$PROVIDER_DIR/mtpo-backend"
  chmod +x "$PROVIDER_DIR/mtpo-backend"

  # Empty value should not break env parsing.
  cat > "$PROVIDER_DIR/config.env" <<'CFG'
SYSTEM_PORT=9090
SYSTEM_ACCESS_TOKENS=
DB_HOST=127.0.0.1
CFG

  create_stub mkdir 'exit 0'
  create_stub pgrep 'exit 0'
  create_stub sleep 'exit 0'
  create_stub env 'exit 0'
  create_stub go 'exit 0'
  create_stub psql 'exit 0'
  create_stub systemctl 'exit 0'
}

teardown() {
  teardown_test_env
}

run_script_with_overridden_cd() {
  run bash -c '
    cd() { builtin cd "$PROVIDER_DIR"; }
    export -f cd
    source "$SCRIPT"
  '
}

@test "run.sh creates /var/log/mytonprovider.app with mkdir -p" {
  export SCRIPT="$PROJECT_ROOT/scripts/run.sh"

  run_script_with_overridden_cd

  [ "$status" -eq 0 ]
  assert_log_contains "mkdir|-p /var/log/mytonprovider.app"
}

@test "run.sh does not fail with empty values in config.env" {
  export SCRIPT="$PROJECT_ROOT/scripts/run.sh"

  run_script_with_overridden_cd

  [ "$status" -eq 0 ]
}

@test "run.sh exits with code 1 when backend process is not found after start" {
  create_stub pgrep 'exit 1'
  export SCRIPT="$PROJECT_ROOT/scripts/run.sh"

  run_script_with_overridden_cd

  [ "$status" -eq 1 ]
}

