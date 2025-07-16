from locust import HttpUser, task, between

class ChatUser(HttpUser):
    wait_time = between(1, 5)

    @task
    def chat_completion(self):
        headers = {
            "Authorization": "Bearer changeme",
            "Content-Type": "application/json"
        }
        self.client.post("/v1/chat/completions", json={
            "model": "gpt-3.5-turbo",
            "messages": [
                {"role": "user", "content": "Hello"}
            ]
        }, headers=headers)
