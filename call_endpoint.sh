#!/bin/bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello, world!"}]
  }'


 http://localhost:8080/v1/chat/completions


   curl --no-buffer  http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "stream": false,
    "messages": [
      {
        "role": "user",
        "content": "Write a very long and detailed essay on the history and future of artificial intelligence, including key developments and researchers. Minimum 3000 words."
      }
    ]
  }'


  curl --no-buffer https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $LLM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "stream": true,
    "messages": [
      {
        "role": "user",
        "content": "Write a very long and detailed essay on the history and future of artificial intelligence, including key developments and researchers. Minimum 3000 words."
      }
    ]
  }'
