#!/usr/bin/env python3
import http.server
import socketserver
import time
import random

class RetryTestHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        # Simulate different failure scenarios
        scenario = random.choice([
            'timeout',      # Simulate slow response
            'server_error', # Return 500
            'success',      # Return 200
            'client_error'  # Return 400 (should not retry)
        ])
        
        print(f"Test server handling request with scenario: {scenario}")
        
        if scenario == 'timeout':
            print("Simulating timeout (sleeping 15 seconds)...")
            time.sleep(15)  # Force timeout if client timeout < 15s
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'{"delayed": "response"}')
            
        elif scenario == 'server_error':
            print("Simulating server error (500)")
            self.send_response(500)
            self.end_headers()
            self.wfile.write(b'{"error": "Internal server error"}')
            
        elif scenario == 'client_error':
            print("Simulating client error (400) - should NOT retry")
            self.send_response(400)
            self.end_headers()
            self.wfile.write(b'{"error": "Bad request"}')
            
        else:  # success
            print("Simulating success (200)")
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"message": "Success after retries!"}')

if __name__ == "__main__":
    PORT = 8999
    with socketserver.TCPServer(("", PORT), RetryTestHandler) as httpd:
        print(f"Test retry server running on port {PORT}")
        print(f"Point your model API endpoint to: http://localhost:{PORT}")
        httpd.serve_forever()