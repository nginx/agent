#!/bin/bash
# Script to process test logs and generate formatted result files
# Usage: ./format_results.sh <job_result> <test_type> <workspace>

set -euo pipefail

# Check if required arguments are provided
if [ $# -lt 3 ]; then
    echo "Usage: $0 <job_result> <test_type> <workspace>"
    exit 1
fi

# Parameters
RESULT="$1"
TEST_TYPE="$2"
WORKSPACE="$3"

# File paths
INPUT_FILE="$WORKSPACE/test/dashboard/logs/$TEST_TYPE/raw_logs.log"
OUTPUT_DIR="$WORKSPACE/test/dashboard/logs/$TEST_TYPE"

# Validate input file exists
if [ ! -f "$INPUT_FILE" ]; then
  echo "Error: Input file $INPUT_FILE does not exist."
  exit 1
fi

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
    start_at="$1"
    end_at="$2"
    result="$3"
    msg="$4"
    output_dir="$5"
    duration_seconds=0
    
    if [[ $end_at == "end_at" ]]; then
      end_at=$(date +"%Y/%m/%d %H:%M:%S")
    fi
    
    # Format timestamps
    if [[ "$start_at" =~ ^[0-9]{4}/[0-9]{2}/[0-9]{2}\ [0-9]{2}:[0-9]{2}:[0-9]{2}$ && \
          "$end_at" =~ ^[0-9]{4}/[0-9]{2}/[0-9]{2}\ [0-9]{2}:[0-9]{2}:[0-9]{2}$ ]]; then
      duration_seconds=$(( $(date -d "$end_at" +%s) - $(date -d "$start_at" +%s) ))
      start_iso=""
      end_iso=""
      start_iso=$(date -d "$start_at" +"%Y-%m-%dT%H:%M:%S.%NZ")
      end_iso=$(date -d "$end_at" +"%Y-%m-%dT%H:%M:%S.%NZ")
    else
      duration_seconds=0
    fi
    
    if [[ ${msg} == "msg" ]]; then
        msg=""
    fi
    
    mkdir -p "$output_dir"
    result_file="$output_dir/result.json"

    echo "{\"start_at\":\"$start_iso\", \"end_at\":\"$end_iso\", \"duration_seconds\":$duration_seconds, \"result\":\"$result\", \"msg\":\"$msg\"}" > "$result_file"
}

