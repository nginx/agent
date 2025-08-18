#!/bin/bash

set -euo pipefail

# parameters
JOB_RESULT="$1"
TEST_TYPE="$2"
WORKSPACE="$3"

# file paths
INPUT_FILE="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/raw_logs.log"
TEST_LOG="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/test.log"

# Validate input file exists
if [ ! -f "$INPUT_FILE" ]; then
  echo "Error: Input file $INPUT_FILE does not exist."
  exit 1
fi

load_job_status(){
    if [ "$RESULT" == "success" ]; then
        JOB_RESULT="pass"
    elif [ "$RESULT" == "failure" ]; then
        JOB_RESULT="fail"
    else
        JOB_RESULT="skip"
    fi
}

format_log() {
    local line="$1"
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

write_result() {
    local test_name="$1"
    local start_at="${start_time[$test]}"
    local end_at="${end_time[$test]}"
    local duration=0
    local result="$4"
    local msg="$5"
    [[ -n "$start_at" && -n "$end_at" ]] && duration_seconds=$(( $(date -d "$end_at" +%s) - $(date -d "$start_at" +%s) ))
    
    local start_iso=""
    local end_iso=""
    
    [[ -n "$start_at" ]] && start_iso=$(date -d "$start_at" +"%Y-%m-%dT%H:%M:%S.%NZ")
    [[ -n "$end_at" ]] && end_iso=$(date -d "$end_at" +"%Y-%m-%dT%H:%M:%S.%NZ")
    
    output_dir="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/$test/"
    mkdir -p "$output_dir"
    result_file="$output_dir/result.json"
    
    echo "{\"start_at\":\"$start_iso\", \"end_at\":\"$end_iso\", \"duration_seconds\":$duration_seconds, \"result\":\"${test_results[$test]}\", \"msg\":\"${test_msg[$test]}\"}" > "$result_file"
}

format_results() {
    test_stack=()
    current_test=""
    
    start_at=""
    end_at=""
    result=""
    msg=""
    
    while IFS= read -r line; do
        # Detect if the line is a test start
        if [[ "$line" =~ ^===\ RUN[[:space:]]+(.+) ]]; then
            if [[ -n "$current_test" ]]; then
                write_result "$current_test" "$start_at" "$end_at" "$result" "$msg"
                test_stack+=("$current_test|$start_at|$end_at|$result|$msg")
            fi
            
            current_test="${BASH_REMATCH[1]}"
            start_at=""
            end_at=""
            result="pass"
            msg=""
            continue
        fi
        
        # Get start time
        if [[ "$line" =~ ^([0-9]{4}/[0-9]{2}/[0-9]{2}[[:space:]][0-9]{2}:[0-9]{2}:[0-9]{2}).*INFO[[:space]]+starting.*tests ]]; then
            test_start="${BASH_REMATCH[1]}"
            continue
        fi
        
        # Get end time
        if [[ "$line" =~ ^([0-9]{4}/[0-9]{2}/[0-9]{2}[[:space:]][0-9]{2}:[0-9]{2}:[0-9]{2}).*INFO[[:space]]+finished.*tests ]]; then
            test_end="${BASH_REMATCH[1]}"
            continue
        fi
        
        # Detect result
        if [[ "$line" == "--- PASS"* || "$line" == "--- FAIL"* || "$line" == "FAIL" ]]; then
            [[ "$line" == "--- PASS"* ]] && result="pass"
            [[ "$line" == "--- FAIL"* || "$line" == "FAIL" ]] && result="fail"
            
            write_result "$current_test" "$start_at" "$end_at" "$result" "$msg" 
            continue
        fi
        
        if [[ "$line" == time=* && "$line" == *level=* ]]; then
            LOG_LINE=$(format_log "$line")
            echo "$LOG_LINE" >> "$LOG_FILE"
            continue
        fi
        if [[ "$result" == "fail" ]]; then
            msg+="$line"
        fi
        
    done < "$INPUT_FILE"
}

{
    load_job_status
    format_results
}
