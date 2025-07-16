from locust import HttpUser, task, between

class MyUser(HttpUser):
    wait_time = between(0, 0)

    @task
    def chat_completion(self):
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer changeme"
        }

        payload = {
            "model": "gpt-3.5-turbo",
            "messages": [
                {"role": "system", "content": "You are a chat bot."},
                {"role": "user", "content": "Hello, how are you?"}
            ]
        }

        self.client.post("/v1/chat/completions", json=payload, headers=headers)
