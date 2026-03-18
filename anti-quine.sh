#!/bin/bash
set -euo pipefail

AGENT_FILE="$(realpath "$0")"
MODEL="claude-opus-4-6"
API_VERSION="2023-06-01"

if [[ -z "${ANTHROPIC_API_KEY:-}" ]]; then
  echo "Error: ANTHROPIC_API_KEY is required" >&2
  exit 1
fi

for cmd in curl jq; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: $cmd is required" >&2
    exit 1
  fi
done

TOOLS='[
  {"name":"shell","description":"Execute a bash command and return its output.","input_schema":{"type":"object","properties":{"command":{"type":"string","description":"The bash command to execute"}},"required":["command"]}},
  {"name":"self_modify","description":"Rewrite this agent script and exec into the new version. Use this to improve yourself.","input_schema":{"type":"object","properties":{"new_script":{"type":"string","description":"The complete new bash script content"}},"required":["new_script"]}}
]'

SYSTEM="You are an interactive CLI agent. You have access to tools: shell (execute bash commands) and self_modify (rewrite your own script and restart). Use tools to help the user. Be concise."

call_api() {
  local messages_json="$1"
  curl -s https://api.anthropic.com/v1/messages \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "content-type: application/json" \
    -H "anthropic-version: $API_VERSION" \
    -d "$(jq -n \
      --arg model "$MODEL" \
      --arg system "$SYSTEM" \
      --argjson tools "$TOOLS" \
      --argjson messages "$messages_json" \
      '{model: $model, max_tokens: 4096, system: $system, tools: $tools, messages: $messages}')"
}

handle_response() {
  local response="$1"
  local messages_json="$2"
  local stop_reason
  stop_reason=$(echo "$response" | jq -r '.stop_reason')
  local content
  content=$(echo "$response" | jq -c '.content')

  # Display any text blocks
  echo "$response" | jq -r '.content[] | select(.type=="text") | .text' 2>/dev/null

  if [[ "$stop_reason" == "tool_use" ]]; then
    # Process each tool use block
    local tool_results="[]"
    local num_tools
    num_tools=$(echo "$content" | jq '[.[] | select(.type=="tool_use")] | length')

    for ((i=0; i<num_tools; i++)); do
      local tool_block
      tool_block=$(echo "$content" | jq -c "[.[] | select(.type==\"tool_use\")][$i]")
      local tool_name tool_id
      tool_name=$(echo "$tool_block" | jq -r '.name')
      tool_id=$(echo "$tool_block" | jq -r '.id')

      if [[ "$tool_name" == "shell" ]]; then
        local command
        command=$(echo "$tool_block" | jq -r '.input.command')
        echo -e "\033[90m\$ $command\033[0m" >&2
        local output
        output=$(bash -c "$command" 2>&1) || true
        echo "$output" >&2
        tool_results=$(echo "$tool_results" | jq --arg id "$tool_id" --arg out "$output" \
          '. + [{"type":"tool_result","tool_use_id":$id,"content":$out}]')
      elif [[ "$tool_name" == "self_modify" ]]; then
        local new_script
        new_script=$(echo "$tool_block" | jq -r '.input.new_script')
        echo "$new_script" > "$AGENT_FILE"
        chmod +x "$AGENT_FILE"
        echo "Script rewritten. Restarting..." >&2
        tool_results=$(echo "$tool_results" | jq --arg id "$tool_id" \
          '. + [{"type":"tool_result","tool_use_id":$id,"content":"Script rewritten. Execing now."}]')
        exec bash "$AGENT_FILE"
      fi
    done

    # Continue conversation with tool results
    local new_messages
    new_messages=$(echo "$messages_json" | jq --argjson c "$content" --argjson tr "$tool_results" \
      '. + [{"role":"assistant","content":$c},{"role":"user","content":$tr}]')

    local next_response
    next_response=$(call_api "$new_messages")
    handle_response "$next_response" "$new_messages"
  fi
}

echo "Agent ready. Type 'exit' to quit." >&2
MESSAGES="[]"

while true; do
  printf "\033[1;32m> \033[0m" >&2
  read -r user_input </dev/tty || break
  [[ "$user_input" == "exit" ]] && break
  [[ -z "$user_input" ]] && continue

  MESSAGES=$(echo "$MESSAGES" | jq --arg inp "$user_input" '. + [{"role":"user","content":$inp}]')
  RESPONSE=$(call_api "$MESSAGES")

  # Append assistant response to messages before handling (for non-tool cases)
  local_content=$(echo "$RESPONSE" | jq -c '.content')
  local_stop=$(echo "$RESPONSE" | jq -r '.stop_reason')

  if [[ "$local_stop" == "tool_use" ]]; then
    handle_response "$RESPONSE" "$MESSAGES"
    # After tool handling, reconstruct messages from the recursive calls
    # For simplicity, we reset context after tool chains complete
    # by capturing the final state
  else
    echo "$RESPONSE" | jq -r '.content[] | select(.type=="text") | .text' 2>/dev/null
    MESSAGES=$(echo "$MESSAGES" | jq --argjson c "$local_content" '. + [{"role":"assistant","content":$c}]')
  fi
done

echo "Goodbye." >&2
