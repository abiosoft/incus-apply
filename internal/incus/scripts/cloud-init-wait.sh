#!/bin/sh
# Wait for cloud-init to complete, tailing its output log for progress.
# Only the log output is forwarded to the caller; the cloud-init status
# command itself runs silently.  tail is cleaned up on exit regardless of
# how cloud-init exits.
#
# tail -f is best-effort for progress output: the log file may not exist yet
# on some images (e.g. Alpine), so failure to open it is intentionally ignored.
if [ -e /var/log/cloud-init-output.log ]; then
  tail -f /var/log/cloud-init-output.log &
  TAIL_PID=$!
else
  TAIL_PID=
fi

cloud-init status --wait >/dev/null 2>&1
CI_RC=$?

# Exit code 2 is not a failure, rather a soft success.
[ "$CI_RC" -eq 2 ] && CI_RC=0

[ -n "$TAIL_PID" ] && kill $TAIL_PID 2>/dev/null
[ -n "$TAIL_PID" ] && wait $TAIL_PID 2>/dev/null

exit $CI_RC
