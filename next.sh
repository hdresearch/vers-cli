#!/bin/bash
set -euo pipefail

# --- Self-sending mechanism ---
if [[ "${1:-}" == "--self-send" ]]; then
  SELF=$(cat "$0")
  OUTFILE="${2:-next.sh}"
  echo "Sending self to Claude..." >&2
  curl -s https://api.anthropic.com/v1/messages \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "content-type: application/json" \
    -H "anthropic-version: 2023-06-01" \
    -d "$(jq -n --arg s "$SELF" '{
      model: "claude-opus-4-6",
      max_tokens: 8192,
      messages: [{role: "user", content: $s}]
    }')" | jq -r '.content[0].text' > "$OUTFILE"
  echo "Wrote to $OUTFILE — review before running." >&2
  exit 0
fi

# --- Dependency checks ---
for cmd in curl jq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Error: $cmd is required" >&2; exit 1; }
done
[[ -n "${ANTHROPIC_API_KEY:-}" ]] || { echo "Error: ANTHROPIC_API_KEY is required" >&2; exit 1; }

MODEL="claude-opus-4-6"
API_URL="https://api.anthropic.com/v1/messages"
TOOLS='[{"name":"shell","description":"Execute a bash command and return its output.","input_schema":{"type":"object","properties":{"command":{"type":"string","description":"The bash command to execute"}},"required":["command"]}}]'

# Conversation history as a jq-compatible array
MESSAGES="[]"

call_api() {
  local payload
  payload=$(jq -n \
    --argjson msgs "$MESSAGES" \
    --argjson tools "$TOOLS" \
    '{
      model: "'"$MODEL"'",
      max_tokens: 4096,
      tools: $tools,
      messages: $msgs
    }')
  curl -s "$API_URL" \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "content-type: application/json" \
    -H "anthropic-version: 2023-06-01" \
    -d "$payload"
}

process_response() {
  local response="$1"
  local stop_reason content_blocks

  stop_reason=$(echo "$response" | jq -r '.stop_reason')
  content_blocks=$(echo "$response" | jq -c '.content')

  # Append assistant message to conversation
  MESSAGES=$(echo "$MESSAGES" | jq --argjson content "$content_blocks" '. + [{role: "assistant", content: $content}]')

  # Display any text blocks
  echo "$response" | jq -r '.content[] | select(.type == "text") | .text' 2>/dev/null

  if [[ "$stop_reason" == "tool_use" ]]; then
    local tool_results="[]"
    local tool_uses
    tool_uses=$(echo "$response" | jq -c '.content[] | select(.type == "tool_use")')

    while IFS= read -r tool_call; do
      [[ -z "$tool_call" ]] && continue
      local tool_id tool_command tool_output tool_exit
      tool_id=$(echo "$tool_call" | jq -r '.id')
      tool_command=$(echo "$tool_call" | jq -r '.input.command')

      echo -e "\n\033[33m$ $tool_command\033[0m" >&2
      set +e
      tool_output=$(bash -c "$tool_command" 2>&1)
      tool_exit=$?
      set -e

      # Truncate if too long
      if [[ ${#tool_output} -gt 10000 ]]; then
        tool_output="${tool_output:0:10000}... [truncated]"
      fi

      echo "$tool_output" >&2

      tool_results=$(echo "$tool_results" | jq \
        --arg id "$tool_id" \
        --arg output "$tool_output" \
        --argjson is_error "$([ $tool_exit -ne 0 ] && echo true || echo false)" \
        '. + [{type: "tool_result", tool_use_id: $id, content: $output, is_error: $is_error}]')
    done <<< "$tool_uses"

    # Append tool results as user message
    MESSAGES=$(echo "$MESSAGES" | jq --argjson results "$tool_results" '. + [{role: "user", content: $results}]')

    # Continue the conversation
    local next_response
    next_response=$(call_api)
    process_response "$next_response"
  fi
}

# --- Interactive REPL ---
echo "Claude REPL with shell tool. Type 'exit' to quit, '--self-send [file]' to evolve."
echo "---"

while true; do
  printf "\n\033[1;32myou>\033[0m " >&2
  IFS= read -r user_input < /dev/tty || break
  [[ -z "$user_input" ]] && continue
  [[ "$user_input" == "exit" ]] && break

  if [[ "$user_input" == --self-send* ]]; then
    outfile=$(echo "$user_input" | awk '{print $2}')
    bash "$0" --self-send "${outfile:-next.sh}"
    continue
  fi

  MESSAGES=$(echo "$MESSAGES" | jq --arg msg "$user_input" '. + [{role: "user", content: $msg}]')

  response=$(call_api)

  # Check for API errors
  if echo "$response" | jq -e '.error' >/dev/null 2>&1; then
    echo "API Error: $(echo "$response" | jq -r '.error.message')" >&2
    # Remove last user message on error
    MESSAGES=$(echo "$MESSAGES" | jq '.[:-1]')
    continue
  fi

  process_response "$response"
done

echo "Goodbye." >&2
