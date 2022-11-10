import json

n = "chi"
data = {
    "name": n * 2,
}
print(json.dumps(data))
