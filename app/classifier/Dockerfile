FROM python:3.10-slim

# --------------------
# Environment
# --------------------
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PYTHONPATH=/app

# --------------------
# System dependencies (Torch needs these)
# --------------------
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libstdc++6 \
    libgomp1 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# --------------------
# Workdir
# --------------------
WORKDIR /app

# --------------------
# Install Python deps first (better cache)
# --------------------
COPY configs/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# --------------------
# Copy application code
# --------------------
COPY api ./api
COPY internal ./internal
COPY cmd ./cmd

# --------------------
# Copy model artifacts
# --------------------
COPY ulc_model_weights.pth .
COPY ulc_vocab.pkl .
COPY ulc_model_config.json .

# --------------------
# Expose gRPC port
# --------------------
EXPOSE 9005

# --------------------
# Run classifier service
# --------------------
CMD ["python", "cmd/classifier/main.py"]
