"""
Demo AI agent — a simple chat web app that talks to an LLM through Aegis.
Open http://localhost:3000 in your browser to use the chat interface.
"""

import os
import requests as http_client
from flask import Flask, render_template, request, jsonify

app = Flask(__name__)

AEGIS_URL = os.environ.get("OPENAI_API_BASE", "http://aegis:8080/v1")
API_KEY = os.environ.get("OPENAI_API_KEY", "mock-key")


@app.route("/")
def index():
    return render_template("index.html")


@app.route("/chat", methods=["POST"])
def chat():
    data = request.get_json(silent=True) or {}
    message = data.get("message", "")

    if not message.strip():
        return jsonify({"error": True, "message": "Empty message"}), 400

    try:
        resp = http_client.post(
            f"{AEGIS_URL}/chat/completions",
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {API_KEY}",
            },
            json={
                "model": "gpt-4",
                "messages": [{"role": "user", "content": message}],
            },
            timeout=30,
        )

        body = resp.json()

        if resp.status_code != 200:
            err = body.get("error", {})
            return jsonify(
                {
                    "error": True,
                    "status": resp.status_code,
                    "message": err.get("message", "Unknown error"),
                    "guard": err.get("guard", ""),
                    "type": err.get("type", ""),
                }
            )

        content = (
            body.get("choices", [{}])[0].get("message", {}).get("content", "")
        )
        return jsonify({"error": False, "content": content})

    except http_client.exceptions.ConnectionError:
        return jsonify(
            {
                "error": True,
                "status": 503,
                "message": "Cannot connect to Aegis. Is it running?",
            }
        )
    except Exception as e:
        return jsonify({"error": True, "status": 500, "message": str(e)})


@app.route("/health")
def health():
    return jsonify({"status": "ok"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=3000, debug=False)
