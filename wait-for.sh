#!/bin/sh

# =============================================================================
# FILE: wait-for.sh
# ROLE: "Wait until another service is reachable, THEN run a command."
#
# This is NOT Go and NOT part of the app's architecture. It is a small, well
# known open-source helper script (by Eficode, MIT licence - see below) that
# was copied into the project. You do not need to write it yourself: to
# replicate the project, copy this file, or get the original from
#     https://github.com/eficode/wait-for
#
# WHY THE PROJECT NEEDS IT
#   docker-compose starts the database container and the app container almost
#   at the same moment, but Postgres needs a few seconds before it accepts
#   connections. If the app started immediately, its migration step would fail
#   and the app would crash. So docker-compose.yml starts the app like this:
#
#       /app/wait-for.sh postgres:5432 -- /app/main
#
#   meaning: keep testing whether host "postgres" answers on port 5432 (once
#   per second, 15 seconds at most); as soon as it does, run /app/main.
#
# HOW TO READ SHELL SCRIPT (the minimum)
#   - The first line "#!/bin/sh" says which program runs this file: sh.
#   - "#" starts a comment.
#   - $1, $2... are the arguments given to the script; "$@" is all of them;
#     $# is how many there are; `shift` throws away $1 and renumbers the rest.
#   - NAME=value sets a variable; "$NAME" reads it.
#   - [ ... ] is a test; `if [ ... ]; then ... fi` is an if block.
#   - `case X in pattern) ... ;; esac` is a switch.
#   - $? is the exit code of the last command: 0 = success, anything else =
#     failure.
#
# The comments explaining the code were added for learning; the code itself is
# unchanged.
# =============================================================================

# The MIT License (MIT)
#
# Copyright (c) 2017 Eficode Oy
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# A TRICK TO PROTECT THE CALLER'S VARIABLES.
# `set -- a b c` REPLACES the script's argument list with a, b, c.
# This line keeps the original arguments ("$@") and APPENDS a "--" marker
# followed by the current values of six variables. Why: plain sh has no local
# variables, so this script is about to overwrite TIMEOUT, QUIET, etc. By
# parking their old values at the end of the argument list, it can put them
# back just before running the final command (see inside wait_for).
set -- "$@" -- "$TIMEOUT" "$QUIET" "$PROTOCOL" "$HOST" "$PORT" "$result"
# Default: give up after 15 seconds.
TIMEOUT=15
# Default: print status messages (1 would mean "stay quiet").
QUIET=0
# The protocol to make the request with, either "tcp" or "http"
PROTOCOL="tcp"

# A FUNCTION named echoerr: print a message to the error output.
echoerr() {
  # Only when not in quiet mode. "$*" is all the function's arguments as one
  # string; "1>&2" sends the output to stderr (the error stream) instead of
  # stdout.
  if [ "$QUIET" -ne 1 ]; then printf "%s\n" "$*" 1>&2; fi
}

# A function that prints the help text and exits.
usage() {
  # $1 here is the function's own first argument: the exit code to use.
  exitcode="$1"
  # "cat << USAGE" is a HERE-DOCUMENT: everything up to the line that contains
  # only USAGE is printed as-is. $0 is the name of this script.
  cat << USAGE >&2
Usage:
  $0 host:port|url [-t timeout] [-- command args]
  -q | --quiet                        Do not output any status messages
  -t TIMEOUT | --timeout=timeout      Timeout in seconds, zero for no timeout
  -- COMMAND ARGS                     Execute command with args after the test finishes
USAGE
  # Stop the script with that exit code.
  exit "$exitcode"
}

