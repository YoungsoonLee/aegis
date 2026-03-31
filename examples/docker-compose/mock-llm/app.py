"""
Mock LLM server — returns OpenAI-compatible chat completion responses.
Echoes the received message so users can see Aegis guards in action
(e.g. PII masking is visible because the mock echoes the masked text).
"""

import time
import uuid
from flask import Flask, request, jsonify

app = Flask(__name__)


@app.route("/v1/chat/completions", methods=["POST"])
def chat_completions():
    data = request.get_json(silent=True) or {}
    messages = data.get("messages", [])

    user_message = ""
    for msg in reversed(messages):
        if msg.get("role") == "user":
            user_message = msg.get("content", "")
            break

    reply = (
        f"You said: \"{user_message}\"\n\n"
        "This is a mock LLM response. "
        "If Aegis is working, any PII in your original message "
        "was masked before reaching me."
    )

    return jsonify(
        {
            "id": f"chatcmpl-mock-{uuid.uuid4().hex[:8]}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": data.get("model", "mock-gpt-4"),
            "choices": [
                {
                    "index": 0,
                    "message": {"role": "assistant", "content": reply},
                    "finish_reason": "stop",
                }
            ],
            "usage": {
                "prompt_tokens": len(user_message.split()) * 2,
                "completion_tokens": len(reply.split()) * 2,
                "total_tokens": (len(user_message.split()) + len(reply.split())) * 2,
            },
        }
    )


@app.route("/v1/models", methods=["GET"])
def list_models():
    return jsonify(
        {
            "object": "list",
            "data": [
                {"id": "mock-gpt-4", "object": "model", "owned_by": "aegis-demo"}
            ],
        }
    )


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=4000)
