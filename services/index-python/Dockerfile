FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

COPY services/index-python/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/index-python/ .

CMD ["python", "-u", "main.py"]

