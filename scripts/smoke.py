#!/usr/bin/env python3
"""Exercise a disposable image and verify persistence across container replacement."""
import json
import subprocess
import sys
import time
import urllib.request
import uuid

image = sys.argv[1] if len(sys.argv) > 1 else 'taskmate:test'
volume = 'taskmate-smoke-' + uuid.uuid4().hex[:12]
containers = []

def docker(*args):
    return subprocess.check_output(['docker', *args], text=True).strip()

def launch():
    cid = docker('run', '-d', '-p', '127.0.0.1::8080', '-v', volume + ':/data', '-e', 'DEMO_MESSAGE=Smoke test deployment', image)
    containers.append(cid)
    details = json.loads(docker('inspect', cid))[0]
    port = details['NetworkSettings']['Ports']['8080/tcp'][0]['HostPort']
    base = 'http://127.0.0.1:' + port
    for attempt in range(50):
        try:
            with urllib.request.urlopen(base + '/healthz', timeout=2) as response:
                assert response.status == 200 and response.read() == b'ok\n'
            return cid, base
        except OSError:
            if attempt == 49:
                raise
            time.sleep(.2)

def request(base, path, method='GET', body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(base + path, data=data, method=method, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=5) as response:
        if response.status == 204:
            return None
        return json.load(response)

try:
    docker('volume', 'create', volume)
    cid, base = launch()
    with urllib.request.urlopen(base, timeout=5) as response:
        assert b'<div id="app">' in response.read()
    assert request(base, '/api/info')['message'] == 'Smoke test deployment'
    task = request(base, '/api/tasks', 'POST', {'title': 'Persistent task', 'description': 'Survives replacement', 'status': 'todo', 'priority': 'high'})
    docker('exec', cid, '/demo', '-healthcheck')
    docker('stop', '-t', '12', cid)
    docker('rm', cid)
    containers.remove(cid)
    cid, base = launch()
    page = request(base, '/api/tasks?limit=20')
    assert page['total'] == 1 and page['tasks'][0]['id'] == task['id'], page
    updated = request(base, '/api/tasks/' + task['id'], 'PUT', {'title': 'Updated task', 'description': '', 'status': 'done', 'priority': 'low'})
    assert updated['status'] == 'done'
    assert request(base, '/api/tasks?status=done')['stats']['done'] == 1
    request(base, '/api/tasks/' + task['id'], 'DELETE')
    assert request(base, '/api/tasks')['total'] == 0
    docker('exec', cid, '/demo', '-healthcheck')
    print('PASS: frontend, API CRUD, environment, container replacement persistence and healthcheck')
finally:
    for cid in containers:
        subprocess.run(['docker', 'rm', '-f', cid], check=True, stdout=subprocess.DEVNULL)
    subprocess.run(['docker', 'volume', 'rm', volume], check=True, stdout=subprocess.DEVNULL)