# THE MAIN FUNCTION: test the connection in a loop until it works or time
# runs out.
wait_for() {
  # First make sure the tool needed for the chosen protocol is installed.
  case "$PROTOCOL" in
    tcp)
      # `command -v nc` checks whether the program `nc` (netcat) exists.
      # ">/dev/null" throws its output away; "!" negates the result.
      if ! command -v nc >/dev/null; then
        echoerr 'nc command is missing!'
        exit 1
      fi
      ;;
    wget)
      # (This branch checks for wget. PROTOCOL is only ever "tcp" or "http",
      # so it is never reached - a leftover in the original script.)
      if ! command -v wget >/dev/null; then
        echoerr 'nc command is missing!'
        exit 1
      fi
      ;;
  esac

  # `while :; do` is an endless loop (":" is a command that always succeeds).
  while :; do
    case "$PROTOCOL" in
      tcp)
        # THE ACTUAL TEST. `nc -z host port` tries to open a TCP connection
        # and closes it immediately without sending data. It succeeds (exit
        # code 0) when something is listening there. "> /dev/null 2>&1" hides
        # both normal and error output.
        nc -z "$HOST" "$PORT" > /dev/null 2>&1
        ;;
      http)
        # For URLs: try to download the page, waiting 1 second at most.
        wget --timeout=1 -q "$HOST" -O /dev/null > /dev/null 2>&1
        ;;
      *)
        # "*" matches anything else.
        echoerr "Unknown protocol '$PROTOCOL'"
        exit 1
        ;;
    esac

    # Remember the exit code of the test above (0 = the service answered).
    result=$?

    # SUCCESS: the service is reachable.
    if [ $result -eq 0 ] ; then
      # More than 7 arguments left means a command was given after "--".
      # (The 7 are the "--" marker plus the six parked values from the top.)
      if [ $# -gt 7 ] ; then
        # ROTATE the argument list: move the command's words, one at a time,
        # from the front to the back. Afterwards the 7 parked items are at
        # the front and the command is at the end.
        for result in $(seq $(($# - 7))); do
          result=$1
          shift
          set -- "$@" "$result"
        done

        # Restore the caller's original variable values ($1 is the "--").
        TIMEOUT=$2 QUIET=$3 PROTOCOL=$4 HOST=$5 PORT=$6 result=$7
        # Drop those 7 items; only the command remains.
        shift 7
        # `exec` REPLACES this script with the command. For this project that
        # is /app/main: the Go app starts and becomes the container's main
        # process.
        exec "$@"
      fi
      # No command was given: just report success.
      exit 0
    fi

    # NOT READY YET. If the time budget is used up, leave the loop...
    if [ "$TIMEOUT" -le 0 ]; then
      break
    fi
    # ...otherwise count down by one...
    TIMEOUT=$((TIMEOUT - 1))

    # ...wait one second, and try again.
    sleep 1
  done
  # The loop ended without success. Exit code 1 = failure. In docker-compose
  # the container then stops, and "restart: on-failure" starts it again.
  echo "Operation timed out" >&2
  exit 1
}

# ---------------------------------------------------------------------------
# THE SCRIPT STARTS RUNNING HERE: read the arguments one by one.
# ---------------------------------------------------------------------------
while :; do
  # Look at the first remaining argument and compare it with each pattern.
  case "$1" in
    # A URL: test with HTTP instead of TCP.
    http://*|https://*)
    HOST="$1"
    PROTOCOL="http"
    shift 1
    ;;
    # Something containing ":" - this is the "postgres:5432" case.
    *:* )
    # `cut -d : -f 1` splits the text at ":" and keeps part 1 -> "postgres".
    HOST=$(printf "%s\n" "$1"| cut -d : -f 1)
    # Part 2 -> "5432".
    PORT=$(printf "%s\n" "$1"| cut -d : -f 2)
    shift 1
    ;;
    # -q or --quiet: do not print status messages.
    -q | --quiet)
    QUIET=1
    shift 1
    ;;
    # "-q-something" is not valid.
    -q-*)
    QUIET=0
    echoerr "Unknown option: $1"
    usage 1
    ;;
    # -q glued to other short options (for example -qt5): turn quiet on and
    # put the remaining letters back as a new option to be read next.
    -q*)
    QUIET=1
    result=$1
    shift 1
    set -- -"${result#-q}" "$@"
    ;;
    # -t 30 or --timeout 30: the value is the NEXT argument, so drop two.
    -t | --timeout)
    TIMEOUT="$2"
    shift 2
    ;;
    # -t30 : "${1#-t}" removes the leading "-t", leaving "30".
    -t*)
    TIMEOUT="${1#-t}"
    shift 1
    ;;
    # --timeout=30 : "${1#*=}" removes everything up to "=", leaving "30".
    --timeout=*)
    TIMEOUT="${1#*=}"
    shift 1
    ;;
    # "--" ends the options. Whatever follows is the command to run later.
    # Drop the "--" and leave the argument-reading loop.
    --)
    shift
    break
    ;;
    --help)
    usage 0
    ;;
    # Any other option starting with "-" is unknown.
    -*)
    QUIET=0
    echoerr "Unknown option: $1"
    usage 1
    ;;
    # Anything else is unknown too.
    *)
    QUIET=0
    echoerr "Unknown argument: $1"
    usage 1
    ;;
  esac
done

# Check that TIMEOUT is a number that is 0 or more. "2>/dev/null" hides the
# error message the test itself prints when TIMEOUT is not a number.
if ! [ "$TIMEOUT" -ge 0 ] 2>/dev/null; then
  echoerr "Error: invalid timeout '$TIMEOUT'"
  usage 3
fi

# Check that we know what to test.
case "$PROTOCOL" in
  tcp)
    # TCP needs both a host and a port. "||" means "or".
    if [ "$HOST" = "" ] || [ "$PORT" = "" ]; then
      echoerr "Error: you need to provide a host and port to test."
      usage 2
    fi
  ;;
  http)
    # HTTP needs only the URL.
    if [ "$HOST" = "" ]; then
      echoerr "Error: you need to provide a host to test."
      usage 2
    fi
  ;;
esac

# Everything is ready: start waiting. "$@" passes along what is left of the
# argument list: the command to run, then the "--" marker and the six parked
# values.
wait_for "$@"
