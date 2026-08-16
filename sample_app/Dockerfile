FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir fastapi uvicorn requests

COPY main.py /app/main.py

EXPOSE 8080

CMD ["python", "main.py"]
