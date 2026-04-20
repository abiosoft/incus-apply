#!/bin/sh
# Wait for cloud-init to complete, tailing its output log for progress.
# Only the log output is forwarded to the caller; the cloud-init status
# command itself runs silently.  tail is cleaned up on exit regardless of
# how cloud-init exits.
#
# The log file may not exist yet when the script starts (e.g. on slow or
# minimal images).  A background watcher polls for up to 30 s for the file
# to appear before starting tail, so output is captured even when the file
# is created after the script begins.

LOG=/var/log/cloud-init-output.log

(
  retries=30
  while [ "$retries" -gt 0 ] && [ ! -e "$LOG" ]; do
    sleep 1
    retries=$((retries - 1))
  done
  [ -e "$LOG" ] && exec tail -f "$LOG"
) &
TAIL_PID=$!

cloud-init status --wait >/dev/null 2>&1
CI_RC=$?

# Exit code 2 is not a failure, rather a soft success.
[ "$CI_RC" -eq 2 ] && CI_RC=0

kill $TAIL_PID 2>/dev/null
wait $TAIL_PID 2>/dev/null

exit $CI_RC
