#!/usr/bin/env bats

source "$BATS_TEST_DIRNAME/test_helper.sh"

setup() {
  setup_test_env

  export WORK_DIR="$TEST_TMPDIR/work"
  export HOST="127.0.0.1"
  export PG_USER="pguser"
  export PG_PASSWORD="pgpass"
  export PG_DB="providerdb"

  mkdir -p "$WORK_DIR/mytonprovider-backend/cmd"

  create_stub go '
output=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then
    output="$arg"
    break
  fi
  prev="$arg"
done
[ -n "$output" ] && : > "$output"
exit 0
'
  create_stub mkdir 'exit 0'
  create_stub mv 'exit 0'
  create_stub psql 'exit 0'
  create_stub systemctl 'exit 0'
}

teardown() {
  teardown_test_env
}

@test "build_backend.sh creates /opt/provider before mv" {
  run bash "$PROJECT_ROOT/scripts/build_backend.sh"

  [ "$status" -eq 0 ]

  local mkdir_line
  local mv_line
  mkdir_line=$(line_of_pattern 'mkdir|-p /opt/provider')
  mv_line=$(line_of_pattern 'mv|mtpo-backend /opt/provider/')

  [ "$mkdir_line" -gt 0 ]
  [ "$mv_line" -gt 0 ]
  [ "$mkdir_line" -lt "$mv_line" ]
}
