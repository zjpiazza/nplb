# Use Python 3.12 slim as base image
FROM python:3.12-slim

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIPENV_VENV_IN_PROJECT=1 \
    PIPENV_DONT_LOAD_ENV=1

# Set work directory
WORKDIR /app

# Install system dependencies
# RUN apt-get update && apt-get install -y \
#     build-essential \
#     curl \
#     && rm -rf /var/lib/apt/lists/*

# Install pipenv
RUN pip install --no-cache-dir pipenv

# Copy Pipfile and Pipfile.lock
COPY Pipfile Pipfile.lock ./

# Install dependencies
RUN pipenv install --system --deploy

# Copy project files
COPY . .

# Expose port
EXPOSE 8000

# Command to run the application
CMD ["pipenv", "run", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]