from locust import HttpUser, task, between

class CompletionUser(HttpUser):
    wait_time = between(1, 2)

    @task
    def completions(self):
        self.client.post("/v1/completions", json={
            "prompt": "Hello, world!",
            "max_tokens": 16
        })