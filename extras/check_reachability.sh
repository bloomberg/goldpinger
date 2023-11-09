#!/bin/bash

# This is a simple script, used in the Tilt GitHub Action
# to validate that all worker nodes can inter-communicate.

function print_help() {
  echo "Usage: $0 [EXPECTED_HOST_COUNT]"
  echo "Arguments:"
  echo "  EXPECTED_HOST_COUNT: The number of expected hosts in the Goldpinger output."
  echo "Examples:"
  echo "  $0 2"
}

if [ "$#" -ne 1 ]; then
  echo "Error: Invalid number of arguments."
  print_help
  exit 1
fi

if ! [[ $1 =~ ^[0-9]+$ ]]; then
  echo "Error: EXPECTED_HOST_COUNT must be a number."
  print_help
  exit 1
fi

expected_host_count=$1
goldpinger_output=""
retry_count=0
host_count=0

while :; do
  if [ "$retry_count" -ge 20 ]; then
    echo "Error: Failed to fetch Goldpinger output after 10 attempts."
    exit 1
  fi
  
  echo "Sleeping for 8s..."
  let retry_count++
  sleep 8

  echo "Attempt $((retry_count)) to fetch Goldpinger output."
  goldpinger_output=$(curl -s http://localhost:8080/check_all)
  echo "Goldpinger output: $goldpinger_output"

  if [ "$goldpinger_output" == "null" ] || [ -z "$goldpinger_output" ]; then
    echo "Goldpinger output is null or empty, retrying..."
    continue
  fi

  host_count=$(echo "$goldpinger_output" | jq '.hosts | length')
  if [ "$host_count" -ne "$expected_host_count" ]; then
    echo "Goldpinger has not identified all hosts, retrying..."
    continue
  fi

  for host in $(echo $goldpinger_output | jq -r '.responses | keys[]'); do
    checksForPod=$(echo "$goldpinger_output" | jq -r --arg host "$host" '.responses[$host].response.podResults | length')
    if [ "$checksForPod" -ne "$expected_host_count" ]; then
      echo "Check for $host is not OK, retrying..."
      continue 2
    fi
  done

  break
done

all_hosts_can_talk=true
for host in $(echo $goldpinger_output | jq -r '.responses | keys[]'); do
  for target in $(echo $goldpinger_output | jq -r --arg host $host '.responses[$host].response.podResults | keys[]'); do
    ok=$(echo $goldpinger_output | jq -r --arg host $host --arg target $target '.responses[$host].response.podResults[$target].OK')
    if [ "$ok" != "true" ]; then
      all_hosts_can_talk=false
      break 2
    fi
  done
done

if [[ $host_count -eq $expected_host_count ]] && [[ $all_hosts_can_talk == "true" ]]; then
  echo "Validation successful. There are $expected_host_count hosts and they can talk to each other."
else
  echo "Validation failed. Expected $expected_host_count hosts but found $host_count, or not all hosts can talk to each other."
  echo "Goldpinger Output: $goldpinger_output"
  exit 1
fi
