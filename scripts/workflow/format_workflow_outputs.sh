#!/bin/sh

START_TIME="$2"
IN_FILE="$3"
OUT_FILE="$4"
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

echo "CATTING"
cat $IN_FILE
echo "CATTED"

if [[ -f "$IN_FILE" ]]; then  
  NUM_ISSUES=$(jq '.Issues | length' "$IN_FILE")
  echo "$NUM_ISSUES"

  if [[ "$NUM_ISSUES" -ge 1 ]]; then
    MSG="`jq -r '.Issues[0].Text' "$IN_FILE"`"
  else
    MSG="`grep -E 'Error:|Error Trace:' $IN_FILE | sed -e 's/^\s*//' -e '/^$/d'`"
  fi
fi

echo "{\"start_at\": \"$START_TIME\", \"end_at\": \"$END_TIME\", \"duration_seconds\": \"$DURATION\", \"result\": \"$RESULT\", \"msg\": \"$MSG\"}" > $OUT_FILE