format_results() {
  test_group=("name" "start_at" "end_at" "result" "msg")
  current_test=("name" "start_at" "end_at" "result" "msg")
  test_queue=()
  is_running=false
  has_failed=false
  error_trace=""
    
  while IFS= read -r line; do
    # Detect if the line is a test start
    if [[ "$line" =~ ^===\ RUN[[:space:]]+(.+) ]]; then
      test_name="${BASH_REMATCH[1]}"
      has_failed=false
  
      if [[ "${test_group[0]}" == "name" && "$is_running" == false ]]; then
        is_running=true
        test_group[0]="$test_name"
      elif [[ "${test_group[0]}" != "name" && "$is_running" == true ]]; then
        is_running=true
        if [[ "${current_test[0]}" != "${test_group[0]}" ]]; then
          test_queue+=("${current_test[@]}")
        fi
      fi
      
      current_test=("$test_name" "start_at" "end_at" "result" "msg")
      continue
    fi
    
    # Get start time
    if [[ "$line" =~ ^([0-9]{4}/[0-9]{2}/[0-9]{2}[[:space:]][0-9]{2}:[0-9]{2}:[0-9]{2}).*INFO[[:space:]]+starting.*test* ]]; then
        test_start="${BASH_REMATCH[1]}"
        current_test[1]="$test_start"
        if [[ "${current_test[0]}" == "${test_group[0]}" ]]; then
          test_group[1]="$test_start"
        fi
        continue
    fi
    
    # Get end time
    if [[ "$line" =~ ^([0-9]{4}/[0-9]{2}/[0-9]{2}[[:space:]][0-9]{2}:[0-9]{2}:[0-9]{2}).*INFO[[:space:]]+finished.*test* ]]; then
        test_end="${BASH_REMATCH[1]}"
        if [[ "${current_test[2]}" == "end_at" ]]; then
          current_test[2]="$test_end"
          if [[ "${current_test[0]}" == "${test_group[0]}" ]]; then
            test_group[2]="$test_end"
          fi
        elif [[ "${current_test[2]}" != "end_at" ]]; then
          test_group[2]="$test_end"
        fi
        continue
    fi
    
    # Capture error messages
    if [[ "$line" == *"Error Trace"* || "$line" == *"runtime error"* ]]; then
      has_failed=true
      error_trace+="${line}"$'\n'
      continue
    fi
    
    # Detect result
    if [[ "$line" == *"--- PASS"* || "$line" == *"--- FAIL"* ]]; then
      [[ "$line" == *"--- PASS"* ]] && result_val="pass"
      [[ "$line" == *"--- FAIL"* ]] && result_val="fail"
      
      has_failed=false
      
      # Clear current_test field
      if [[ "${current_test[0]}" != "name" ]]; then
        if [[ "${current_test[0]}" == "${test_group[0]}" ]]; then
          current_test=("name" "start_at" "end_at" "result" "msg")
        else
          test_queue+=("${current_test[@]}")
          current_test=("name" "start_at" "end_at" "result" "msg")
        fi
      fi
      
      # Write results for the test group
      if [[ "${test_group[0]}" != "name" && "${#test_queue[@]}" -eq 0 ]] || 
      [[ "${test_group[0]}" != "name" && "${#test_queue[@]}" -gt 0 ]]; then
        if [[ "$line" != *"${test_group[0]}"* ]]; then
          echo "Error: Test name did not match. Expected '${test_group[0]}', in line: '$line'."
          exit 1
        fi 
        test_group[3]="$result_val"
        if [[ "$result_val" == "fail" ]]; then
          if [[ ${test_group[4]} == "msg" ]]; then
            test_group[4]=""
          fi
          test_group[4]+="$error_trace"
        fi
        write_result "${test_group[1]}" "${test_group[2]}" "${test_group[3]}" "${test_group[4]}" "$OUTPUT_DIR/${test_group[0]}"
        test_group=("name" "start_at" "end_at" "result" "msg")
        is_running=false
        continue
      fi 
      
      # Write results for individual tests in the queue
      if [[ "${test_group[0]}" == "name" && "${#test_queue[@]}" -gt 0 ]]; then
        test_match=("${test_queue[0]}" "${test_queue[1]}" "${test_queue[2]}" "${test_queue[3]}" "${test_queue[4]}")
        test_match[3]="$result_val"
        if [[ "$result_val" == "fail" ]]; then
          if [[ ${test_match[4]} == "msg" ]]; then
            test_match[4]=""
          fi
          test_match[4]+="$error_trace"
        fi
        write_result "${test_match[1]}" "${test_match[2]}" "${test_match[3]}" "${test_match[4]}" "$OUTPUT_DIR/${test_match[0]}"
        
        for i in {0..4}; do
          unset 'test_queue[$i]'
        done
        test_queue=("${test_queue[@]:5}")
      fi
      
      # No tests to analyze
      if [[ "${test_group[0]}" == "name" && "${#test_queue[@]}" -eq 0 ]]; then
        error_trace=""
        continue
      fi
    fi
    
    # Capture error messages
    if [[ $has_failed == true ]]; then
      error_trace+="${line}"$'\n'
    fi
    
    # Capture logs
    if [[ "$line" =~ time=([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z)[[:space:]]+level= ]]; then
        LOG_LINE=$(format_log "$line")
        LOG_FILE_OUT_DIR="$OUTPUT_DIR/${test_group[0]}"
        LOG_FILE=${LOG_FILE_OUT_DIR}/test.log
        if [[ ! -d "$LOG_FILE_OUT_DIR" ]]; then
          mkdir -p "$LOG_FILE_OUT_DIR"
        fi
        echo "$LOG_LINE" >> "$LOG_FILE"
        continue
    fi
  done < "$INPUT_FILE"
}

{
    format_results
}
