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
    results = rdb.xread({"llm_requests": "$"}, block=0, count=1)

    for stream, messages in results:
        for msg_id, data in messages:
            message = data["message"]
            request_id = data["request_id"]
            print(f"Received: {message}")

            response = client.chat.completions.create(
                model="llama3.2",
                messages=[{"role": "user", "content": message}],
                max_tokens=200,
            )

            reply = response.choices[0].message.content
            print(f"Reply: {reply}")

            # publish the message to the channel
            rdb.publish("response:" + request_id, reply)
