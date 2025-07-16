FROM python:3.11-slim

WORKDIR /mnt/locust

RUN pip install --no-cache-dir locust

CMD ["locust", "-f", "locustfile.py", "--processes", "16"]
