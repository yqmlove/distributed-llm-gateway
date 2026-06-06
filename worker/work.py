import os
import redis
from openai import OpenAI
from dotenv import load_dotenv

load_dotenv()

client = OpenAI(
    api_key="ollama",
    base_url="http://localhost:11434/v1"
)
rdb = redis.Redis(host="localhost", port=6379, decode_responses=True)

print("Worker started, waiting for messages...")

while True:
    # Block until a message arrives in the stream
    results = rdb.xread({"llm_requests": "$"}, block=0, count=1)

    for stream, messages in results:
        for msg_id, data in messages:
            message = data["message"]
            request_id = data["request_id"]
            print(f"[{request_id}] received: {message}")

            # Enable streaming so tokens are returned one by one
            stream_response = client.chat.completions.create(
                model="llama3.2",
                messages=[{"role": "user", "content": message}],
                max_tokens=200,
                stream=True,
            )

            # Publish each token to the channel as it arrives
            token_count = 0
            for chunk in stream_response:
                token = chunk.choices[0].delta.content
                if token:
                    token_count += 1
                    rdb.publish("response:" + request_id, token)

            # Send the end marker so the Gateway knows the stream is finished
            rdb.publish("response:" + request_id, "[DONE]")
            print(f"[{request_id}] stream complete, tokens sent: {token_count}")
