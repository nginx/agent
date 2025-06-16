LINT_FILE="lint-output.json"
START_TIME="$2"
END_TIME="`date "+%Y-%m-%dT%H:%M:%S.%NZ"`"
START_SECONDS=$(date -d "$START_TIME" +%s.%N)
END_SECONDS=$(date -d "$END_TIME" +%s.%N)
DURATION=$(echo "$END_SECONDS - $START_SECONDS" | bc)
MSG=""

if [ "$1" == "success" ]; then
  RESULT="pass"
elif [ "$1" == "failure" ]; then
  RESULT="fail"
else
  RESULT="skip"
fi

if [[ -f "$LINT_FILE" ]]; then  
  NUM_ISSUES=$(jq '.Issues | length' "$LINT_FILE")
  echo "$NUM_ISSUES"

  if [[ "$NUM_ISSUES" -ge 1 ]]; then
    MSG="`jq -r '.Issues[0].Text' "$LINT_FILE"`"
    echo
  fi
fi

echo "{\"start_at\": \"$START_TIME\", \"end_at\": \"$END_TIME\", \"duration_seconds\": \"$DURATION\", \"result\": \"$RESULT\", \"msg\": \"$MSG\"}" > lint-result.json
