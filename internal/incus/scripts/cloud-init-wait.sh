#!/bin/sh
# Wait for cloud-init to complete, tailing its output log for progress.
# Only the log output is forwarded to the caller; the cloud-init status
# command itself runs silently.  tail is cleaned up on exit regardless of
# how cloud-init exits.
tail -f /var/log/cloud-init-output.log &
TAIL_PID=$!

cloud-init status --wait >/dev/null 2>&1
CI_RC=$?

kill $TAIL_PID 2>/dev/null
wait $TAIL_PID 2>/dev/null

exit $CI_RC
