#!/bin/bash

JOB_RESULT="$1"
START_TIME="$2"
TEST_TYPE="$3"
WORKSPACE="$4"

INPUT_FILE="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/raw_logs.log"
OUTPUT_PATH="$WORKSPACE/test/dashboard/logs/$TEST_TYPE"
JOB_OUTPUT_FILE="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/result.json"

END_TIME="`date "+%Y-%m-%dT%H:%M:%S.%NZ"`"
START_SECONDS=$(date -d "$START_TIME" +%s.%N)
END_SECONDS=$(date -d "$END_TIME" +%s.%N)
DURATION=$(echo "$END_SECONDS - $START_SECONDS" | bc)

MSG=""        # individual test msg
FAIL_MSG=""   # msg for entire job run
RESULT=""
HAS_FAILED=false
IS_RUNNING=false

load_job_status(){
    if [ "$JOB_RESULT" == "success" ]; then
        RESULT="pass"
    elif [ "$JOB_RESULT" == "failure" ]; then
        RESULT="fail"
    else
        RESULT="skip"
    fi
}

format_logs_to_json(){
    line="$1"
    json="{"
    
    while [[ "$line" =~ ([a-zA-Z0-9_]+)=((\"([^\"\\]|\\.)*\")|[^[:space:]]+) ]]; do
        key="${BASH_REMATCH[1]}"
        value="${BASH_REMATCH[2]}"
        line="${line#*"${key}=${value}"}"
        
        if [[ "$value" == \"*\" ]]; then
            value="${value:1:${#value}-2}"
            value="${value//\"/\\\"}"
        fi
        json+="\"$key\":\"$value\","
    done
    
    json="${json%,}}"    
    echo "$json"
}

format_results(){
    while IFS= read -r line; do
        
        if [[ "$line" =~ ^===\ RUN[[:space:]]+(.+) ]]; then
            TEST_NAME="${BASH_REMATCH[1]}"
            IS_RUNNING=true
            MSG=""
            TEST_START=""
            TEST_END=""
            mkdir -p "$OUTPUT_PATH/$TEST_NAME/"
            RESULT_FILE="$OUTPUT_PATH/$TEST_NAME/result.json"
            LOG_FILE="$OUTPUT_PATH/$TEST_NAME/test.log"
        elif [[ "$line" =~ ([0-9T:\.\-Z]+)[[:space:]]+testing ]]; then
            TEST_START="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ([0-9T:\.\-Z]+)[[:space:]]+finished[[:space]]testing ]]; then
            TEST_END="${BASH_REMATCH[1]}"
        elif [[ "$line" == "FAIL" ]]; then
            HAS_FAILED=false
            MSG="$MSG_STR"
            FAIL_MSG+="$MSG"
            HAS_FAILED=false
            echo "{\"start_at\": \"$START_TIME\", \"end_at\": \"$END_TIME\", \"duration_seconds\": \"$DURATION\", \"result\": \"$TEST_RES\", \"msg\": \"$MSG\"}" > $RESULT_FILE
        elif [[ "$line" == "--- PASS"* ]]; then
            TEST_RES="pass"
            IS_RUNNING=false
            echo "{\"start_at\": \"$START_TIME\", \"end_at\": \"$END_TIME\", \"duration_seconds\": \"$DURATION\", \"result\": \"$TEST_RES\", \"msg\": \"$MSG\"}" > $RESULT_FILE
        elif [[ "$line" == "--- FAIL"* ]]; then
            TEST_RES="fail"
            HAS_FAILED=true
            IS_RUNNING=false
        elif [[ "$line" == time=* && "$line" == *level=* ]]; then
            LOG_LINE=$(format_logs_to_json "$line")
            echo "$LOG_LINE" >> "$LOG_FILE"
        fi
        
        if [ $HAS_FAILED == true ]; then
            MSG_STR+="$line"
        fi

    done < "$INPUT_FILE"
            
    # Store the result of the whole job
    echo "{\"start_at\": \"$START_TIME\", \"end_at\": \"$END_TIME\", \"duration_seconds\": \"$DURATION\", \"result\": \"$RESULT\", \"msg\": \"$FAIL_MSG\"}" > $JOB_OUTPUT_FILE
}

# Main body of the script
{
    load_job_status
    format_results
}
