import os
import socket
from fastapi import FastAPI
import uvicorn

app = FastAPI()


@app.get("/")
def read_root():
    hostname = socket.gethostname()
    ip_address = socket.gethostbyname(hostname)
    instance_id = os.environ.get("INSTANCE_ID", "default")
    return {
        "message": "Hello from Dynamic Docker Instance!",
        "hostname": hostname,
        "ip": ip_address,
        "instance_id": instance_id,
    }


@app.get("/health")
def health_check():
    return {"status": "ok"}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
