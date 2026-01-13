FROM python:3.10-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
# This ensures "import internal" works from anywhere
ENV PYTHONPATH=/app

RUN apt-get update && apt-get install -y \
    gcc g++ libstdc++6 libgomp1 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 1. Install deps
COPY app/classifier/configs/requirements.txt ./configs/requirements.txt
RUN pip install --no-cache-dir -r ./configs/requirements.txt

# 2. Copy shared API
COPY api ./api

# 3. Copy application code (Preserving the full path)
COPY app/classifier/internal ./internal
COPY app/classifier/cmd ./cmd

# 4. Copy model artifacts INTO the classifier folder
# This makes os.path.join(BASE_DIR, "file") much easier to manage
COPY app/classifier/ulc_model_weights.pth ./
COPY app/classifier/ulc_vocab.pkl ./
COPY app/classifier/ulc_model_config.json ./

EXPOSE 9005

# 5. Fixed path to main.py
CMD ["python", "cmd/classifier/main.py"]