#!/usr/bin/env bats

source "$BATS_TEST_DIRNAME/test_helper.sh"

setup() {
  setup_test_env
  create_stub psql 'exit 0'
  create_stub go 'exit 0'
  create_stub systemctl 'exit 0'
}

teardown() {
  teardown_test_env
}

@test "init_db.sh exits with code 1 when required env vars are missing" {
  run env -i PATH="$PATH" bash "$PROJECT_ROOT/scripts/init_db.sh"

  [ "$status" -eq 1 ]
}

@test "init_db.sh prints missing variables error message" {
  run env -i PATH="$PATH" bash "$PROJECT_ROOT/scripts/init_db.sh"

  [ "$status" -eq 1 ]
  assert_output_contains "Missing required environment variables"
}

@test "init_db.sh passes SQL path resolved from script dirname" {
  local random_cwd="$TEST_TMPDIR/random-cwd"
  mkdir -p "$random_cwd"

  run bash -c '
    set -e
    cd "$1"
    PG_USER=user PG_PASSWORD=pass PG_DB=db PATH="$2" bash "$3"
  ' -- "$random_cwd" "$PATH" "$PROJECT_ROOT/scripts/init_db.sh"

  [ "$status" -eq 0 ]
  assert_log_contains "psql|-h 127.0.0.1 -p 5432 -U user -d db -f $PROJECT_ROOT/scripts/../db/init.sql"
}
