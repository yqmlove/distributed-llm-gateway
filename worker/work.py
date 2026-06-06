import structlog
import redis
from openai import OpenAI
from dotenv import load_dotenv

load_dotenv()

# Configure structlog to output JSON
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ]
)
log = structlog.get_logger()

# Point to local Ollama instead of OpenAI
client = OpenAI(
    api_key="ollama",
    base_url="http://localhost:11434/v1"
)
rdb = redis.Redis(host="localhost", port=6379, decode_responses=True)

STREAM = "llm_requests"
GROUP  = "workers"
WORKER = "worker-1"

# Create consumer group if it does not exist
# "0" means the group will read from the very beginning of the stream
try:
    rdb.xgroup_create(STREAM, GROUP, id="$", mkstream=True)
    log.info("consumer group created", group=GROUP)
except Exception:
    log.info("consumer group already exists", group=GROUP)

log.info("worker started", stream=STREAM, group=GROUP)

while True:
    # Read one message assigned to this worker; block until one arrives
    results = rdb.xreadgroup(GROUP, WORKER, {STREAM: ">"}, count=1, block=0)

    if not results:
        continue

    for stream, messages in results:
        for msg_id, data in messages:
            message    = data["message"]
            request_id = data["request_id"]

            log.info("request received",
                     request_id=request_id,
                     message=message)

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
            log.info("stream complete",
                     request_id=request_id,
                     tokens_sent=token_count)

            # Acknowledge the message so Redis removes it from the pending list
            rdb.xack(STREAM, GROUP, msg_id)
